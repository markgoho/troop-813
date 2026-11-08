package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

// addScout calls POST /api/scouts and returns the status, the new ID on 201,
// and the error code otherwise.
func addScout(t *testing.T, env *testenv.Env, token string, body any) (int, string, contract.ErrorCode) {
	t.Helper()
	resp := env.Request(t, http.MethodPost, "/api/scouts", token, body)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		var e contract.ErrorBody
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			t.Fatalf("decode error body (status %d): %v", resp.StatusCode, err)
		}
		return resp.StatusCode, "", e.Code
	}
	var got contract.AddScoutResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp.StatusCode, got.ID, ""
}

func scoutDoc(t *testing.T, env *testenv.Env, id string) contract.Scout {
	t.Helper()
	snap, err := env.Firestore.Collection("scouts").Doc(id).Get(context.Background())
	if err != nil {
		t.Fatalf("get scouts/%s: %v", id, err)
	}
	var s contract.Scout
	if err := snap.DataTo(&s); err != nil {
		t.Fatalf("decode scouts/%s: %v", id, err)
	}
	return s
}

func TestALeaderAddsAScout(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)

	status, id, code := addScout(t, env, leader, contract.AddScoutRequest{FirstName: " Avery ", LastName: "Quill  "})

	if status != http.StatusCreated || id == "" {
		t.Fatalf("POST /api/scouts = %d %q (id %q), want 201 with an id", status, code, id)
	}
	got := scoutDoc(t, env, id)
	if got.FirstName != "Avery" || got.LastName != "Quill" || !got.Active || !got.CreatedAt.Equal(testenv.DefaultNow) {
		t.Fatalf("scouts/%s = %+v, want Avery Quill, active, created at %v", id, got, testenv.DefaultNow)
	}
}

func TestAnExactDuplicateOfAnActiveScoutIsBlocked(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	if status, _, code := addScout(t, env, leader, contract.AddScoutRequest{FirstName: "Avery", LastName: "Quill"}); status != http.StatusCreated {
		t.Fatalf("first add: %d %q", status, code)
	}

	for name, req := range map[string]contract.AddScoutRequest{
		"same":       {FirstName: "Avery", LastName: "Quill"},
		"case":       {FirstName: "AVERY", LastName: "quill"},
		"whitespace": {FirstName: "  avery", LastName: "Quill "},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status, _, code := addScout(t, env, leader, req)
			if status != http.StatusConflict || code != contract.ErrorCodeDuplicateScout {
				t.Fatalf("POST %+v = %d %q, want 409 duplicate_scout", req, status, code)
			}
		})
	}
}

func TestAScoutSharingOnlyOneNameIsNotADuplicate(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	addScout(t, env, leader, contract.AddScoutRequest{FirstName: "Avery", LastName: "Quill"})

	for _, req := range []contract.AddScoutRequest{
		{FirstName: "Avery", LastName: "Marsh"},
		{FirstName: "Rowan", LastName: "Quill"},
	} {
		if status, _, code := addScout(t, env, leader, req); status != http.StatusCreated {
			t.Fatalf("POST %+v = %d %q, want 201", req, status, code)
		}
	}
}

func TestAddScoutNeedsBothNames(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)

	for name, body := range map[string]any{
		"blank first": contract.AddScoutRequest{FirstName: "  ", LastName: "Quill"},
		"blank last":  contract.AddScoutRequest{FirstName: "Avery", LastName: ""},
		"not json":    "Avery Quill",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status, _, code := addScout(t, env, leader, body)
			if status != http.StatusBadRequest || code != contract.ErrorCodeBadRequest {
				t.Fatalf("POST %v = %d %q, want 400 bad_request", body, status, code)
			}
		})
	}
}

func TestOnlyLeadersAddScouts(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	req := contract.AddScoutRequest{FirstName: "Avery", LastName: "Quill"}
	cases := []struct {
		name   string
		token  string
		status int
	}{
		{"anonymous", "", http.StatusUnauthorized},
		{"parent", env.RoleToken(t, "u-parent", contract.RoleParent), http.StatusForbidden},
		{"admin", env.RoleToken(t, "u-admin", contract.RoleAdmin), http.StatusCreated},
	}
	for _, c := range cases {
		if status, _, _ := addScout(t, env, c.token, req); status != c.status {
			t.Errorf("%s: status %d, want %d", c.name, status, c.status)
		}
	}
	snaps, err := env.Firestore.Collection("scouts").Documents(context.Background()).GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 1 {
		t.Fatalf("%d scouts stored, want 1 (the Admin's)", len(snaps))
	}
}

func TestAddScoutLogsTheIDNotTheName(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	_, id, _ := addScout(t, env, leader, contract.AddScoutRequest{FirstName: "Peregrine", LastName: "Thistlewood"})

	logs := env.Logs()
	if strings.Contains(logs, "Peregrine") || strings.Contains(logs, "Thistlewood") {
		t.Fatalf("logs carry the Scout's name:\n%s", logs)
	}
	if !strings.Contains(logs, id) {
		t.Fatalf("logs don't carry scoutId %s:\n%s", id, logs)
	}
}
