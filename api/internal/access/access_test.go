package access_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

func listUsers(t *testing.T, env *testenv.Env, token string) (int, contract.UserList) {
	t.Helper()
	resp := env.Request(t, http.MethodGet, "/api/users", token, nil)
	defer func() { _ = resp.Body.Close() }()
	var l contract.UserList
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&l); err != nil {
			t.Fatal(err)
		}
	}
	return resp.StatusCode, l
}

func setRole(t *testing.T, env *testenv.Env, token, uid string, role contract.Role) (int, contract.ErrorCode) {
	t.Helper()
	resp := env.Request(t, http.MethodPut, "/api/users/"+uid+"/role", token, contract.SetRole{Role: role})
	defer func() { _ = resp.Body.Close() }()
	var e contract.ErrorBody
	if resp.StatusCode != http.StatusNoContent {
		_ = json.NewDecoder(resp.Body).Decode(&e)
	}
	return resp.StatusCode, e.Code
}

func roleOf(t *testing.T, env *testenv.Env, token string) contract.Role {
	t.Helper()
	resp := env.Request(t, http.MethodGet, "/api/me", token, nil)
	defer func() { _ = resp.Body.Close() }()
	var me contract.Me
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		t.Fatal(err)
	}
	return me.Role
}

func TestAdminListsEveryoneNewestSignInFirst(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	admin := env.RoleToken(t, "u-admin", contract.RoleAdmin)
	for _, u := range []struct{ uid, email, name string }{
		{"u-avery", "avery.fictional@example.com", "Avery Fictional"},
		{"u-blake", "blake.fictional@example.com", "Blake Fictional"},
	} {
		env.Clock.Advance(time.Hour)
		resp := env.Request(t, http.MethodGet, "/api/me", env.GoogleToken(t, u.uid, u.email, u.name), nil)
		_ = resp.Body.Close()
	}

	status, l := listUsers(t, env, admin)

	if status != http.StatusOK || len(l.Users) != 3 {
		t.Fatalf("GET /api/users = %d %+v, want 200 with 3 users", status, l)
	}
	got := []string{l.Users[0].UID, l.Users[1].UID, l.Users[2].UID}
	if got[0] != "u-blake" || got[1] != "u-avery" || got[2] != "u-admin" {
		t.Fatalf("order = %v, want [u-blake u-avery u-admin]", got)
	}
	// DefaultNow is 19:00 NY (00:00Z) + 2h.
	want := contract.UserSummary{UID: "u-blake", Email: "blake.fictional@example.com", DisplayName: "Blake Fictional", Role: contract.RoleParent, CreatedAt: "2026-11-10T02:00:00Z"}
	if l.Users[0] != want {
		t.Fatalf("first row = %+v, want %+v", l.Users[0], want)
	}
}

func TestPromotionAndDemotionTakeEffectOnTheNextRequest(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	admin := env.RoleToken(t, "u-admin", contract.RoleAdmin)
	quinn := env.RoleToken(t, "u-quinn", contract.RoleParent)

	if s, _ := setRole(t, env, admin, "u-quinn", contract.RoleLeader); s != http.StatusNoContent {
		t.Fatalf("promote: %d", s)
	}
	if r := roleOf(t, env, quinn); r != contract.RoleLeader {
		t.Fatalf("after promotion: %q", r)
	}
	if s, _ := setRole(t, env, admin, "u-quinn", contract.RoleParent); s != http.StatusNoContent {
		t.Fatalf("demote: %d", s)
	}
	if r := roleOf(t, env, quinn); r != contract.RoleParent {
		t.Fatalf("after demotion: %q", r)
	}
}

func TestOnlyTheAdminManagesAccess(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	env.SetRole(t, "u-parent", contract.RoleParent)

	if s, _ := listUsers(t, env, leader); s != http.StatusForbidden {
		t.Errorf("leader list: %d, want 403", s)
	}
	if s, _ := setRole(t, env, leader, "u-parent", contract.RoleLeader); s != http.StatusForbidden {
		t.Errorf("leader set role: %d, want 403", s)
	}
	if u, _ := env.User(t, "u-parent"); u.Role != contract.RoleParent {
		t.Errorf("role changed to %q", u.Role)
	}
}

func TestSetRoleRefusals(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	admin := env.RoleToken(t, "u-admin", contract.RoleAdmin)
	env.SetRole(t, "u-parent", contract.RoleParent)
	env.SetRole(t, "u-other-admin", contract.RoleAdmin)
	cases := []struct {
		name   string
		uid    string
		role   contract.Role
		status int
		code   contract.ErrorCode
	}{
		{"own row", "u-admin", contract.RoleLeader, http.StatusConflict, contract.ErrorCodeFixedRole},
		{"another admin's row", "u-other-admin", contract.RoleParent, http.StatusConflict, contract.ErrorCodeFixedRole},
		{"never sets admin", "u-parent", contract.RoleAdmin, http.StatusBadRequest, contract.ErrorCodeBadRequest},
		{"unknown role", "u-parent", "chief", http.StatusBadRequest, contract.ErrorCodeBadRequest},
		{"no such user", "u-nobody", contract.RoleLeader, http.StatusNotFound, contract.ErrorCodeNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, code := setRole(t, env, admin, c.uid, c.role)
			if s != c.status || code != c.code {
				t.Fatalf("= %d %q, want %d %q", s, code, c.status, c.code)
			}
		})
	}
	if u, _ := env.User(t, "u-admin"); u.Role != contract.RoleAdmin {
		t.Errorf("own role changed to %q", u.Role)
	}
	if u, _ := env.User(t, "u-other-admin"); u.Role != contract.RoleAdmin {
		t.Errorf("other admin role changed to %q", u.Role)
	}
	if u, _ := env.User(t, "u-parent"); u.Role != contract.RoleParent {
		t.Errorf("u-parent role changed to %q", u.Role)
	}
	if _, ok := env.User(t, "u-nobody"); ok {
		t.Error("u-nobody doc created")
	}
}

func dismiss(t *testing.T, env *testenv.Env, token, uid string) (int, contract.ErrorCode) {
	t.Helper()
	resp := env.Request(t, http.MethodPost, "/api/users/"+uid+"/dismiss", token, nil)
	defer func() { _ = resp.Body.Close() }()
	var e contract.ErrorBody
	if resp.StatusCode != http.StatusNoContent {
		_ = json.NewDecoder(resp.Body).Decode(&e)
	}
	return resp.StatusCode, e.Code
}

func listed(l contract.UserList, uid string) bool {
	for _, u := range l.Users {
		if u.UID == uid {
			return true
		}
	}
	return false
}

func TestDismissHidesAWaitingRequestAcrossSignIns(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	admin := env.RoleToken(t, "u-admin", contract.RoleAdmin)
	quinn := env.RoleToken(t, "u-quinn", contract.RoleParent)

	if s, code := dismiss(t, env, admin, "u-quinn"); s != http.StatusNoContent {
		t.Fatalf("dismiss = %d %q, want 204", s, code)
	}
	// Signing in again keeps the doc (and the dismissal) and stays a Parent.
	if r := roleOf(t, env, quinn); r != contract.RoleParent {
		t.Fatalf("role after dismiss: %q", r)
	}
	if _, l := listUsers(t, env, admin); listed(l, "u-quinn") {
		t.Fatalf("dismissed request still listed: %+v", l)
	}
	// Promoting anyway (e.g. straight through the API) clears it, so a later
	// demotion lists them again.
	if s, _ := setRole(t, env, admin, "u-quinn", contract.RoleLeader); s != http.StatusNoContent {
		t.Fatalf("promote: %d", s)
	}
	if s, _ := setRole(t, env, admin, "u-quinn", contract.RoleParent); s != http.StatusNoContent {
		t.Fatalf("demote: %d", s)
	}
	if _, l := listUsers(t, env, admin); !listed(l, "u-quinn") {
		t.Fatalf("demoted Leader not listed: %+v", l)
	}
}

func TestDismissRefusals(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	admin := env.RoleToken(t, "u-admin", contract.RoleAdmin)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	env.SetRole(t, "u-parent", contract.RoleParent)
	cases := []struct {
		name, token, uid string
		status           int
		code             contract.ErrorCode
	}{
		{"a Leader", admin, "u-leader", http.StatusConflict, contract.ErrorCodeNotWaiting},
		{"own row", admin, "u-admin", http.StatusConflict, contract.ErrorCodeNotWaiting},
		{"no such user", admin, "u-nobody", http.StatusNotFound, contract.ErrorCodeNotFound},
		{"not the Admin", leader, "u-parent", http.StatusForbidden, contract.ErrorCodeForbidden},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, code := dismiss(t, env, c.token, c.uid)
			if s != c.status || code != c.code {
				t.Fatalf("= %d %q, want %d %q", s, code, c.status, c.code)
			}
		})
	}
	if _, l := listUsers(t, env, admin); !listed(l, "u-parent") || !listed(l, "u-leader") {
		t.Fatalf("refused dismissals hid someone: %+v", l)
	}
	if _, ok := env.User(t, "u-nobody"); ok {
		t.Error("u-nobody doc created")
	}
}
