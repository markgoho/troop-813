package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

// getMe calls GET /api/me and returns the status and, on 200, the body; on
// an error status, the error code.
func getMe(t *testing.T, env *testenv.Env, token string) (int, contract.Me, contract.ErrorCode) {
	t.Helper()
	resp := env.Request(t, http.MethodGet, "/api/me", token, nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		var e contract.ErrorBody
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			t.Fatalf("decode error body: %v", err)
		}
		return resp.StatusCode, contract.Me{}, e.Code
	}
	var me contract.Me
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp.StatusCode, me, ""
}

func TestFirstSignInCreatesAParentUser(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	token := env.GoogleToken(t, "u-riley", "riley.fictional@example.com", "Riley Fictional")

	status, me, _ := getMe(t, env, token)

	if status != http.StatusOK || me != (contract.Me{UID: "u-riley", Role: contract.RoleParent}) {
		t.Fatalf("GET /api/me = %d %+v, want 200 {u-riley parent}", status, me)
	}
	got, ok := env.User(t, "u-riley")
	if !ok {
		t.Fatal("users/u-riley was not created")
	}
	want := contract.User{
		Email:       "riley.fictional@example.com",
		DisplayName: "Riley Fictional",
		Role:        contract.RoleParent,
		CreatedAt:   testenv.DefaultNow,
	}
	if !got.CreatedAt.Equal(want.CreatedAt) || got.Email != want.Email || got.DisplayName != want.DisplayName || got.Role != want.Role {
		t.Fatalf("users/u-riley = %+v, want %+v", got, want)
	}
}

func TestLaterSignInsKeepTheRole(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	token := env.GoogleToken(t, "u-jordan", "jordan.fictional@example.com", "Jordan Fictional")
	if status, _, _ := getMe(t, env, token); status != http.StatusOK {
		t.Fatalf("first sign-in: status %d", status)
	}
	env.SetRole(t, "u-jordan", contract.RoleLeader)
	env.Clock.Advance(7 * 24 * 60 * 60 * 1e9)

	_, me, _ := getMe(t, env, token)

	if me.Role != contract.RoleLeader {
		t.Fatalf("role after a later sign-in = %q, want leader", me.Role)
	}
	if u, _ := env.User(t, "u-jordan"); u.Role != contract.RoleLeader {
		t.Fatalf("stored role = %q, want leader (not overwritten)", u.Role)
	}
}

func TestTokensResolveToTheirCaller(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	cases := []struct {
		name  string
		token string
		want  contract.Me
	}{
		{"leader", env.RoleToken(t, "u-leader", contract.RoleLeader), contract.Me{UID: "u-leader", Role: contract.RoleLeader}},
		{"admin", env.RoleToken(t, "u-admin", contract.RoleAdmin), contract.Me{UID: "u-admin", Role: contract.RoleAdmin}},
		{"parent", env.RoleToken(t, "u-parent", contract.RoleParent), contract.Me{UID: "u-parent", Role: contract.RoleParent}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			status, me, _ := getMe(t, env, c.token)
			if status != http.StatusOK || me != c.want {
				t.Fatalf("GET /api/me = %d %+v, want 200 %+v", status, me, c.want)
			}
		})
	}
}

func TestRequestsWithoutAValidTokenAreUnauthenticated(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	// A Kiosk token is a real ID token, but the Kiosk caller arrives with the
	// Start Meeting ticket; until then it must not pass as a Parent.
	kiosk := env.IDToken(t, "kiosk-1", map[string]any{"role": "kiosk", "kioskSessionId": "s1", "meetingId": "2026-11-09"})
	for name, token := range map[string]string{
		"anonymous": "",
		"garbage":   "not-a-token",
		"kiosk":     kiosk,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status, _, code := getMe(t, env, token)
			if status != http.StatusUnauthorized || code != contract.ErrorCodeUnauthenticated {
				t.Fatalf("GET /api/me = %d %q, want 401 unauthenticated", status, code)
			}
		})
	}
	if _, ok := env.User(t, "kiosk-1"); ok {
		t.Fatal("a Kiosk token created a users doc")
	}
}

func TestARoleChangeTakesEffectOnTheNextRequest(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	token := env.RoleToken(t, "u-casey", contract.RoleLeader)
	if _, me, _ := getMe(t, env, token); me.Role != contract.RoleLeader {
		t.Fatalf("before: role %q, want leader", me.Role)
	}

	env.SetRole(t, "u-casey", contract.RoleParent)

	if _, me, _ := getMe(t, env, token); me.Role != contract.RoleParent {
		t.Fatalf("after demotion: role %q, want parent", me.Role)
	}
}

func TestSignInLogsNoNamesOrEmails(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	token := env.GoogleToken(t, "u-morgan", "morgan.fictional@example.com", "Morgan Fictional")
	getMe(t, env, token)

	if logs := env.Logs(); strings.Contains(logs, "Morgan") || strings.Contains(logs, "morgan.fictional") {
		t.Fatalf("logs carry a name or email:\n%s", logs)
	}
}
