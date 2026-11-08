// Package server builds the API's HTTP handler: the route table and its
// dependencies. main wires real dependencies; testenv wires emulator-backed ones.
package server

import (
	"log/slog"
	"net/http"

	"cloud.google.com/go/firestore"
	fbauth "firebase.google.com/go/v4/auth"

	"github.com/markgoho/troop-813/api/internal/access"
	"github.com/markgoho/troop-813/api/internal/auth"
	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/events"
	"github.com/markgoho/troop-813/api/internal/meeting"
	"github.com/markgoho/troop-813/api/internal/roster"
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
	acc := access.Handlers{FS: d.Firestore}
	r.Gated("GET /api/users", []router.Accept{router.Admin}, acc.List)
	r.Gated("PUT /api/users/{uid}/role", []router.Accept{router.Admin}, acc.SetRole)
	ev := events.Handlers{FS: d.Firestore}
	leader := []router.Accept{router.Leader}
	r.Gated("POST /api/events", leader, ev.Create)
	r.Gated("PUT /api/events/{id}", leader, ev.Update)
	r.Gated("POST /api/events/{id}/cancel", leader, ev.Cancel)
	r.Gated("POST /api/events/{id}/uncancel", leader, ev.Uncancel)
	r.Gated("DELETE /api/events/{id}", leader, ev.Delete)
	r.Gated("PUT /api/events/{id}/rsvps/{scoutId}", leader, ev.SetRSVP)
	ros := roster.Handlers{FS: d.Firestore, Logger: d.Logger}
	r.Gated("GET /api/scouts/attendance", leader, ros.Attendance)
	r.Gated("GET /api/scouts/{id}/history", leader, ros.History)
	r.Gated("POST /api/scouts", leader, ros.Add)
	r.Gated("PUT /api/scouts/{id}", leader, ros.Edit)
	r.Gated("POST /api/scouts/{id}/deactivate", leader, ros.Deactivate)
	r.Gated("POST /api/scouts/{id}/reactivate", leader, ros.Reactivate)
	r.Gated("DELETE /api/scouts/{id}", leader, ros.Delete)
	mtg := meeting.Handlers{FS: d.Firestore, Auth: d.Auth, Logger: d.Logger}
	r.Gated("POST /api/kiosk/start", []router.Accept{router.Admin}, mtg.Start)
	r.Gated("POST /api/kiosk/check-in", []router.Accept{router.Kiosk}, mtg.KioskCheckIn)
	r.Gated("POST /api/meetings/{date}/check-ins", leader, mtg.LeaderCheckIn)
	r.Gated("POST /api/meetings/{date}/attendance/{id}/undo", leader, mtg.UndoCheckIn)
	r.Gated("GET /api/leaders", leader, acc.Leaders)
	r.Gated("POST /api/kiosk/not-found", []router.Accept{router.Kiosk}, mtg.KioskNotFound)
	r.Gated("POST /api/kiosk/wake", leader, mtg.Wake)
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

func health(w http.ResponseWriter, r *http.Request) {
	router.WriteJSON(w, http.StatusOK, contract.Health{
		Status: "ok",
		Today:  clock.LocalDate(clock.Now(r.Context())),
	})
}
