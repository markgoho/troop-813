package server_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

// kioskRSVP calls PUT /api/kiosk/rsvps.
func kioskRSVP(t *testing.T, env *testenv.Env, token, eventID, scoutID string, a contract.RSVPAnswer) (int, contract.ErrorCode) {
	t.Helper()
	return call(t, env, http.MethodPut, "/api/kiosk/rsvps", token,
		contract.KioskSetRSVPRequest{EventID: eventID, ScoutID: scoutID, Answer: a}, nil)
}

func mustRSVP(t *testing.T, env *testenv.Env, token, ev, scout string, a contract.RSVPAnswer) {
	t.Helper()
	if status, code := kioskRSVP(t, env, token, ev, scout, a); status != http.StatusNoContent {
		t.Fatalf("kiosk RSVP = %d %q, want 204", status, code)
	}
}

// rollbackPreview calls GET /api/meetings/{date}/attendance/{id}/rsvp-rollback.
func rollbackPreview(t *testing.T, env *testenv.Env, token, date, attendanceID string) (int, contract.RSVPRollbackPreview) {
	t.Helper()
	resp := env.Request(t, http.MethodGet, "/api/meetings/"+date+"/attendance/"+attendanceID+"/rsvp-rollback", token, nil)
	defer func() { _ = resp.Body.Close() }()
	var out contract.RSVPRollbackPreview
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decode preview: %v", err)
		}
	}
	return resp.StatusCode, out
}

func fallEvent(t *testing.T, env *testenv.Env, leader, name string) string {
	t.Helper()
	return createEvent(t, env, leader, contract.EventInput{Name: name, StartDate: "2026-11-13", EndDate: "2026-11-15"})
}

func TestAKioskRSVPRecordsTheCheckInOnTheEntry(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	kiosk, started := unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	ev := fallEvent(t, env, leader, "Fall Campout")
	_, att, _ := checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList)
	env.Clock.Advance(2 * time.Minute)

	mustRSVP(t, env, kiosk, ev, scouts[0], contract.RSVPAnswerGoing)

	r, _, ok := storedRSVP(t, env, ev, scouts[0])
	if !ok || str(r.Answer) != "going" || len(r.History) != 1 {
		t.Fatalf("rsvp = %+v (found %v), want going with one entry", r, ok)
	}
	e := r.History[0]
	if e.By != started.KioskSessionID || e.AttendanceID == nil || *e.AttendanceID != att || !e.At.Equal(testenv.DefaultNow.Add(2*time.Minute)) {
		t.Errorf("entry = %+v, want by the session with attendanceId %s", e, att)
	}
	if s := sessionDoc(t, env, started.KioskSessionID); !s.LastActivityAt.Equal(testenv.DefaultNow.Add(2 * time.Minute)) {
		t.Errorf("lastActivityAt = %v, want bumped", s.LastActivityAt)
	}
}

func TestAKioskRSVPNeedsAScoutCheckedInThisSession(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery", "Rowan")
	kiosk, _ := unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	ev := fallEvent(t, env, leader, "Fall Campout")

	if status, code := kioskRSVP(t, env, kiosk, ev, scouts[0], contract.RSVPAnswerGoing); status != http.StatusForbidden || code != contract.ErrorCodeForbidden {
		t.Errorf("not checked in = %d %q, want 403 forbidden", status, code)
	}
	// A Leader's check-in has no Kiosk session behind it.
	_, _, _ = leaderCheckIn(t, env, leader, "2026-11-09", scouts[1])
	if status, _ := kioskRSVP(t, env, kiosk, ev, scouts[1], contract.RSVPAnswerGoing); status != http.StatusForbidden {
		t.Errorf("leader-checked-in = %d, want 403", status)
	}
	// Checked in under a session a later Start replaced.
	_, _, _ = checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList)
	newKiosk, _ := unlockKiosk(t, env)
	if status, _ := kioskRSVP(t, env, newKiosk, ev, scouts[0], contract.RSVPAnswerGoing); status != http.StatusForbidden {
		t.Errorf("other session's check-in = %d, want 403", status)
	}
	for _, id := range scouts {
		if _, _, ok := storedRSVP(t, env, ev, id); ok {
			t.Errorf("rsvp for %s exists, want none", id)
		}
	}
}

func TestAKioskRSVPRejectsBadRequests(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	kiosk, _ := unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	ev := fallEvent(t, env, leader, "Fall Campout")
	cancelled := fallEvent(t, env, leader, "Cancelled")
	if status, _ := call(t, env, http.MethodPost, "/api/events/"+cancelled+"/cancel", leader, nil, nil); status != http.StatusNoContent {
		t.Fatalf("cancel = %d", status)
	}
	started := createEvent(t, env, leader, contract.EventInput{Name: "Started", StartDate: "2026-11-09", EndDate: "2026-11-10"})
	_, _, _ = checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList)

	for name, tc := range map[string]struct {
		event  string
		answer contract.RSVPAnswer
		want   int
	}{
		"unknown Event":       {"nope", contract.RSVPAnswerGoing, http.StatusNotFound},
		"cancelled Event":     {cancelled, contract.RSVPAnswerGoing, http.StatusBadRequest},
		"Event already begun": {started, contract.RSVPAnswerGoing, http.StatusBadRequest},
		"an unknown answer":   {ev, "maybe", http.StatusBadRequest},
		"no answer":           {ev, "", http.StatusBadRequest},
		"no Event":            {"", contract.RSVPAnswerGoing, http.StatusBadRequest},
	} {
		if status, _ := kioskRSVP(t, env, kiosk, tc.event, scouts[0], tc.answer); status != tc.want {
			t.Errorf("%s = %d, want %d", name, status, tc.want)
		}
	}
	if status, _ := kioskRSVP(t, env, leader, ev, scouts[0], contract.RSVPAnswerGoing); status != http.StatusForbidden {
		t.Errorf("leader on the Kiosk route = %d, want 403", status)
	}
}

// A Kiosk answer equal to the current one appends no entry (the rsvp
// module's rule for every caller): nothing for Undo to roll back, and the
// answer is right either way.
func TestAKioskAnswerEqualToTheCurrentOneRecordsNothing(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	kiosk, _ := unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	ev := fallEvent(t, env, leader, "Fall Campout")
	setRSVP(t, env, leader, ev, scouts[0], answer(contract.RSVPAnswerGoing))
	_, _, _ = checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList)

	mustRSVP(t, env, kiosk, ev, scouts[0], contract.RSVPAnswerGoing)

	if r, _, _ := storedRSVP(t, env, ev, scouts[0]); len(r.History) != 1 {
		t.Errorf("history = %+v, want only the Leader's entry", r.History)
	}
}

func TestUndoRollsBackOnlyThatCheckInsAnswers(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	scout := scouts[0]
	kiosk, _ := unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	fall := fallEvent(t, env, leader, "Fall Campout")
	winter := fallEvent(t, env, leader, "Winter Campout")
	spring := fallEvent(t, env, leader, "Spring Campout")
	summer := fallEvent(t, env, leader, "Summer Camp")

	// Fall: a Leader said notGoing earlier. Winter: answered in a check-in
	// that was undone before tonight's.
	setRSVP(t, env, leader, fall, scout, answer(contract.RSVPAnswerNotGoing))
	_, first, _ := checkIn(t, env, kiosk, scout, contract.AttendanceMethodNameList)
	mustRSVP(t, env, kiosk, winter, scout, contract.RSVPAnswerGoing)
	_, _ = undoCheckIn(t, env, leader, "2026-11-09", first)
	// Tonight's check-in: changes Fall, answers Spring and Summer; then a
	// Leader overrides Summer.
	_, second, _ := checkIn(t, env, kiosk, scout, contract.AttendanceMethodNameList)
	mustRSVP(t, env, kiosk, fall, scout, contract.RSVPAnswerGoing)
	mustRSVP(t, env, kiosk, spring, scout, contract.RSVPAnswerGoing)
	mustRSVP(t, env, kiosk, summer, scout, contract.RSVPAnswerNotGoing)
	setRSVP(t, env, leader, summer, scout, answer(contract.RSVPAnswerGoing))

	status, preview := rollbackPreview(t, env, leader, "2026-11-09", second)
	if status != http.StatusOK {
		t.Fatalf("preview = %d", status)
	}
	got := map[string][2]string{}
	for _, it := range preview.Items {
		got[it.EventName] = [2]string{str(it.From), str(it.To)}
	}
	// Summer's Leader edit stays the answer, so it isn't a rollback.
	want := map[string][2]string{"Fall Campout": {"going", "notGoing"}, "Spring Campout": {"going", "null"}}
	if len(got) != len(want) || got["Fall Campout"] != want["Fall Campout"] || got["Spring Campout"] != want["Spring Campout"] {
		t.Fatalf("preview = %v, want %v", got, want)
	}

	if status, code := undoCheckIn(t, env, leader, "2026-11-09", second); status != http.StatusNoContent {
		t.Fatalf("undo = %d %q", status, code)
	}

	check := func(name, ev, want string, entries int) {
		t.Helper()
		r, raw, ok := storedRSVP(t, env, ev, scout)
		if !ok || str(r.Answer) != want || len(r.History) != entries {
			t.Errorf("%s = %+v (found %v), want %s with %d entries", name, r, ok, want, entries)
		}
		if _, has := raw["answer"]; ok && !has {
			t.Errorf("%s: stored answer missing, want an explicit null", name)
		}
	}
	check("fall", fall, "notGoing", 1)  // back to the Leader's answer
	check("winter", winter, "null", 0)  // rolled back by the first undo
	check("spring", spring, "null", 0)  // no earlier answer
	check("summer", summer, "going", 1) // the Leader's edit survives

	if _, again := rollbackPreview(t, env, leader, "2026-11-09", second); len(again.Items) != 0 {
		t.Errorf("preview after undo = %+v, want empty", again)
	}
	// Undoing twice is still fine.
	if status, _ := undoCheckIn(t, env, leader, "2026-11-09", second); status != http.StatusNoContent {
		t.Errorf("second undo = %d, want 204", status)
	}
}

func TestRollbackPreviewNeedsALeaderAndARealCheckIn(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	kiosk, _ := unlockKiosk(t, env)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)

	if status, _ := rollbackPreview(t, env, leader, "2026-11-09", "nope"); status != http.StatusNotFound {
		t.Errorf("unknown attendance = %d, want 404", status)
	}
	if status, _ := rollbackPreview(t, env, leader, "2026-11-10", "nope"); status != http.StatusNotFound {
		t.Errorf("unknown Meeting = %d, want 404", status)
	}
	if status, _ := rollbackPreview(t, env, kiosk, "2026-11-09", "nope"); status != http.StatusForbidden {
		t.Errorf("kiosk token = %d, want 403", status)
	}
}
