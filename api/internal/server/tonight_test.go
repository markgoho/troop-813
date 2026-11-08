package server_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

// leaderCheckIn calls POST /api/meetings/{date}/check-ins.
func leaderCheckIn(t *testing.T, env *testenv.Env, token, date, scoutID string) (int, string, contract.ErrorCode) {
	t.Helper()
	resp := env.Request(t, http.MethodPost, "/api/meetings/"+date+"/check-ins", token, contract.LeaderCheckInRequest{ScoutID: scoutID})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, "", errorCode(t, resp)
	}
	var out contract.CheckInResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode check-in: %v", err)
	}
	return resp.StatusCode, out.AttendanceID, ""
}

// undoCheckIn calls POST /api/meetings/{date}/attendance/{id}/undo.
func undoCheckIn(t *testing.T, env *testenv.Env, token, date, attendanceID string) (int, contract.ErrorCode) {
	t.Helper()
	resp := env.Request(t, http.MethodPost, "/api/meetings/"+date+"/attendance/"+attendanceID+"/undo", token, nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		return resp.StatusCode, errorCode(t, resp)
	}
	return resp.StatusCode, ""
}

func TestALeaderChecksAScoutIn(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery", "Rowan")
	_, _ = unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	env.Clock.Advance(5 * time.Minute)

	status, id, code := leaderCheckIn(t, env, leader, "2026-11-09", scouts[0])

	if status != http.StatusOK || id == "" {
		t.Fatalf("leader check-in = %d %q, want 200 with an attendanceId", status, code)
	}
	a := attendanceDocs(t, env, "2026-11-09")[id]
	if a.ScoutID != scouts[0] || a.Method != contract.AttendanceMethodLeader || a.By != "u-leader" || !a.At.Equal(testenv.DefaultNow.Add(5*time.Minute)) || a.UndoneAt != nil {
		t.Errorf("attendance/%s = %+v, want a leader check-in by u-leader", id, a)
	}
	if m := meetingDoc(t, env, "2026-11-09"); m.CheckedIn[scouts[0]] != id || len(m.CheckedIn) != 1 {
		t.Errorf("checkedIn = %v, want only %s → %s", m.CheckedIn, scouts[0], id)
	}
	// A repeat returns the existing record.
	if status, again, _ := leaderCheckIn(t, env, leader, "2026-11-09", scouts[0]); status != http.StatusOK || again != id {
		t.Errorf("repeat = %d %q, want 200 %q", status, again, id)
	}
	if n := len(attendanceDocs(t, env, "2026-11-09")); n != 1 {
		t.Errorf("%d attendance records, want 1", n)
	}
}

func TestLeaderCheckInRejectsBadRequests(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	_, _ = unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)

	if status, _, code := leaderCheckIn(t, env, leader, "2026-11-09", "nope"); status != http.StatusNotFound || code != contract.ErrorCodeNotFound {
		t.Errorf("unknown Scout = %d %q, want 404 not_found", status, code)
	}
	if status, _, code := leaderCheckIn(t, env, leader, "2026-11-10", scouts[0]); status != http.StatusNotFound || code != contract.ErrorCodeNotFound {
		t.Errorf("no such Meeting = %d %q, want 404 not_found", status, code)
	}
	if status, _, code := leaderCheckIn(t, env, leader, "2026-11-09", ""); status != http.StatusBadRequest || code != contract.ErrorCodeBadRequest {
		t.Errorf("no scoutId = %d %q, want 400 bad_request", status, code)
	}
}

func TestAKioskTokenCannotCheckInOrUndoAsALeader(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	kiosk, _ := unlockKiosk(t, env)
	_, id, _ := checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList)

	if status, _, code := leaderCheckIn(t, env, kiosk, "2026-11-09", scouts[0]); status != http.StatusForbidden || code != contract.ErrorCodeForbidden {
		t.Errorf("kiosk leader check-in = %d %q, want 403", status, code)
	}
	if status, code := undoCheckIn(t, env, kiosk, "2026-11-09", id); status != http.StatusForbidden {
		t.Errorf("kiosk undo = %d %q, want 403", status, code)
	}
	if m := meetingDoc(t, env, "2026-11-09"); m.CheckedIn[scouts[0]] != id {
		t.Errorf("checkedIn = %v, want unchanged", m.CheckedIn)
	}
}

func TestUndoIsASoftDeleteAndFreesTheScout(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	kiosk, _ := unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	_, first, _ := checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList)
	env.Clock.Advance(3 * time.Minute)

	if status, code := undoCheckIn(t, env, leader, "2026-11-09", first); status != http.StatusNoContent {
		t.Fatalf("undo = %d %q, want 204", status, code)
	}

	a, ok := attendanceDocs(t, env, "2026-11-09")[first]
	if !ok || a.UndoneAt == nil || !a.UndoneAt.Equal(testenv.DefaultNow.Add(3*time.Minute)) || a.UndoneBy != "u-leader" {
		t.Errorf("attendance/%s = %+v (found %v), want soft-deleted by u-leader", first, a, ok)
	}
	if m := meetingDoc(t, env, "2026-11-09"); len(m.CheckedIn) != 0 {
		t.Errorf("checkedIn = %v, want empty", m.CheckedIn)
	}

	// Undoing twice changes nothing and is not an error.
	if status, code := undoCheckIn(t, env, leader, "2026-11-09", first); status != http.StatusNoContent {
		t.Errorf("second undo = %d %q, want 204", status, code)
	}

	// Checking in again makes a new record; the old one stays undone.
	status, second, code := leaderCheckIn(t, env, leader, "2026-11-09", scouts[0])
	if status != http.StatusOK || second == first {
		t.Fatalf("re-check-in = %d %q %q, want a new record", status, code, second)
	}
	docs := attendanceDocs(t, env, "2026-11-09")
	if len(docs) != 2 || docs[first].UndoneAt == nil || docs[second].UndoneAt != nil {
		t.Errorf("attendance = %+v, want old undone, new live", docs)
	}
	if m := meetingDoc(t, env, "2026-11-09"); m.CheckedIn[scouts[0]] != second {
		t.Errorf("checkedIn = %v, want → %s", m.CheckedIn, second)
	}

	// Undoing the stale record again must not unmark the current check-in.
	_, _ = undoCheckIn(t, env, leader, "2026-11-09", first)
	if m := meetingDoc(t, env, "2026-11-09"); m.CheckedIn[scouts[0]] != second {
		t.Errorf("checkedIn = %v after re-undoing the old record, want → %s", m.CheckedIn, second)
	}
}

func TestUndoOfAnUnknownAttendanceIsNotFound(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	_, _ = unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)

	if status, code := undoCheckIn(t, env, leader, "2026-11-09", "nope"); status != http.StatusNotFound || code != contract.ErrorCodeNotFound {
		t.Errorf("undo unknown = %d %q, want 404 not_found", status, code)
	}
	if status, code := undoCheckIn(t, env, leader, "2026-11-10", "nope"); status != http.StatusNotFound || code != contract.ErrorCodeNotFound {
		t.Errorf("undo on no Meeting = %d %q, want 404 not_found", status, code)
	}
}

func TestLeadersListLeaderNames(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	parent := env.RoleToken(t, "u-parent", contract.RoleParent)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	_ = env.RoleToken(t, "u-admin", contract.RoleAdmin)

	if got := status(t, env, http.MethodGet, "/api/leaders", parent); got != http.StatusForbidden {
		t.Errorf("parent GET /api/leaders = %d, want 403", got)
	}
	resp := env.Request(t, http.MethodGet, "/api/leaders", leader, nil)
	defer func() { _ = resp.Body.Close() }()
	var out contract.LeaderList
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&out) != nil {
		t.Fatalf("GET /api/leaders = %d", resp.StatusCode)
	}
	got := map[string]string{}
	for _, l := range out.Leaders {
		got[l.UID] = l.DisplayName
	}
	// testenv names each user after its uid; Parents and emails never appear.
	if len(got) != 2 || got["u-leader"] != "u-leader" || got["u-admin"] != "u-admin" {
		t.Errorf("leaders = %+v, want the Leader and the Admin only", out.Leaders)
	}
}
