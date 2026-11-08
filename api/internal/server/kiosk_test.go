package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

// seedScouts writes active Scouts (first names only matter) and returns their IDs.
func seedScouts(t *testing.T, env *testenv.Env, active bool, firstNames ...string) []string {
	t.Helper()
	ids := make([]string, 0, len(firstNames))
	for _, first := range firstNames {
		ref, _, err := env.Firestore.Collection("scouts").Add(context.Background(), contract.Scout{
			FirstName: first, LastName: "Quill", Active: active, CreatedAt: testenv.DefaultNow,
		})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, ref.ID)
	}
	return ids
}

// startMeeting calls POST /api/kiosk/start. On 200 it returns the body; otherwise the error code.
func startMeeting(t *testing.T, env *testenv.Env, token string) (int, contract.StartMeetingResponse, contract.ErrorCode) {
	t.Helper()
	resp := env.Request(t, http.MethodPost, "/api/kiosk/start", token, nil)
	defer func() { _ = resp.Body.Close() }()
	var out contract.StartMeetingResponse
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, out, errorCode(t, resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode start: %v", err)
	}
	return resp.StatusCode, out, ""
}

// unlockKiosk starts today's Meeting as the Admin and signs in with the
// returned custom token, as the tablet does. It returns the Kiosk ID token.
func unlockKiosk(t *testing.T, env *testenv.Env) (string, contract.StartMeetingResponse) {
	t.Helper()
	admin := env.RoleToken(t, "u-admin", contract.RoleAdmin)
	status, started, code := startMeeting(t, env, admin)
	if status != http.StatusOK {
		t.Fatalf("start: %d %q", status, code)
	}
	return env.SignInWithCustomToken(t, started.Token), started
}

// checkIn calls POST /api/kiosk/check-in.
func checkIn(t *testing.T, env *testenv.Env, token, scoutID string, method contract.AttendanceMethod) (int, string, contract.ErrorCode) {
	t.Helper()
	resp := env.Request(t, http.MethodPost, "/api/kiosk/check-in", token, contract.KioskCheckInRequest{ScoutID: scoutID, Method: method})
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

func errorCode(t *testing.T, resp *http.Response) contract.ErrorCode {
	t.Helper()
	var e contract.ErrorBody
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		t.Fatalf("decode error body (status %d): %v", resp.StatusCode, err)
	}
	return e.Code
}

func meetingDoc(t *testing.T, env *testenv.Env, id string) contract.Meeting {
	t.Helper()
	snap, err := env.Firestore.Collection("meetings").Doc(id).Get(context.Background())
	if err != nil {
		t.Fatalf("get meetings/%s: %v", id, err)
	}
	var m contract.Meeting
	if err := snap.DataTo(&m); err != nil {
		t.Fatalf("decode meetings/%s: %v", id, err)
	}
	return m
}

func sessionDoc(t *testing.T, env *testenv.Env, id string) contract.KioskSession {
	t.Helper()
	snap, err := env.Firestore.Collection("kioskSessions").Doc(id).Get(context.Background())
	if err != nil {
		t.Fatalf("get kioskSessions/%s: %v", id, err)
	}
	var s contract.KioskSession
	if err := snap.DataTo(&s); err != nil {
		t.Fatalf("decode kioskSessions/%s: %v", id, err)
	}
	return s
}

func attendanceDocs(t *testing.T, env *testenv.Env, meetingID string) map[string]contract.Attendance {
	t.Helper()
	snaps, err := env.Firestore.Collection("meetings").Doc(meetingID).Collection("attendance").Documents(context.Background()).GetAll()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]contract.Attendance{}
	for _, s := range snaps {
		var a contract.Attendance
		if err := s.DataTo(&a); err != nil {
			t.Fatal(err)
		}
		out[s.Ref.ID] = a
	}
	return out
}

func TestTheAdminStartsTodaysMeeting(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	active := seedScouts(t, env, true, "Avery", "Rowan", "Juniper")
	seedScouts(t, env, false, "Marlow")
	admin := env.RoleToken(t, "u-admin", contract.RoleAdmin)

	status, started, code := startMeeting(t, env, admin)

	if status != http.StatusOK || started.MeetingID != "2026-11-09" || started.Token == "" || started.KioskSessionID == "" {
		t.Fatalf("start = %d %q %+v, want 200 for 2026-11-09 with a token and session", status, code, started)
	}
	raw, err := env.Firestore.Collection("meetings").Doc("2026-11-09").Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// HQ iterates these: they must be stored as an empty array and map, never null.
	if nf, ok := raw.Data()["notFound"].([]any); !ok || len(nf) != 0 {
		t.Errorf("stored notFound = %#v, want []", raw.Data()["notFound"])
	}
	if ci, ok := raw.Data()["checkedIn"].(map[string]any); !ok || len(ci) != 0 {
		t.Errorf("stored checkedIn = %#v, want {}", raw.Data()["checkedIn"])
	}
	m := meetingDoc(t, env, "2026-11-09")
	if m.StartedBy != "u-admin" || !m.StartedAt.Equal(testenv.DefaultNow) {
		t.Errorf("meeting started %v by %q, want %v by u-admin", m.StartedAt, m.StartedBy, testenv.DefaultNow)
	}
	got := slices.Sorted(slices.Values(m.HandoffOrder))
	if want := slices.Sorted(slices.Values(active)); !slices.Equal(got, want) {
		t.Errorf("handoffOrder = %v, want the active Scouts %v in some order", m.HandoffOrder, active)
	}

	s := sessionDoc(t, env, started.KioskSessionID)
	if s.MeetingID != "2026-11-09" || s.CreatedBy != "u-admin" || s.Replaced || !s.LastActivityAt.Equal(testenv.DefaultNow) {
		t.Errorf("session = %+v, want live for 2026-11-09, by u-admin, active now", s)
	}

	tok, err := env.Auth.VerifyIDToken(context.Background(), env.SignInWithCustomToken(t, started.Token))
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"role": "kiosk", "kioskSessionId": started.KioskSessionID, "meetingId": "2026-11-09"} {
		if tok.Claims[k] != want {
			t.Errorf("claim %s = %v, want %s", k, tok.Claims[k], want)
		}
	}
}

func TestOnlyTheAdminStartsAMeeting(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	for name, c := range map[string]struct {
		token  string
		status int
	}{
		"anonymous": {"", http.StatusUnauthorized},
		"parent":    {env.RoleToken(t, "u-parent", contract.RoleParent), http.StatusForbidden},
		"leader":    {env.RoleToken(t, "u-leader", contract.RoleLeader), http.StatusForbidden},
	} {
		if status, _, _ := startMeeting(t, env, c.token); status != c.status {
			t.Errorf("%s: status %d, want %d", name, status, c.status)
		}
	}
	if _, err := env.Firestore.Collection("meetings").Doc("2026-11-09").Get(context.Background()); err == nil {
		t.Fatal("a refused Start created the Meeting")
	}
}

func TestStartWorksOnAnyWeekday(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	env.Clock.Set(time.Date(2026, 11, 12, 18, 30, 0, 0, clock.NY)) // a Thursday
	admin := env.RoleToken(t, "u-admin", contract.RoleAdmin)

	if status, started, code := startMeeting(t, env, admin); status != http.StatusOK || started.MeetingID != "2026-11-12" {
		t.Fatalf("start = %d %q %+v, want 200 for 2026-11-12", status, code, started)
	}
}

func TestStartIsRefusedFrom2200(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	admin := env.RoleToken(t, "u-admin", contract.RoleAdmin)

	env.Clock.Set(time.Date(2026, 11, 9, 22, 0, 0, 0, clock.NY))
	if status, _, code := startMeeting(t, env, admin); status != http.StatusConflict || code != contract.ErrorCodeTooLate {
		t.Fatalf("start at 22:00 = %d %q, want 409 too_late", status, code)
	}
	if _, err := env.Firestore.Collection("meetings").Doc("2026-11-09").Get(context.Background()); err == nil {
		t.Fatal("a late Start created the Meeting")
	}

	env.Clock.Set(time.Date(2026, 11, 9, 21, 59, 59, 0, clock.NY))
	if status, _, code := startMeeting(t, env, admin); status != http.StatusOK {
		t.Fatalf("start at 21:59:59 = %d %q, want 200", status, code)
	}
}

func TestStartingAgainKeepsTheMeetingAndReplacesTheSession(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery", "Rowan", "Juniper", "Sage", "Wren", "Ellis")
	first, started1 := unlockKiosk(t, env)
	if status, _, code := checkIn(t, env, first, scouts[2], contract.AttendanceMethodNameList); status != http.StatusOK {
		t.Fatalf("check-in: %d %q", status, code)
	}
	before := meetingDoc(t, env, "2026-11-09")

	env.Clock.Advance(15 * time.Minute)
	second, started2 := unlockKiosk(t, env)

	after := meetingDoc(t, env, "2026-11-09")
	if !slices.Equal(after.HandoffOrder, before.HandoffOrder) || !after.StartedAt.Equal(before.StartedAt) || len(after.CheckedIn) != 1 || after.CheckedIn[scouts[2]] != before.CheckedIn[scouts[2]] {
		t.Fatalf("meeting changed on a second Start:\nbefore %+v\nafter  %+v", before, after)
	}
	if started2.KioskSessionID == started1.KioskSessionID {
		t.Fatal("second Start reused the session")
	}
	if !sessionDoc(t, env, started1.KioskSessionID).Replaced {
		t.Error("first session not marked replaced")
	}
	if status, _, code := checkIn(t, env, first, scouts[0], contract.AttendanceMethodNameList); status != http.StatusUnauthorized || code != contract.ErrorCodeSessionExpired {
		t.Errorf("replaced session check-in = %d %q, want 401 session_expired", status, code)
	}
	if status, _, code := checkIn(t, env, second, scouts[0], contract.AttendanceMethodNameList); status != http.StatusOK {
		t.Errorf("new session check-in = %d %q, want 200", status, code)
	}
}

func TestAKioskCheckInRecordsAttendance(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery", "Rowan")
	kiosk, started := unlockKiosk(t, env)
	env.Clock.Advance(10 * time.Minute)

	status, id, code := checkIn(t, env, kiosk, scouts[1], contract.AttendanceMethodNameList)

	if status != http.StatusOK || id == "" {
		t.Fatalf("check-in = %d %q, want 200 with an attendanceId", status, code)
	}
	now := testenv.DefaultNow.Add(10 * time.Minute)
	a, ok := attendanceDocs(t, env, "2026-11-09")[id]
	if !ok || a.ScoutID != scouts[1] || a.Method != contract.AttendanceMethodNameList || !a.At.Equal(now) || a.By != started.KioskSessionID || a.UndoneAt != nil {
		t.Errorf("attendance/%s = %+v (found %v), want Rowan by name list at %v by the session", id, a, ok, now)
	}
	if m := meetingDoc(t, env, "2026-11-09"); m.CheckedIn[scouts[1]] != id || len(m.CheckedIn) != 1 {
		t.Errorf("checkedIn = %v, want only %s → %s", m.CheckedIn, scouts[1], id)
	}
	if s := sessionDoc(t, env, started.KioskSessionID); !s.LastActivityAt.Equal(now) {
		t.Errorf("lastActivityAt = %v, want %v", s.LastActivityAt, now)
	}
	if status, _, _ := checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodHandoff); status != http.StatusOK {
		t.Errorf("handoff check-in = %d, want 200", status)
	}
}

func TestARepeatCheckInReturnsTheExistingAttendance(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	kiosk, _ := unlockKiosk(t, env)
	_, first, _ := checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList)

	status, again, code := checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodHandoff)

	if status != http.StatusOK || again != first {
		t.Fatalf("repeat check-in = %d %q %q, want 200 with %q", status, code, again, first)
	}
	if n := len(attendanceDocs(t, env, "2026-11-09")); n != 1 {
		t.Fatalf("%d attendance records, want 1", n)
	}
}

func TestConcurrentCheckInsOfOneScoutYieldOneAttendance(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	kiosk, _ := unlockKiosk(t, env)

	const n = 5
	ids := make(chan string, n)
	for range n {
		go func() {
			status, id, code := checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList)
			if status != http.StatusOK {
				t.Errorf("concurrent check-in = %d %q", status, code)
			}
			ids <- id
		}()
	}
	got := map[string]bool{}
	for range n {
		got[<-ids] = true
	}

	records := attendanceDocs(t, env, "2026-11-09")
	if len(records) != 1 || len(got) != 1 {
		t.Fatalf("%d attendance records, %d distinct ids returned (%v), want 1 and 1", len(records), len(got), got)
	}
	for id := range got {
		if _, ok := records[id]; !ok {
			t.Fatalf("returned id %s is not the stored record", id)
		}
	}
}

func TestKioskCheckInRejectsBadRequests(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	kiosk, _ := unlockKiosk(t, env)

	for name, c := range map[string]struct {
		scoutID string
		method  contract.AttendanceMethod
		status  int
		code    contract.ErrorCode
	}{
		"unknown scout":    {"no-such-scout", contract.AttendanceMethodNameList, http.StatusNotFound, contract.ErrorCodeNotFound},
		"no scout":         {"", contract.AttendanceMethodNameList, http.StatusBadRequest, contract.ErrorCodeBadRequest},
		"leader method":    {scouts[0], contract.AttendanceMethodLeader, http.StatusBadRequest, contract.ErrorCodeBadRequest},
		"no method":        {scouts[0], "", http.StatusBadRequest, contract.ErrorCodeBadRequest},
		"made-up method":   {scouts[0], "teleport", http.StatusBadRequest, contract.ErrorCodeBadRequest},
		"slash in scoutId": {"a/b", contract.AttendanceMethodNameList, http.StatusNotFound, contract.ErrorCodeNotFound},
	} {
		if status, _, code := checkIn(t, env, kiosk, c.scoutID, c.method); status != c.status || code != c.code {
			t.Errorf("%s: %d %q, want %d %q", name, status, code, c.status, c.code)
		}
	}
	if n := len(attendanceDocs(t, env, "2026-11-09")); n != 0 {
		t.Fatalf("%d attendance records after bad requests, want 0", n)
	}
}

func TestAKioskSessionEndsAt2200(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery", "Rowan")
	env.Clock.Set(time.Date(2026, 11, 9, 21, 45, 0, 0, clock.NY))
	kiosk, _ := unlockKiosk(t, env)

	env.Clock.Set(time.Date(2026, 11, 9, 21, 59, 59, 0, clock.NY))
	if status, _, code := checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList); status != http.StatusOK {
		t.Fatalf("check-in at 21:59:59 = %d %q, want 200", status, code)
	}
	env.Clock.Set(time.Date(2026, 11, 9, 22, 0, 0, 0, clock.NY))
	if status, _, code := checkIn(t, env, kiosk, scouts[1], contract.AttendanceMethodNameList); status != http.StatusUnauthorized || code != contract.ErrorCodeSessionExpired {
		t.Fatalf("check-in at 22:00 = %d %q, want 401 session_expired", status, code)
	}
}

func TestAKioskSessionFallsAsleepAfter30IdleMinutes(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery", "Rowan", "Juniper")
	kiosk, _ := unlockKiosk(t, env)

	// Each Kiosk write restarts the 30 minutes.
	env.Clock.Advance(29*time.Minute + 59*time.Second)
	if status, _, code := checkIn(t, env, kiosk, scouts[0], contract.AttendanceMethodNameList); status != http.StatusOK {
		t.Fatalf("check-in after 29:59 idle = %d %q, want 200", status, code)
	}
	env.Clock.Advance(29 * time.Minute)
	if status, _, code := checkIn(t, env, kiosk, scouts[1], contract.AttendanceMethodNameList); status != http.StatusOK {
		t.Fatalf("check-in 29 min after the last one = %d %q, want 200", status, code)
	}
	env.Clock.Advance(30 * time.Minute)
	if status, _, code := checkIn(t, env, kiosk, scouts[2], contract.AttendanceMethodNameList); status != http.StatusUnauthorized || code != contract.ErrorCodeSessionAsleep {
		t.Fatalf("check-in after 30 min idle = %d %q, want 401 session_asleep", status, code)
	}
}

func TestAKioskTokenWithoutALiveSessionIsRejected(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	scouts := seedScouts(t, env, true, "Avery")
	_, started := unlockKiosk(t, env)

	for name, claims := range map[string]map[string]any{
		"unknown session": {"role": "kiosk", "kioskSessionId": "no-such-session", "meetingId": "2026-11-09"},
		"other meeting":   {"role": "kiosk", "kioskSessionId": started.KioskSessionID, "meetingId": "2026-11-10"},
		"no session id":   {"role": "kiosk", "meetingId": "2026-11-09"},
	} {
		forged := env.IDToken(t, "kiosk-forged", claims)
		if status, _, _ := checkIn(t, env, forged, scouts[0], contract.AttendanceMethodNameList); status != http.StatusUnauthorized {
			t.Errorf("%s: status %d, want 401", name, status)
		}
	}
}

func TestCheckInLogsIDsNotNames(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	ref, _, err := env.Firestore.Collection("scouts").Add(context.Background(), contract.Scout{
		FirstName: "Peregrine", LastName: "Thistlewood", Active: true, CreatedAt: testenv.DefaultNow,
	})
	if err != nil {
		t.Fatal(err)
	}
	kiosk, started := unlockKiosk(t, env)
	_, id, _ := checkIn(t, env, kiosk, ref.ID, contract.AttendanceMethodNameList)

	logs := env.Logs()
	if strings.Contains(logs, "Peregrine") || strings.Contains(logs, "Thistlewood") {
		t.Fatalf("logs carry the Scout's name:\n%s", logs)
	}
	for _, want := range []string{id, ref.ID, started.KioskSessionID, "2026-11-09"} {
		if !strings.Contains(logs, want) {
			t.Errorf("logs don't carry %s:\n%s", want, logs)
		}
	}
}
