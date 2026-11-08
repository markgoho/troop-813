package server_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

// markNotFound calls POST /api/kiosk/not-found.
func markNotFound(t *testing.T, env *testenv.Env, token, scoutID string) (int, contract.ErrorCode) {
	t.Helper()
	resp := env.Request(t, http.MethodPost, "/api/kiosk/not-found", token, contract.KioskNotFoundRequest{ScoutID: scoutID})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		return resp.StatusCode, errorCode(t, resp)
	}
	return resp.StatusCode, ""
}

func TestMarkingAScoutNotFoundRecordsItAndBumpsActivity(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery", "Rowan")
	kiosk, started := unlockKiosk(t, env)
	env.Clock.Advance(10 * time.Minute)

	if status, code := markNotFound(t, env, kiosk, scouts[1]); status != http.StatusNoContent {
		t.Fatalf("not-found = %d %q, want 204", status, code)
	}
	// Marking twice keeps one entry.
	if status, code := markNotFound(t, env, kiosk, scouts[1]); status != http.StatusNoContent {
		t.Fatalf("repeat not-found = %d %q, want 204", status, code)
	}

	m := meetingDoc(t, env, "2026-11-09")
	if !slices.Equal(m.NotFound, []string{scouts[1]}) {
		t.Errorf("notFound = %v, want only %s", m.NotFound, scouts[1])
	}
	if s := sessionDoc(t, env, started.KioskSessionID); !s.LastActivityAt.Equal(testenv.DefaultNow.Add(10 * time.Minute)) {
		t.Errorf("lastActivityAt = %v, want bumped to +10m", s.LastActivityAt)
	}
}

func TestANotFoundScoutCanStillBeCheckedIn(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	kiosk, _ := unlockKiosk(t, env)
	if status, code := markNotFound(t, env, kiosk, scouts[0]); status != http.StatusNoContent {
		t.Fatalf("not-found = %d %q", status, code)
	}

	status, id, code := checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList)

	if status != http.StatusOK || id == "" {
		t.Fatalf("check-in after not-found = %d %q, want 200", status, code)
	}
	if m := meetingDoc(t, env, "2026-11-09"); m.CheckedIn[scouts[0]] != id {
		t.Errorf("checkedIn = %v, want %s → %s", m.CheckedIn, scouts[0], id)
	}
}

func TestMarkNotFoundRejectsBadRequests(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	kiosk, _ := unlockKiosk(t, env)

	if status, code := markNotFound(t, env, kiosk, ""); status != http.StatusBadRequest || code != contract.ErrorCodeBadRequest {
		t.Errorf("blank scout = %d %q, want 400 bad_request", status, code)
	}
	if status, code := markNotFound(t, env, kiosk, "nope"); status != http.StatusNotFound || code != contract.ErrorCodeNotFound {
		t.Errorf("unknown scout = %d %q, want 404 not_found", status, code)
	}
}

func TestMarkNotFoundIsKioskOnly(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	unlockKiosk(t, env)

	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	if status, _ := markNotFound(t, env, leader, scouts[0]); status != http.StatusForbidden {
		t.Errorf("leader not-found = %d, want 403", status)
	}
	if m := meetingDoc(t, env, "2026-11-09"); len(m.NotFound) != 0 {
		t.Errorf("notFound = %v, want none", m.NotFound)
	}
}

func TestMarkNotFoundLogsIDsNotNames(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Peregrine")
	kiosk, _ := unlockKiosk(t, env)
	if status, code := markNotFound(t, env, kiosk, scouts[0]); status != http.StatusNoContent {
		t.Fatalf("not-found = %d %q", status, code)
	}

	logs := env.Logs()
	if strings.Contains(logs, "Peregrine") || !strings.Contains(logs, scouts[0]) {
		t.Fatalf("logs should carry the scout ID and no name:\n%s", logs)
	}
}
