package server_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

// createPastMeeting calls POST /api/meetings.
func createPastMeeting(t *testing.T, env *testenv.Env, token, date string) (int, string, contract.ErrorCode) {
	t.Helper()
	resp := env.Request(t, http.MethodPost, "/api/meetings", token, contract.PastMeetingRequest{Date: date})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		return resp.StatusCode, "", errorCode(t, resp)
	}
	var out contract.PastMeetingCreated
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp.StatusCode, out.ID, ""
}

// deleteMeeting calls DELETE /api/meetings/{date}.
func deleteMeeting(t *testing.T, env *testenv.Env, token, date string) (int, contract.ErrorCode) {
	t.Helper()
	resp := env.Request(t, http.MethodDelete, "/api/meetings/"+date, token, nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		return resp.StatusCode, errorCode(t, resp)
	}
	return resp.StatusCode, ""
}

func meetingExists(t *testing.T, env *testenv.Env, date string) bool {
	t.Helper()
	snap, err := env.Firestore.Collection("meetings").Doc(date).Get(t.Context())
	return err == nil && snap.Exists()
}

func TestALeaderBackfillsAPastMeeting(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t) // clock: Mon 2026-11-09 19:00
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)

	status, id, code := createPastMeeting(t, env, leader, "2026-11-02")

	if status != http.StatusCreated || id != "2026-11-02" {
		t.Fatalf("create = %d %q %q, want 201 2026-11-02", status, id, code)
	}
	snap, err := env.Firestore.Collection("meetings").Doc("2026-11-02").Get(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	raw := snap.Data() // HQ iterates these, so they must be stored, not null
	if raw["startedBy"] != "u-leader" {
		t.Errorf("startedBy = %v, want u-leader", raw["startedBy"])
	}
	for _, f := range []string{"handoffOrder", "notFound"} {
		if v, ok := raw[f].([]any); !ok || len(v) != 0 {
			t.Errorf("%s = %#v, want an empty array", f, raw[f])
		}
	}
	if v, ok := raw["checkedIn"].(map[string]any); !ok || len(v) != 0 {
		t.Errorf("checkedIn = %#v, want an empty map", raw["checkedIn"])
	}
	// The backfilled Meeting is a real Meeting: a Leader can check a Scout in.
	scouts := seedScouts(t, env, true, "Avery")
	if status, attID, code := leaderCheckIn(t, env, leader, "2026-11-02", scouts[0]); status != http.StatusOK || attID == "" {
		t.Errorf("check-in on backfilled Meeting = %d %q", status, code)
	}
}

func TestPastMeetingDateRules(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	if status, _, code := createPastMeeting(t, env, leader, "2026-11-02"); status != http.StatusCreated {
		t.Fatalf("setup create = %d %q", status, code)
	}

	for _, tc := range []struct {
		name, date string
		status     int
		code       contract.ErrorCode
	}{
		{"today", "2026-11-09", http.StatusBadRequest, contract.ErrorCodeBadRequest},
		{"future", "2026-11-16", http.StatusBadRequest, contract.ErrorCodeBadRequest},
		{"already exists", "2026-11-02", http.StatusConflict, contract.ErrorCodeMeetingExists},
		{"not a date", "last monday", http.StatusBadRequest, contract.ErrorCodeBadRequest},
		{"impossible date", "2026-02-30", http.StatusBadRequest, contract.ErrorCodeBadRequest},
		{"empty", "", http.StatusBadRequest, contract.ErrorCodeBadRequest},
	} {
		if status, _, code := createPastMeeting(t, env, leader, tc.date); status != tc.status || code != tc.code {
			t.Errorf("%s: create %q = %d %q, want %d %q", tc.name, tc.date, status, code, tc.status, tc.code)
		}
	}
	for _, d := range []string{"2026-11-09", "2026-11-16"} {
		if meetingExists(t, env, d) {
			t.Errorf("meetings/%s was created", d)
		}
	}
}

func TestTodaysExistingMeetingIsRefusedAsNotPast(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	_, _ = unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)

	if status, _, code := createPastMeeting(t, env, leader, "2026-11-09"); status != http.StatusBadRequest {
		t.Errorf("create today = %d %q, want 400", status, code)
	}
}

func TestAMeetingWithNobodyPresentCanBeDeleted(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	scouts := seedScouts(t, env, true, "Avery")
	_, _, _ = createPastMeeting(t, env, leader, "2026-11-02")
	_, attID, _ := leaderCheckIn(t, env, leader, "2026-11-02", scouts[0])

	if status, code := deleteMeeting(t, env, leader, "2026-11-02"); status != http.StatusConflict || code != contract.ErrorCodeInUse {
		t.Fatalf("delete with a Scout present = %d %q, want 409 in_use", status, code)
	}
	if !meetingExists(t, env, "2026-11-02") {
		t.Fatal("Meeting vanished despite 409")
	}

	if status, code := undoCheckIn(t, env, leader, "2026-11-02", attID); status != http.StatusNoContent {
		t.Fatalf("undo = %d %q", status, code)
	}
	if status, code := deleteMeeting(t, env, leader, "2026-11-02"); status != http.StatusNoContent {
		t.Fatalf("delete after undo = %d %q, want 204", status, code)
	}
	if meetingExists(t, env, "2026-11-02") {
		t.Error("Meeting doc still exists")
	}
	// The undone record goes with it, so it can't pin the Scout as in-use.
	if n := len(attendanceDocs(t, env, "2026-11-02")); n != 0 {
		t.Errorf("%d attendance docs left behind, want 0", n)
	}
	if status, code := deleteMeeting(t, env, leader, "2026-11-02"); status != http.StatusNotFound || code != contract.ErrorCodeNotFound {
		t.Errorf("delete again = %d %q, want 404 not_found", status, code)
	}
}
