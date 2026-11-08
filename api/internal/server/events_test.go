package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

var campout = contract.EventInput{Name: "Fall Campout", StartDate: "2026-11-13", EndDate: "2026-11-15", Location: "Camp Fictional"}

// call sends a request and returns its status and, for an error status, the code.
// out, if non-nil, receives a 2xx JSON body.
func call(t *testing.T, env *testenv.Env, method, path, token string, body, out any) (int, contract.ErrorCode) {
	t.Helper()
	resp := env.Request(t, method, path, token, body)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		var e contract.ErrorBody
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return resp.StatusCode, e.Code
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("decode %s %s: %v", method, path, err)
		}
	}
	return resp.StatusCode, ""
}

func createEvent(t *testing.T, env *testenv.Env, token string, in contract.EventInput) string {
	t.Helper()
	var created contract.EventCreated
	if status, code := call(t, env, http.MethodPost, "/api/events", token, in, &created); status != http.StatusCreated {
		t.Fatalf("POST /api/events = %d %q, want 201", status, code)
	}
	return created.ID
}

func storedEvent(t *testing.T, env *testenv.Env, id string) (contract.Event, bool) {
	t.Helper()
	snap, err := env.Firestore.Collection("events").Doc(id).Get(context.Background())
	if err != nil {
		return contract.Event{}, false
	}
	var e contract.Event
	if err := snap.DataTo(&e); err != nil {
		t.Fatalf("decode events/%s: %v", id, err)
	}
	return e, true
}

func TestALeaderAddsAnEvent(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)

	id := createEvent(t, env, leader, contract.EventInput{Name: "  Fall Campout ", StartDate: "2026-11-13", EndDate: "2026-11-15"})

	got, ok := storedEvent(t, env, id)
	want := contract.Event{Name: "Fall Campout", StartDate: "2026-11-13", EndDate: "2026-11-15"}
	if !ok || got != want {
		t.Fatalf("events/%s = %+v (exists %v), want %+v", id, got, ok, want)
	}
}

func TestAnEventNeedsANameAndValidDates(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	for name, in := range map[string]contract.EventInput{
		"blank name":        {Name: "  ", StartDate: "2026-11-13", EndDate: "2026-11-15"},
		"bad start":         {Name: "X", StartDate: "11/13/2026", EndDate: "2026-11-15"},
		"impossible date":   {Name: "X", StartDate: "2026-02-30", EndDate: "2026-03-01"},
		"ends before start": {Name: "X", StartDate: "2026-11-15", EndDate: "2026-11-13"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status, code := call(t, env, http.MethodPost, "/api/events", leader, in, nil)
			if status != http.StatusBadRequest || code != contract.ErrorCodeBadRequest {
				t.Fatalf("POST = %d %q, want 400 invalid", status, code)
			}
		})
	}
}

func TestALeaderEditsAnEventKeepingItsCancelledState(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	id := createEvent(t, env, leader, campout)
	if status, code := call(t, env, http.MethodPost, "/api/events/"+id+"/cancel", leader, nil, nil); status != http.StatusNoContent {
		t.Fatalf("cancel = %d %q, want 204", status, code)
	}

	edit := contract.EventInput{Name: "Winter Campout", StartDate: "2026-12-11", EndDate: "2026-12-13"}
	if status, code := call(t, env, http.MethodPut, "/api/events/"+id, leader, edit, nil); status != http.StatusNoContent {
		t.Fatalf("PUT = %d %q, want 204", status, code)
	}

	got, _ := storedEvent(t, env, id)
	want := contract.Event{Name: "Winter Campout", StartDate: "2026-12-11", EndDate: "2026-12-13", Cancelled: true}
	if got != want {
		t.Fatalf("after edit = %+v, want %+v (location removed, still cancelled)", got, want)
	}
}

func TestALeaderCancelsAndUncancelsAnEvent(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	id := createEvent(t, env, leader, campout)

	call(t, env, http.MethodPost, "/api/events/"+id+"/cancel", leader, nil, nil)
	if got, _ := storedEvent(t, env, id); !got.Cancelled {
		t.Fatal("not cancelled after cancel")
	}
	if status, code := call(t, env, http.MethodPost, "/api/events/"+id+"/uncancel", leader, nil, nil); status != http.StatusNoContent {
		t.Fatalf("uncancel = %d %q, want 204", status, code)
	}
	if got, _ := storedEvent(t, env, id); got.Cancelled || got.Name != campout.Name {
		t.Fatalf("after uncancel = %+v, want not cancelled, fields kept", got)
	}
}

func TestChangingAMissingEventIsNotFound(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPut, "/api/events/nope", campout},
		{http.MethodPost, "/api/events/nope/cancel", nil},
		{http.MethodPost, "/api/events/nope/uncancel", nil},
		{http.MethodDelete, "/api/events/nope", nil},
		{http.MethodPut, "/api/events/nope/rsvps/s1", contract.SetRSVP{Answer: answer(contract.RSVPAnswerGoing)}},
	} {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			t.Parallel()
			status, code := call(t, env, c.method, c.path, leader, c.body, nil)
			if status != http.StatusNotFound || code != contract.ErrorCodeNotFound {
				t.Fatalf("= %d %q, want 404 notFound", status, code)
			}
			if _, ok := storedEvent(t, env, "nope"); ok {
				t.Fatal("created events/nope")
			}
		})
	}
}

func TestALeaderDeletesAnUnusedEvent(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	id := createEvent(t, env, leader, campout)

	if status, code := call(t, env, http.MethodDelete, "/api/events/"+id, leader, nil, nil); status != http.StatusNoContent {
		t.Fatalf("DELETE = %d %q, want 204", status, code)
	}
	if _, ok := storedEvent(t, env, id); ok {
		t.Fatal("event still stored")
	}
}

func TestAnEventWithRSVPsCannotBeDeleted(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	id := createEvent(t, env, leader, campout)
	scout := rosterScout(t, env, "Avery", "Fictional")
	setRSVP(t, env, leader, id, scout, answer(contract.RSVPAnswerGoing))
	// Even a cleared RSVP keeps its history, which delete would wipe (no backups, ADR 0004).
	setRSVP(t, env, leader, id, scout, nil)

	status, code := call(t, env, http.MethodDelete, "/api/events/"+id, leader, nil, nil)

	if status != http.StatusConflict || code != contract.ErrorCodeInUse {
		t.Fatalf("DELETE = %d %q, want 409 inUse", status, code)
	}
	if _, ok := storedEvent(t, env, id); !ok {
		t.Fatal("event was deleted")
	}
}
