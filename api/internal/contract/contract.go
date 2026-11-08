// Package contract holds the Go↔TS contract (ADR 0002): every HTTP request and
// response body, and every Firestore doc the browser reads. Types only.
// tygo turns this package into apps/hq/src/lib/contract.gen.ts; run `bun run gen`
// after any change and commit the output.
//
// Use only types tygo can map: no unexported fields, no methods that matter to
// the shape. For Firestore docs give each field matching `json` and `firestore`
// tags, since the browser reads the raw doc.
package contract

import "time"

// Health is the body of GET /api/health.
type Health struct {
	Status string `json:"status"`
	// Today is the current date in America/New_York (YYYY-MM-DD), the ID today's Meeting would have.
	Today string `json:"today"`
}

// ErrorCode is a machine-readable reason in an error body. The UI shows a
// message per code; add codes as screens need them.
type ErrorCode string

const (
	ErrorCodeUnauthenticated ErrorCode = "unauthenticated" // no valid credential (401)
	ErrorCodeForbidden       ErrorCode = "forbidden"       // valid credential, caller not accepted (403)
	ErrorCodeInternal        ErrorCode = "internal"        // server-side failure, e.g. Firestore unreachable (500)
	ErrorCodeBadRequest      ErrorCode = "bad_request"     // malformed or disallowed request body (400)
	ErrorCodeNotFound        ErrorCode = "not_found"       // the thing named in the path doesn't exist (404)
	ErrorCodeFixedRole       ErrorCode = "fixed_role"      // an Admin's role can't be changed from HQ (409)
	ErrorCodeInUse           ErrorCode = "in_use"          // can't delete: other records point at it (409)
	ErrorCodeDuplicateScout  ErrorCode = "duplicate_scout" // an active Scout already has this first + last name (409)
	ErrorCodeTooLate         ErrorCode = "too_late"        // past 22:00 America/New_York: today's Meeting can't be started (409)
	ErrorCodeSessionAsleep   ErrorCode = "session_asleep"  // Kiosk session idle 30 min; a Leader can wake it (401)
	ErrorCodeSessionExpired  ErrorCode = "session_expired" // Kiosk session past 22:00 or replaced by a newer unlock (401)
)

// ErrorBody is the body of every non-2xx API response.
type ErrorBody struct {
	Code ErrorCode `json:"code"`
}

// Role is a signed-in person's access level, stored in users/{uid}.role.
// Everyone starts as a Parent; the Admin promotes Leaders; only the bootstrap
// script sets admin (ADR 0003).
type Role string

const (
	RoleParent Role = "parent"
	RoleLeader Role = "leader"
	RoleAdmin  Role = "admin"
)

// User is the Firestore doc users/{uid}, created at first sign-in.
// Its owner can read it (firestore.rules); only Go writes it.
type User struct {
	Email       string    `json:"email" firestore:"email"`
	DisplayName string    `json:"displayName" firestore:"displayName"`
	Role        Role      `json:"role" firestore:"role"`
	CreatedAt   time.Time `json:"createdAt" firestore:"createdAt"`
}

// Me is the body of GET /api/me: who the caller is, so HQ can route them.
type Me struct {
	UID  string `json:"uid"`
	Role Role   `json:"role"`
}

// Event is the Firestore doc events/{id}: a non-Meeting troop activity
// Scouts RSVP to. Dates are YYYY-MM-DD with no time; endDate >= startDate.
type Event struct {
	Name      string `json:"name" firestore:"name"`
	StartDate string `json:"startDate" firestore:"startDate"`
	EndDate   string `json:"endDate" firestore:"endDate"`
	Location  string `json:"location,omitempty" firestore:"location,omitempty"`
	Cancelled bool   `json:"cancelled" firestore:"cancelled"`
}

// EventInput is the body of POST /api/events and PUT /api/events/{id}.
// Name is trimmed and required; Location is optional (empty removes it).
type EventInput struct {
	Name      string `json:"name"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Location  string `json:"location,omitempty"`
}

// EventCreated is the body of a 201 from POST /api/events.
type EventCreated struct {
	ID string `json:"id"`
}

// RSVPAnswer is a saved RSVP answer. No answer is null, never a third value.
type RSVPAnswer string

const (
	RSVPAnswerGoing    RSVPAnswer = "going"
	RSVPAnswerNotGoing RSVPAnswer = "notGoing"
)

// RSVPEntry is one change to an RSVP (ADR 0004). AttendanceID is the Kiosk
// check-in the answer was given in, or null for a Leader's change. A null
// Answer is a clear: the Kiosk asks the Scout again.
type RSVPEntry struct {
	Answer       *RSVPAnswer `json:"answer" firestore:"answer" tstype:"RSVPAnswer | null,required"`
	At           time.Time   `json:"at" firestore:"at"`
	By           string      `json:"by" firestore:"by"` // a Leader's uid or a kioskSessionId
	AttendanceID *string     `json:"attendanceId" firestore:"attendanceId" tstype:"string | null,required"`
}

// RSVP is the Firestore doc rsvps/{eventId}_{scoutId}. Answer always equals
// the last History entry's answer (Go keeps them consistent in one
// transaction). EventID and ScoutID repeat the doc ID's parts so the Event
// page, the Scout page and Undo check-in can query by either.
type RSVP struct {
	EventID string      `json:"eventId" firestore:"eventId"`
	ScoutID string      `json:"scoutId" firestore:"scoutId"`
	Answer  *RSVPAnswer `json:"answer" firestore:"answer" tstype:"RSVPAnswer | null,required"`
	History []RSVPEntry `json:"history" firestore:"history"`
}

// SetRSVP is the body of PUT /api/events/{eventId}/rsvps/{scoutId}, a
// Leader setting or (with null) clearing a Scout's RSVP.
type SetRSVP struct {
	Answer *RSVPAnswer `json:"answer" tstype:"RSVPAnswer | null,required"`
}

// UserSummary is one row of the Access tab.
type UserSummary struct {
	UID         string `json:"uid"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Role        Role   `json:"role"`
	CreatedAt   string `json:"createdAt"` // RFC 3339, first sign-in
}

// UserList is the body of GET /api/users (Admin only): everyone who has
// signed in, newest first.
type UserList struct {
	Users []UserSummary `json:"users"`
}

// SetRole is the body of PUT /api/users/{uid}/role (Admin only). Role is
// parent or leader; admin is never set from HQ.
type SetRole struct {
	Role Role `json:"role"`
}

// Scout is the Firestore doc scouts/{id}: one Scout on the Roster. Leaders
// read it live (firestore.rules); only Go writes it.
type Scout struct {
	FirstName string    `json:"firstName" firestore:"firstName"`
	LastName  string    `json:"lastName" firestore:"lastName"`
	Active    bool      `json:"active" firestore:"active"`
	CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
}

// AddScoutRequest is the body of POST /api/scouts.
type AddScoutRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// AddScoutResponse is the 201 body of POST /api/scouts.
type AddScoutResponse struct {
	ID string `json:"id"`
}

// Meeting is the Firestore doc meetings/{YYYY-MM-DD}: one Meeting, keyed by
// its local date (ADR 0004). It is also the Kiosk's live state, read through
// onSnapshot. There is no status field: a Meeting stops being live at 22:00.
// Go always writes the slices and the map (never null), so HQ can iterate them.
type Meeting struct {
	StartedAt time.Time `json:"startedAt" firestore:"startedAt"`
	StartedBy string    `json:"startedBy" firestore:"startedBy"` // uid of the Admin who first started it
	// HandoffOrder is every active Scout ID, shuffled once when the Meeting is created.
	HandoffOrder []string `json:"handoffOrder" firestore:"handoffOrder"`
	// NotFound is the Scout IDs the Kiosk couldn't find this Meeting.
	NotFound []string `json:"notFound" firestore:"notFound"`
	// CheckedIn maps scoutId to the attendanceId of their current (not undone) Attendance.
	CheckedIn map[string]string `json:"checkedIn" firestore:"checkedIn"`
}

// AttendanceMethod is how a Scout was checked in.
type AttendanceMethod string

const (
	AttendanceMethodNameList AttendanceMethod = "nameList" // tapped their name on the Kiosk
	AttendanceMethodHandoff  AttendanceMethod = "handoff"  // answered Yes when the Kiosk named them
	AttendanceMethodLeader   AttendanceMethod = "leader"   // a Leader checked them in
)

// Attendance is the Firestore doc meetings/{date}/attendance/{id}: the record
// that a Scout was present. Undo is a soft delete (ADR 0004).
type Attendance struct {
	ScoutID string           `json:"scoutId" firestore:"scoutId"`
	Method  AttendanceMethod `json:"method" firestore:"method"`
	At      time.Time        `json:"at" firestore:"at"`
	// By is the Leader's uid, or the kioskSessionId for a Kiosk check-in.
	By       string     `json:"by" firestore:"by"`
	UndoneAt *time.Time `json:"undoneAt,omitempty" firestore:"undoneAt,omitempty"`
	UndoneBy string     `json:"undoneBy,omitempty" firestore:"undoneBy,omitempty"`
}

// KioskSession is the Firestore doc kioskSessions/{id}: one Kiosk unlock
// (ADR 0003). Go checks it on every Kiosk call. It is valid while not
// replaced and now < min(22:00 on the Meeting date, lastActivityAt + 30 min).
type KioskSession struct {
	MeetingID      string    `json:"meetingId" firestore:"meetingId"`
	CreatedAt      time.Time `json:"createdAt" firestore:"createdAt"`
	CreatedBy      string    `json:"createdBy" firestore:"createdBy"` // the Admin's uid
	LastActivityAt time.Time `json:"lastActivityAt" firestore:"lastActivityAt"`
	// Replaced is set when a later Start issues a new session.
	Replaced bool `json:"replaced" firestore:"replaced"`
}

// StartMeetingResponse is the 200 body of POST /api/kiosk/start (Admin only):
// today's Meeting exists and a new Kiosk session replaced any other.
type StartMeetingResponse struct {
	MeetingID      string `json:"meetingId"`
	KioskSessionID string `json:"kioskSessionId"`
	// Token is a Firebase custom token (claims role: kiosk, kioskSessionId,
	// meetingId). The tablet signs in with it, which drops the Admin's session.
	Token string `json:"token"`
}

// KioskCheckInRequest is the body of POST /api/kiosk/check-in (Kiosk only).
// The Meeting is the Kiosk session's; method is nameList or handoff.
type KioskCheckInRequest struct {
	ScoutID string           `json:"scoutId"`
	Method  AttendanceMethod `json:"method"`
}

// LeaderCheckInRequest is the body of POST /api/meetings/{date}/check-ins
// (Leader): check a Scout in with method leader.
type LeaderCheckInRequest struct {
	ScoutID string `json:"scoutId"`
}

// LeaderSummary is one Leader's uid and name, so the Tonight tab can say
// which Leader checked a Scout in. No email.
type LeaderSummary struct {
	UID         string `json:"uid"`
	DisplayName string `json:"displayName"`
}

// LeaderList is the body of GET /api/leaders (any Leader): every Leader and the Admin.
type LeaderList struct {
	Leaders []LeaderSummary `json:"leaders"`
}

// KioskNotFoundRequest is the body of POST /api/kiosk/not-found (Kiosk only):
// the Kiosk couldn't find this Scout. The Meeting is the Kiosk session's.
type KioskNotFoundRequest struct {
	ScoutID string `json:"scoutId"`
}

// KioskSetRSVPRequest is the body of PUT /api/kiosk/rsvps (Kiosk only): the
// Scout's answer to an Event, given right after they checked in. Only going
// or notGoing; "ask me next week" and "not sure" send nothing. The Scout must
// have been checked in by this Kiosk session.
type KioskSetRSVPRequest struct {
	EventID string     `json:"eventId"`
	ScoutID string     `json:"scoutId"`
	Answer  RSVPAnswer `json:"answer"`
}

// RSVPRollbackItem is one answer Undo check-in would roll back: the RSVP's
// current answer (From) and what it reverts to (To, null when no earlier answer remains).
type RSVPRollbackItem struct {
	EventID   string      `json:"eventId"`
	EventName string      `json:"eventName"`
	From      *RSVPAnswer `json:"from" tstype:"RSVPAnswer | null,required"`
	To        *RSVPAnswer `json:"to" tstype:"RSVPAnswer | null,required"`
}

// RSVPRollbackPreview is the body of GET /api/meetings/{date}/attendance/{id}/rsvp-rollback
// (Leader): the answers Undo would change, so the confirm can name them.
// Entries a Leader's later edit already superseded change nothing and aren't listed.
type RSVPRollbackPreview struct {
	Items []RSVPRollbackItem `json:"items"`
}

// CheckInResponse is the 200 body of a check-in: the Scout's Attendance,
// new or (for a repeat check-in) the existing one.
type CheckInResponse struct {
	AttendanceID string `json:"attendanceId"`
}
