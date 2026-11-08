package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

// historyScout writes a Scout created at noon New York time on date.
func historyScout(t *testing.T, env *testenv.Env, first, createdOn string) string {
	t.Helper()
	at, err := time.Parse(time.RFC3339, createdOn+"T17:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	ref, _, err := env.Firestore.Collection("scouts").Add(context.Background(), contract.Scout{
		FirstName: first, LastName: "Quill", Active: true, CreatedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ref.ID
}

func historyMeeting(t *testing.T, env *testenv.Env, date string) {
	t.Helper()
	_, err := env.Firestore.Collection("meetings").Doc(date).Set(context.Background(), contract.Meeting{
		StartedAt: testenv.DefaultNow, StartedBy: "u-admin",
		HandoffOrder: []string{}, NotFound: []string{}, CheckedIn: map[string]string{},
	})
	if err != nil {
		t.Fatal(err)
	}
}

// historyAttendance writes an Attendance for the Scout; undone makes it a soft-deleted one.
func historyAttendance(t *testing.T, env *testenv.Env, date, scoutID string, undone bool) {
	t.Helper()
	a := contract.Attendance{ScoutID: scoutID, Method: contract.AttendanceMethodLeader, At: testenv.DefaultNow, By: "u-leader"}
	if undone {
		at := testenv.DefaultNow
		a.UndoneAt, a.UndoneBy = &at, "u-leader"
	}
	if _, _, err := env.Firestore.Collection("meetings").Doc(date).Collection("attendance").Add(context.Background(), a); err != nil {
		t.Fatal(err)
	}
}

func getJSON(t *testing.T, env *testenv.Env, path, token string, out any) int {
	t.Helper()
	resp := env.Request(t, http.MethodGet, path, token, nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	return resp.StatusCode
}

func scoutHistory(t *testing.T, env *testenv.Env, token, scoutID string) (int, contract.ScoutHistory) {
	t.Helper()
	var h contract.ScoutHistory
	return getJSON(t, env, "/api/scouts/"+scoutID+"/history", token, &h), h
}

func presence(rows ...contract.MeetingPresence) []contract.MeetingPresence {
	return rows
}

func row(date string, present bool) contract.MeetingPresence {
	return contract.MeetingPresence{Date: date, Present: present}
}

func TestHistoryCountsMeetingsSinceTheScoutJoined(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	scout := historyScout(t, env, "Avery", "2026-10-12")
	for _, d := range []string{"2026-10-05", "2026-10-12", "2026-10-19", "2026-10-26"} {
		historyMeeting(t, env, d)
	}
	historyAttendance(t, env, "2026-10-12", scout, false)
	historyAttendance(t, env, "2026-10-26", scout, false)

	status, h := scoutHistory(t, env, leader, scout)

	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	want := contract.ScoutHistory{
		Since: "2026-10-12", Present: 2, Total: 3,
		Meetings: presence(row("2026-10-26", true), row("2026-10-19", false), row("2026-10-12", true)),
	}
	if !reflect.DeepEqual(h, want) {
		t.Fatalf("history = %+v, want %+v", h, want)
	}
}

func TestHistoryStartsAtABackfilledAttendanceBeforeCreatedAt(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	scout := historyScout(t, env, "Avery", "2026-10-19")
	for _, d := range []string{"2026-10-05", "2026-10-12", "2026-10-19"} {
		historyMeeting(t, env, d)
	}
	historyAttendance(t, env, "2026-10-05", scout, false)

	_, h := scoutHistory(t, env, leader, scout)

	want := contract.ScoutHistory{
		Since: "2026-10-05", Present: 1, Total: 3,
		Meetings: presence(row("2026-10-19", false), row("2026-10-12", false), row("2026-10-05", true)),
	}
	if !reflect.DeepEqual(h, want) {
		t.Fatalf("history = %+v, want %+v", h, want)
	}
}

func TestHistoryIgnoresUndoneAttendance(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	scout := historyScout(t, env, "Avery", "2026-10-12")
	for _, d := range []string{"2026-10-05", "2026-10-12"} {
		historyMeeting(t, env, d)
	}
	// An undone record neither counts as present nor pulls the start back.
	historyAttendance(t, env, "2026-10-05", scout, true)

	_, h := scoutHistory(t, env, leader, scout)

	want := contract.ScoutHistory{Since: "2026-10-12", Present: 0, Total: 1, Meetings: presence(row("2026-10-12", false))}
	if !reflect.DeepEqual(h, want) {
		t.Fatalf("history = %+v, want %+v", h, want)
	}
}

func TestHistoryIgnoresOtherScouts(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	avery := historyScout(t, env, "Avery", "2026-10-12")
	rowan := historyScout(t, env, "Rowan", "2026-10-12")
	historyMeeting(t, env, "2026-10-12")
	historyAttendance(t, env, "2026-10-12", rowan, false)

	_, h := scoutHistory(t, env, leader, avery)

	if h.Present != 0 || h.Total != 1 {
		t.Fatalf("history = %+v, want 0 of 1", h)
	}
}

func TestHistoryOfAScoutWithNoMeetingsYetIsEmpty(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	scout := historyScout(t, env, "Avery", "2026-10-12")
	historyMeeting(t, env, "2026-10-05")

	_, h := scoutHistory(t, env, leader, scout)

	want := contract.ScoutHistory{Since: "2026-10-12", Meetings: []contract.MeetingPresence{}}
	if !reflect.DeepEqual(h, want) {
		t.Fatalf("history = %+v, want %+v", h, want)
	}
}

func TestHistoryOfAnUnknownScoutIsNotFound(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)

	status, _ := scoutHistory(t, env, leader, "nope")

	if status != http.StatusNotFound {
		t.Fatalf("status %d, want 404", status)
	}
}

func TestAttendanceCountsForTheRoster(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	avery := historyScout(t, env, "Avery", "2026-10-05")
	rowan := historyScout(t, env, "Rowan", "2026-10-19")
	for _, d := range []string{"2026-10-05", "2026-10-12", "2026-10-19"} {
		historyMeeting(t, env, d)
	}
	historyAttendance(t, env, "2026-10-05", avery, false)
	historyAttendance(t, env, "2026-10-12", avery, true)
	historyAttendance(t, env, "2026-10-19", avery, false)
	historyAttendance(t, env, "2026-10-19", rowan, false)

	var got contract.ScoutAttendance
	if status := getJSON(t, env, "/api/scouts/attendance", leader, &got); status != http.StatusOK {
		t.Fatalf("status %d", status)
	}

	want := map[string]contract.AttendanceCount{
		avery: {Present: 2, Total: 3},
		rowan: {Present: 1, Total: 1},
	}
	if !reflect.DeepEqual(got.Scouts, want) {
		t.Fatalf("counts = %+v, want %+v", got.Scouts, want)
	}
}
