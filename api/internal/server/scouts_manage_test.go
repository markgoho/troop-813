package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

// scoutCall sends a request and returns the status plus the error body (zero on 2xx).
func scoutCall(t *testing.T, env *testenv.Env, method, path, token string, body any) (int, contract.ErrorBody) {
	t.Helper()
	resp := env.Request(t, method, path, token, body)
	defer func() { _ = resp.Body.Close() }()
	var e contract.ErrorBody
	if resp.StatusCode >= 400 {
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			t.Fatalf("decode error body (status %d): %v", resp.StatusCode, err)
		}
	}
	return resp.StatusCode, e
}

func newScout(t *testing.T, env *testenv.Env, first, last string, active bool) string {
	t.Helper()
	ref, _, err := env.Firestore.Collection("scouts").Add(context.Background(), contract.Scout{
		FirstName: first, LastName: last, Active: active, CreatedAt: testenv.DefaultNow,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ref.ID
}

func scoutExists(t *testing.T, env *testenv.Env, id string) bool {
	t.Helper()
	snap, err := env.Firestore.Collection("scouts").Doc(id).Get(context.Background())
	return err == nil && snap.Exists()
}

func TestALeaderEditsAScoutsName(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	id := newScout(t, env, "Avery", "Quill", true)

	status, _ := scoutCall(t, env, http.MethodPut, "/api/scouts/"+id, leader, contract.EditScoutRequest{FirstName: " Averie ", LastName: "Quill-Marsh"})

	got := scoutDoc(t, env, id)
	if status != http.StatusNoContent || got.FirstName != "Averie" || got.LastName != "Quill-Marsh" || !got.Active {
		t.Fatalf("PUT = %d, scout %+v, want 204 renamed and still active", status, got)
	}
}

func TestEditingKeepsYourOwnNameButNotSomeoneElses(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	id := newScout(t, env, "Avery", "Quill", true)
	other := newScout(t, env, "Rowan", "Pike", true)

	if status, _ := scoutCall(t, env, http.MethodPut, "/api/scouts/"+id, leader, contract.EditScoutRequest{FirstName: "AVERY", LastName: "Quill"}); status != http.StatusNoContent {
		t.Fatalf("re-casing own name = %d, want 204", status)
	}
	status, e := scoutCall(t, env, http.MethodPut, "/api/scouts/"+other, leader, contract.EditScoutRequest{FirstName: "Avery", LastName: "Quill"})
	if status != http.StatusConflict || e.Code != contract.ErrorCodeDuplicateScout {
		t.Fatalf("PUT onto another Scout's name = %d %+v, want 409 duplicate_scout", status, e)
	}
}

func TestEditingNeedsBothNamesAndARealScout(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	id := newScout(t, env, "Avery", "Quill", true)

	if status, e := scoutCall(t, env, http.MethodPut, "/api/scouts/"+id, leader, contract.EditScoutRequest{FirstName: " ", LastName: "Q"}); status != http.StatusBadRequest || e.Code != contract.ErrorCodeBadRequest {
		t.Fatalf("blank = %d %+v, want 400", status, e)
	}
	if status, e := scoutCall(t, env, http.MethodPut, "/api/scouts/nope", leader, contract.EditScoutRequest{FirstName: "A", LastName: "B"}); status != http.StatusNotFound || e.Code != contract.ErrorCodeNotFound {
		t.Fatalf("missing = %d %+v, want 404", status, e)
	}
}

func TestDeactivateAndReactivate(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	id := newScout(t, env, "Avery", "Quill", true)

	if status, _ := scoutCall(t, env, http.MethodPost, "/api/scouts/"+id+"/deactivate", leader, nil); status != http.StatusNoContent || scoutDoc(t, env, id).Active {
		t.Fatalf("deactivate = %d, active %v", status, scoutDoc(t, env, id).Active)
	}
	if status, _ := scoutCall(t, env, http.MethodPost, "/api/scouts/"+id+"/reactivate", leader, nil); status != http.StatusNoContent || !scoutDoc(t, env, id).Active {
		t.Fatalf("reactivate = %d, active %v", status, scoutDoc(t, env, id).Active)
	}
	if status, _ := scoutCall(t, env, http.MethodPost, "/api/scouts/nope/deactivate", leader, nil); status != http.StatusNotFound {
		t.Fatalf("deactivate missing = %d, want 404", status)
	}
}

func TestReactivatingIsRefusedWhenAnActiveScoutHasTheName(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	old := newScout(t, env, "Avery", "Quill", false)
	newScout(t, env, "Avery", "Quill", true)

	status, e := scoutCall(t, env, http.MethodPost, "/api/scouts/"+old+"/reactivate", leader, nil)
	if status != http.StatusConflict || e.Code != contract.ErrorCodeDuplicateScout {
		t.Fatalf("reactivate = %d %+v, want 409 duplicate_scout", status, e)
	}
}

func TestAddingADuplicateOfAnInactiveScoutOffersReactivate(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	old := newScout(t, env, "Avery", "Quill", false)

	status, e := scoutCall(t, env, http.MethodPost, "/api/scouts", leader, contract.AddScoutRequest{FirstName: "avery", LastName: "QUILL"})

	if status != http.StatusConflict || e.Code != contract.ErrorCodeDuplicateScout || e.InactiveMatchID != old {
		t.Fatalf("POST = %d %+v, want 409 duplicate_scout with inactiveMatchId %s", status, e, old)
	}
	snaps, err := env.Firestore.Collection("scouts").Documents(context.Background()).GetAll()
	if err != nil || len(snaps) != 1 {
		t.Fatalf("%d scouts (%v), want 1", len(snaps), err)
	}
}

func TestAnActiveDuplicateCarriesNoInactiveMatchID(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	newScout(t, env, "Avery", "Quill", false)
	newScout(t, env, "Avery", "Quill", true)

	status, e := scoutCall(t, env, http.MethodPost, "/api/scouts", leader, contract.AddScoutRequest{FirstName: "Avery", LastName: "Quill"})
	if status != http.StatusConflict || e.InactiveMatchID != "" {
		t.Fatalf("POST = %d %+v, want 409 without inactiveMatchId", status, e)
	}
}

func TestAnUnusedScoutIsDeleted(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	id := newScout(t, env, "Avery", "Quill", true)

	if status, _ := scoutCall(t, env, http.MethodDelete, "/api/scouts/"+id, leader, nil); status != http.StatusNoContent || scoutExists(t, env, id) {
		t.Fatalf("DELETE = %d, exists %v", status, scoutExists(t, env, id))
	}
	if status, _ := scoutCall(t, env, http.MethodDelete, "/api/scouts/"+id, leader, nil); status != http.StatusNotFound {
		t.Fatalf("second DELETE = %d, want 404", status)
	}
}

func TestAScoutWithAttendanceCannotBeDeleted(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	ids := seedScouts(t, env, true, "Avery")
	kiosk, _ := unlockKiosk(t, env)
	if status, _, code := checkIn(t, env, kiosk, ids[0], contract.AttendanceMethodNameList); status != http.StatusOK {
		t.Fatalf("check-in: %d %q", status, code)
	}

	status, e := scoutCall(t, env, http.MethodDelete, "/api/scouts/"+ids[0], leader, nil)
	if status != http.StatusConflict || e.Code != contract.ErrorCodeInUse || !scoutExists(t, env, ids[0]) {
		t.Fatalf("DELETE = %d %+v, want 409 in_use and the Scout kept", status, e)
	}
}

func TestAScoutWithAnRSVPCannotBeDeleted(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	id := newScout(t, env, "Avery", "Quill", true)
	eventID := createEvent(t, env, leader, campout)
	setRSVP(t, env, leader, eventID, id, answer(contract.RSVPAnswerGoing))

	status, e := scoutCall(t, env, http.MethodDelete, "/api/scouts/"+id, leader, nil)
	if status != http.StatusConflict || e.Code != contract.ErrorCodeInUse {
		t.Fatalf("DELETE = %d %+v, want 409 in_use", status, e)
	}
}

func TestOnlyLeadersManageScouts(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	id := newScout(t, env, "Avery", "Quill", true)
	parent := env.RoleToken(t, "u-parent", contract.RoleParent)
	for _, c := range []struct{ method, path string }{
		{http.MethodPut, "/api/scouts/" + id},
		{http.MethodPost, "/api/scouts/" + id + "/deactivate"},
		{http.MethodPost, "/api/scouts/" + id + "/reactivate"},
		{http.MethodDelete, "/api/scouts/" + id},
	} {
		if status, _ := scoutCall(t, env, c.method, c.path, "", nil); status != http.StatusUnauthorized {
			t.Errorf("%s %s anonymous = %d, want 401", c.method, c.path, status)
		}
		if status, _ := scoutCall(t, env, c.method, c.path, parent, nil); status != http.StatusForbidden {
			t.Errorf("%s %s parent = %d, want 403", c.method, c.path, status)
		}
	}
	if !scoutExists(t, env, id) || !scoutDoc(t, env, id).Active {
		t.Fatal("a rejected call changed the Scout")
	}
}

// --- live Meeting insert ---

func TestAScoutAddedDuringALiveMeetingJoinsTonightsHandoffOrder(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	ids := seedScouts(t, env, true, "A", "B", "C", "D", "E", "F")
	kiosk, started := unlockKiosk(t, env)
	for _, id := range ids[:2] {
		checkIn(t, env, kiosk, id, contract.AttendanceMethodNameList)
	}
	before := meetingDoc(t, env, started.MeetingID)

	status, newID, code := addScout(t, env, leader, contract.AddScoutRequest{FirstName: "Rowan", LastName: "Pike"})
	if status != http.StatusCreated {
		t.Fatalf("add: %d %q", status, code)
	}

	after := meetingDoc(t, env, started.MeetingID)
	if n := countOf(after.HandoffOrder, newID); n != 1 {
		t.Fatalf("new Scout appears %d times in %v", n, after.HandoffOrder)
	}
	rest := slices.DeleteFunc(slices.Clone(after.HandoffOrder), func(s string) bool { return s == newID })
	if !slices.Equal(rest, before.HandoffOrder) {
		t.Fatalf("existing order changed: %v -> %v", before.HandoffOrder, after.HandoffOrder)
	}
	i := slices.Index(after.HandoffOrder, newID)
	nextUnchecked := i+1 < len(after.HandoffOrder) && !isChecked(after, after.HandoffOrder[i+1])
	prevUnchecked := i > 0 && !isChecked(after, after.HandoffOrder[i-1])
	if !nextUnchecked && !prevUnchecked {
		t.Fatalf("new Scout at %d sits among checked-in Scouts only: %v (checked %v)", i, after.HandoffOrder, after.CheckedIn)
	}
	if _, in := after.CheckedIn[newID]; in {
		t.Fatal("new Scout marked checked in")
	}
}

func TestAScoutAddedWithNoLiveMeetingTouchesNoMeeting(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)

	if status, _, code := addScout(t, env, leader, contract.AddScoutRequest{FirstName: "Rowan", LastName: "Pike"}); status != http.StatusCreated {
		t.Fatalf("add: %d %q", status, code)
	}
	snaps, err := env.Firestore.Collection("meetings").Documents(context.Background()).GetAll()
	if err != nil || len(snaps) != 0 {
		t.Fatalf("meetings = %d (%v), want none", len(snaps), err)
	}
}

func TestReactivatingDuringALiveMeetingJoinsTheHandoffOrder(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	seedScouts(t, env, true, "A", "B")
	away := newScout(t, env, "Rowan", "Pike", false)
	_, started := unlockKiosk(t, env)
	if slices.Contains(meetingDoc(t, env, started.MeetingID).HandoffOrder, away) {
		t.Fatal("inactive Scout in the order at Start")
	}

	if status, _ := scoutCall(t, env, http.MethodPost, "/api/scouts/"+away+"/reactivate", leader, nil); status != http.StatusNoContent {
		t.Fatalf("reactivate = %d", status)
	}
	if n := countOf(meetingDoc(t, env, started.MeetingID).HandoffOrder, away); n != 1 {
		t.Fatalf("reactivated Scout appears %d times", n)
	}
	scoutCall(t, env, http.MethodPost, "/api/scouts/"+away+"/deactivate", leader, nil)
	scoutCall(t, env, http.MethodPost, "/api/scouts/"+away+"/reactivate", leader, nil)
	if n := countOf(meetingDoc(t, env, started.MeetingID).HandoffOrder, away); n != 1 {
		t.Fatalf("after a second reactivate the Scout appears %d times", n)
	}
}

func TestDeletingAScoutDuringALiveMeetingLeavesTheHandoffOrder(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	ids := seedScouts(t, env, true, "A", "B", "C")
	_, started := unlockKiosk(t, env)

	if status, _ := scoutCall(t, env, http.MethodDelete, "/api/scouts/"+ids[0], leader, nil); status != http.StatusNoContent {
		t.Fatalf("delete = %d", status)
	}
	if got := meetingDoc(t, env, started.MeetingID).HandoffOrder; slices.Contains(got, ids[0]) || len(got) != 2 {
		t.Fatalf("handoffOrder = %v, want the other two", got)
	}
}

func countOf(s []string, v string) int {
	n := 0
	for _, x := range s {
		if x == v {
			n++
		}
	}
	return n
}

func isChecked(m contract.Meeting, id string) bool { _, ok := m.CheckedIn[id]; return ok }
