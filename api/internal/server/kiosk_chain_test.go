package server_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

// rawLastCheckIn reads the stored lastCheckIn: present reports whether the
// field exists at all (HQ needs it null on a new Meeting, not missing).
func rawLastCheckIn(t *testing.T, env *testenv.Env, id string) (value any, present bool) {
	t.Helper()
	snap, err := env.Firestore.Collection("meetings").Doc(id).Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	value, present = snap.Data()["lastCheckIn"]
	return value, present
}

func TestAMeetingStartsWithNoLastCheckIn(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	seedScouts(t, env, true, "Avery")
	unlockKiosk(t, env)

	if v, present := rawLastCheckIn(t, env, "2026-11-09"); !present || v != nil {
		t.Fatalf("stored lastCheckIn = %#v (present %v), want an explicit null", v, present)
	}
}

func TestAKioskCheckInMovesTheChain(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery", "Rowan")
	kiosk, _ := unlockKiosk(t, env)

	checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList)
	checkIn(t, env, kiosk, scouts[1], contract.AttendanceMethodHandoff)
	// A repeat check-in writes nothing, so the chain stays where it was.
	checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList)

	m := meetingDoc(t, env, "2026-11-09")
	if m.LastCheckIn == nil || *m.LastCheckIn != scouts[1] {
		t.Fatalf("lastCheckIn = %v, want %s", m.LastCheckIn, scouts[1])
	}
}

func TestALeaderCheckInDoesNotMoveTheChain(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery", "Rowan")
	kiosk, _ := unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList)

	if status, _, code := leaderCheckIn(t, env, leader, "2026-11-09", scouts[1]); status != http.StatusOK {
		t.Fatalf("leader check-in = %d %q", status, code)
	}

	m := meetingDoc(t, env, "2026-11-09")
	if m.LastCheckIn == nil || *m.LastCheckIn != scouts[0] {
		t.Fatalf("lastCheckIn = %v, want still %s", m.LastCheckIn, scouts[0])
	}
}

func TestTheKioskRefusesInactiveScouts(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	gone := seedScouts(t, env, false, "Marlow")[0]
	kiosk, _ := unlockKiosk(t, env)

	if status, _, code := checkIn(t, env, kiosk, gone, contract.AttendanceMethodNameList); status != http.StatusNotFound || code != contract.ErrorCodeNotFound {
		t.Errorf("kiosk check-in of an inactive Scout = %d %q, want 404 not_found", status, code)
	}
	if status, code := markNotFound(t, env, kiosk, gone); status != http.StatusNotFound || code != contract.ErrorCodeNotFound {
		t.Errorf("not-found for an inactive Scout = %d %q, want 404 not_found", status, code)
	}
	m := meetingDoc(t, env, "2026-11-09")
	if len(m.CheckedIn) != 0 || len(m.NotFound) != 0 || m.LastCheckIn != nil {
		t.Errorf("meeting = %+v, want untouched", m)
	}
	if n := len(attendanceDocs(t, env, "2026-11-09")); n != 0 {
		t.Errorf("%d attendance records, want 0", n)
	}
}

func TestALeaderCanStillCheckInAnInactiveScout(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	gone := seedScouts(t, env, false, "Marlow")[0]
	unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)

	if status, _, code := leaderCheckIn(t, env, leader, "2026-11-09", gone); status != http.StatusOK {
		t.Fatalf("leader check-in of an inactive Scout = %d %q, want 200", status, code)
	}
}
