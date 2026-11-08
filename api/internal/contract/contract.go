// Package contract holds the Go↔TS contract (ADR 0002): every HTTP request and
// response body, and every Firestore doc the browser reads. Types only.
// tygo turns this package into apps/hq/src/lib/contract.gen.ts; run `bun run gen`
// after any change and commit the output.
//
// Use only types tygo can map: no unexported fields, no methods that matter to
// the shape. For Firestore docs give each field matching `json` and `firestore`
// tags, since the browser reads the raw doc.
package contract

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
)

// ErrorBody is the body of every non-2xx API response.
type ErrorBody struct {
	Code ErrorCode `json:"code"`
}
