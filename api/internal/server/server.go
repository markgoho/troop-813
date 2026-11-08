// Package server builds the API's HTTP handler: the route table and its
// dependencies. main wires real dependencies; testenv wires emulator-backed ones.
package server

import (
	"log/slog"
	"net/http"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/auth"

	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
)

// Deps is everything a handler may use. Tests leave unused fields zero.
type Deps struct {
	Now           clock.Clock // nil means clock.Real
	Firestore     *firestore.Client
	Auth          *auth.Client
	Logger        *slog.Logger // log IDs, never Scout names
	Authenticator router.Authenticator
}

// New returns the API handler.
func New(d Deps) http.Handler {
	return clock.Middleware(d.Now, Routes(d))
}

// Routes mounts every route. Every route says who may call it (see router).
func Routes(d Deps) *router.Router {
	r := router.New(d.Authenticator)
	r.Open("GET /api/health", "liveness probe and the HQ smoke check; reveals only today's date", health)
	return r
}

func health(w http.ResponseWriter, r *http.Request) {
	router.WriteJSON(w, http.StatusOK, contract.Health{
		Status: "ok",
		Today:  clock.LocalDate(clock.Now(r.Context())),
	})
}
