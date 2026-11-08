// Package clock is the one seam the API reads the current instant through.
// golangci's forbidigo rule bans time.Now elsewhere, so every time rule
// (the 22:00 cutoff, the 30-minute idle lock, "today's Meeting") can be
// driven by a test.
//
// Every time rule is in America/New_York, and Meeting IDs are local dates.
//
// Token verification does not read this clock: the Firebase Admin SDK checks
// ID token expiry against wall time, so faking "now" for business rules never
// breaks sign-in in tests.
package clock

import (
	"context"
	"net/http"
	"time"

	// Embeds zone data so America/New_York loads on a distroless Cloud Run image.
	_ "time/tzdata"
)

// Clock returns the current instant.
type Clock = func() time.Time

// Real is the wall clock: the only sanctioned use of time.Now in the API.
//
//nolint:forbidigo // the seam's own default
var Real Clock = time.Now

// NY is the troop's time zone, used for every date and cutoff.
var NY = mustLoad("America/New_York")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// LocalDate is t's date in America/New_York as YYYY-MM-DD, the format of a Meeting ID.
func LocalDate(t time.Time) string {
	return t.In(NY).Format(time.DateOnly)
}

// MeetingEnds is 22:00 America/New_York on a Meeting's date (its ID,
// YYYY-MM-DD). From then on the Meeting can't be started and its Kiosk
// sessions are over (ADR 0003).
func MeetingEnds(meetingID string) (time.Time, error) {
	d, err := time.ParseInLocation(time.DateOnly, meetingID, NY)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(d.Year(), d.Month(), d.Day(), 22, 0, 0, 0, NY), nil
}

type contextKey struct{}

// Into returns ctx carrying now, for Now to read back.
func Into(ctx context.Context, now Clock) context.Context {
	if now == nil {
		now = Real
	}
	return context.WithValue(ctx, contextKey{}, now)
}

// Now is the current instant from the Clock on ctx, or Real if none was set.
func Now(ctx context.Context) time.Time {
	if now, ok := ctx.Value(contextKey{}).(Clock); ok {
		return now()
	}
	return Real()
}

// Middleware puts now on every request's context.
func Middleware(now Clock, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(Into(r.Context(), now)))
	})
}
