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
	ErrorCodeDuplicateScout  ErrorCode = "duplicate_scout" // an active Scout already has this first + last name (409)
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
