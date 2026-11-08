package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

func answer(a contract.RSVPAnswer) *contract.RSVPAnswer { return &a }

func rosterScout(t *testing.T, env *testenv.Env, first, last string) string {
	t.Helper()
	ref, _, err := env.Firestore.Collection("scouts").Add(context.Background(), contract.Scout{FirstName: first, LastName: last, Active: true, CreatedAt: testenv.DefaultNow})
	if err != nil {
		t.Fatalf("add scout: %v", err)
	}
	return ref.ID
}

func setRSVP(t *testing.T, env *testenv.Env, token, eventID, scoutID string, a *contract.RSVPAnswer) {
	t.Helper()
	path := "/api/events/" + eventID + "/rsvps/" + scoutID
	if status, code := call(t, env, http.MethodPut, path, token, contract.SetRSVP{Answer: a}, nil); status != http.StatusNoContent {
		t.Fatalf("PUT %s = %d %q, want 204", path, status, code)
	}
}

// storedRSVP reads rsvps/{eventId}_{scoutId}, also returning the raw map so
// tests can see that null fields are stored as null, not left out.
func storedRSVP(t *testing.T, env *testenv.Env, eventID, scoutID string) (contract.RSVP, map[string]any, bool) {
	t.Helper()
	snap, err := env.Firestore.Collection("rsvps").Doc(eventID + "_" + scoutID).Get(context.Background())
	if err != nil {
		return contract.RSVP{}, nil, false
	}
	var r contract.RSVP
	if err := snap.DataTo(&r); err != nil {
		t.Fatalf("decode rsvp: %v", err)
	}
	return r, snap.Data(), true
}

func str(p *contract.RSVPAnswer) string {
	if p == nil {
		return "null"
	}
	return string(*p)
}

// wantHistory checks the doc's answers in order, that answer equals the last
// one, and that every entry is a Leader entry (attendanceId null) by uid.
func wantHistory(t *testing.T, r contract.RSVP, raw map[string]any, uid string, answers ...*contract.RSVPAnswer) {
	t.Helper()
	if len(r.History) != len(answers) {
		t.Fatalf("history has %d entries, want %d: %+v", len(r.History), len(answers), r.History)
	}
	for i, a := range answers {
		e := r.History[i]
		if str(e.Answer) != str(a) || e.By != uid || e.AttendanceID != nil {
			t.Errorf("history[%d] = {%s by %s attendance %v}, want {%s by %s attendance null}", i, str(e.Answer), e.By, e.AttendanceID, str(a), uid)
		}
	}
	if str(r.Answer) != str(answers[len(answers)-1]) {
		t.Errorf("answer = %s, want the last history entry's %s", str(r.Answer), str(answers[len(answers)-1]))
	}
	entries, _ := raw["history"].([]any)
	for i, e := range entries {
		m, _ := e.(map[string]any)
		if v, ok := m["attendanceId"]; !ok || v != nil {
			t.Errorf("stored history[%d].attendanceId = %v (present %v), want null", i, v, ok)
		}
	}
	if _, ok := raw["answer"]; !ok {
		t.Error("stored answer is missing, want it present (null when cleared)")
	}
}

func TestALeaderSetsAnRSVP(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	event := createEvent(t, env, leader, campout)
	scout := rosterScout(t, env, "Avery", "Fictional")

	setRSVP(t, env, leader, event, scout, answer(contract.RSVPAnswerGoing))

	r, raw, ok := storedRSVP(t, env, event, scout)
	if !ok {
		t.Fatal("no rsvp doc")
	}
	if r.EventID != event || r.ScoutID != scout {
		t.Errorf("eventId/scoutId = %q/%q, want %q/%q", r.EventID, r.ScoutID, event, scout)
	}
	wantHistory(t, r, raw, "u-leader", answer(contract.RSVPAnswerGoing))
	if !r.History[0].At.Equal(testenv.DefaultNow) {
		t.Errorf("at = %v, want the clock's %v", r.History[0].At, testenv.DefaultNow)
	}
}

func TestChangingAnRSVPAppendsToItsHistory(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	other := env.RoleToken(t, "u-admin", contract.RoleAdmin)
	event := createEvent(t, env, leader, campout)
	scout := rosterScout(t, env, "Avery", "Fictional")
	setRSVP(t, env, leader, event, scout, answer(contract.RSVPAnswerGoing))
	env.Clock.Advance(time.Hour)

	setRSVP(t, env, other, event, scout, answer(contract.RSVPAnswerNotGoing))

	r, _, _ := storedRSVP(t, env, event, scout)
	if len(r.History) != 2 || str(r.Answer) != "notGoing" {
		t.Fatalf("rsvp = %+v, want 2 entries ending notGoing", r)
	}
	last := r.History[1]
	if last.By != "u-admin" || !last.At.Equal(testenv.DefaultNow.Add(time.Hour)) {
		t.Errorf("last entry = %+v, want by u-admin an hour later", last)
	}
}

func TestClearingAnRSVPAppendsANullEntry(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	event := createEvent(t, env, leader, campout)
	scout := rosterScout(t, env, "Avery", "Fictional")
	setRSVP(t, env, leader, event, scout, answer(contract.RSVPAnswerGoing))

	setRSVP(t, env, leader, event, scout, nil)

	r, raw, _ := storedRSVP(t, env, event, scout)
	wantHistory(t, r, raw, "u-leader", answer(contract.RSVPAnswerGoing), nil)
	if v, ok := raw["answer"]; !ok || v != nil {
		t.Errorf("stored answer = %v (present %v), want null", v, ok)
	}
}

func TestAnUnchangedAnswerAddsNoEntry(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	event := createEvent(t, env, leader, campout)
	scout := rosterScout(t, env, "Avery", "Fictional")
	never := rosterScout(t, env, "Blake", "Fictional")
	setRSVP(t, env, leader, event, scout, answer(contract.RSVPAnswerGoing))

	setRSVP(t, env, leader, event, scout, answer(contract.RSVPAnswerGoing))
	setRSVP(t, env, leader, event, never, nil)

	if r, _, _ := storedRSVP(t, env, event, scout); len(r.History) != 1 {
		t.Errorf("repeat answer: %d entries, want 1", len(r.History))
	}
	if _, _, ok := storedRSVP(t, env, event, never); ok {
		t.Error("clearing a never-answered RSVP created a doc (it would block deleting the Event)")
	}
}

func TestAnRSVPNeedsAValidAnswerAndAScout(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	event := createEvent(t, env, leader, campout)
	scout := rosterScout(t, env, "Avery", "Fictional")

	status, code := call(t, env, http.MethodPut, "/api/events/"+event+"/rsvps/"+scout, leader, map[string]any{"answer": "maybe"}, nil)
	if status != http.StatusBadRequest || code != contract.ErrorCodeBadRequest {
		t.Errorf("answer maybe = %d %q, want 400 invalid", status, code)
	}
	status, code = call(t, env, http.MethodPut, "/api/events/"+event+"/rsvps/nobody", leader, contract.SetRSVP{Answer: answer(contract.RSVPAnswerGoing)}, nil)
	if status != http.StatusNotFound || code != contract.ErrorCodeNotFound {
		t.Errorf("missing scout = %d %q, want 404 notFound", status, code)
	}
	if _, _, ok := storedRSVP(t, env, event, scout); ok {
		t.Error("an invalid answer was stored")
	}
}
