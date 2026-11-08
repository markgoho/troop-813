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
