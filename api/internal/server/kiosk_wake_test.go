package server_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

// wakeKiosk calls POST /api/kiosk/wake.
func wakeKiosk(t *testing.T, env *testenv.Env, token string) (int, contract.ErrorCode) {
	t.Helper()
	resp := env.Request(t, http.MethodPost, "/api/kiosk/wake", token, nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		return resp.StatusCode, errorCode(t, resp)
	}
	return resp.StatusCode, ""
}

func TestALeaderWakesAnAsleepKiosk(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery", "Rowan")
	kiosk, started := unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)

	env.Clock.Advance(31 * time.Minute)
	if status, _, code := checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList); status != http.StatusUnauthorized || code != contract.ErrorCodeSessionAsleep {
		t.Fatalf("check-in while asleep = %d %q, want 401 session_asleep", status, code)
	}
	if status, code := wakeKiosk(t, env, leader); status != http.StatusNoContent {
		t.Fatalf("wake = %d %q, want 204", status, code)
	}
	if got := sessionDoc(t, env, started.KioskSessionID).LastActivityAt; !got.Equal(env.Clock.Now()) {
		t.Fatalf("lastActivityAt = %v, want now %v", got, env.Clock.Now())
	}
	if status, _, code := checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList); status != http.StatusOK {
		t.Fatalf("check-in after wake = %d %q, want 200", status, code)
	}
}

func TestWakingAKioskThatIsAlreadyAwakeIsHarmless(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	env.Clock.Advance(10 * time.Minute)
	if status, code := wakeKiosk(t, env, leader); status != http.StatusNoContent {
		t.Fatalf("wake = %d %q, want 204", status, code)
	}
}

func TestAKioskCannotBeWokenAfter2200(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	env.Clock.Set(time.Date(2026, 11, 9, 21, 0, 0, 0, clock.NY))
	kiosk, started := unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)

	env.Clock.Set(time.Date(2026, 11, 9, 22, 0, 0, 0, clock.NY))
	before := sessionDoc(t, env, started.KioskSessionID).LastActivityAt
	if status, code := wakeKiosk(t, env, leader); status != http.StatusConflict || code != contract.ErrorCodeSessionExpired {
		t.Fatalf("wake at 22:00 = %d %q, want 409 session_expired", status, code)
	}
	if got := sessionDoc(t, env, started.KioskSessionID).LastActivityAt; !got.Equal(before) {
		t.Fatalf("a refused wake changed lastActivityAt to %v", got)
	}
	if status, _, code := checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList); status != http.StatusUnauthorized || code != contract.ErrorCodeSessionExpired {
		t.Fatalf("check-in after 22:00 = %d %q, want 401 session_expired", status, code)
	}
}

func TestWakingNeedsALiveSession(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	if status, code := wakeKiosk(t, env, leader); status != http.StatusNotFound || code != contract.ErrorCodeNotFound {
		t.Fatalf("wake with no session = %d %q, want 404 not_found", status, code)
	}
}

func TestOnlyALeaderWakesTheKiosk(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	kiosk, _ := unlockKiosk(t, env)
	for name, token := range map[string]string{
		"kiosk":  kiosk,
		"parent": env.RoleToken(t, "u-parent", contract.RoleParent),
	} {
		if status, _ := wakeKiosk(t, env, token); status != http.StatusForbidden {
			t.Errorf("%s wake = %d, want 403", name, status)
		}
	}
	if status, _ := wakeKiosk(t, env, ""); status != http.StatusUnauthorized {
		t.Errorf("anonymous wake = %d, want 401", status)
	}
}
