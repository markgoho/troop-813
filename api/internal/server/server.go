// Package server builds the API's HTTP handler: the route table and its
// dependencies. main wires real dependencies; testenv wires emulator-backed ones.
package server

import (
	"log/slog"
	"net/http"

	"cloud.google.com/go/firestore"
	fbauth "firebase.google.com/go/v4/auth"

	"github.com/markgoho/troop-813/api/internal/auth"
	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
)

// Deps is everything a handler may use. Tests leave unused fields zero.
type Deps struct {
	Now       clock.Clock // nil means clock.Real
	Firestore *firestore.Client
	Auth      *fbauth.Client
	Logger    *slog.Logger // log IDs, never Scout names
	// Authenticator resolves callers for Gated routes. Nil means auth.New
	// over Auth and Firestore; tests may swap in another.
	Authenticator router.Authenticator
}

// New returns the API handler.
func New(d Deps) http.Handler {
	return clock.Middleware(d.Now, Routes(d))
}

// Routes mounts every route. Every route says who may call it (see router).
func Routes(d Deps) *router.Router {
	authn := d.Authenticator
	if authn == nil {
		authn = auth.New(d.Auth, d.Firestore, d.Logger)
	}
	r := router.New(authn)
	r.Open("GET /api/health", "liveness probe and the HQ smoke check; reveals only today's date", health)
	r.Gated("GET /api/me", []router.Accept{router.Leader, router.Parent}, me)
	r.Gated("POST /api/scouts", []router.Accept{router.Leader}, addScout(d.Firestore, logger(d)))
	return r
}

// me says who the caller is, so HQ can route them. Calling it is also what
// creates users/{uid} at first sign-in (auth does that for any Gated route).
func me(w http.ResponseWriter, r *http.Request) {
	c := router.CallerFrom(r.Context())
	role := contract.RoleParent
	if l, ok := c.(auth.Leader); ok {
		role = contract.RoleLeader
		if l.IsAdmin {
			role = contract.RoleAdmin
		}
	}
	router.WriteJSON(w, http.StatusOK, contract.Me{UID: auth.UID(c), Role: role})
}

// logger is d.Logger, or a discarding one when unset (tests that only walk routes).
func logger(d Deps) *slog.Logger {
	if d.Logger == nil {
		return slog.New(slog.DiscardHandler)
	}
	return d.Logger
}

func health(w http.ResponseWriter, r *http.Request) {
	router.WriteJSON(w, http.StatusOK, contract.Health{
		Status: "ok",
		Today:  clock.LocalDate(clock.Now(r.Context())),
	})
}
