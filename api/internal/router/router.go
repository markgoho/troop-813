// Package router mounts every API route and makes each one declare who may
// call it. A route is either Open (with a written reason) or Gated (with a
// non-empty list of accepted callers); there is no third way in, and a missing
// declaration panics at startup.
//
// Turning a request into a Caller is the auth module's job (#20): it implements
// Authenticator. This package only applies the declaration.
package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/markgoho/troop-813/api/internal/contract"
)

// Accept names a kind of caller a route accepts.
type Accept string

const (
	Leader Accept = "leader" // any Leader, the Admin included
	Admin  Accept = "admin"  // only the Admin
	Kiosk  Accept = "kiosk"  // a valid Kiosk session
	Parent Accept = "parent" // signed in, not a Leader
)

// Caller is who made a request. Is reports whether the caller counts as a;
// an Admin is both Admin and Leader.
type Caller interface {
	Is(a Accept) bool
}

// ErrUnauthenticated is what an Authenticator returns (or wraps) when the
// request carries no valid credential. Gated answers it with 401; any other
// error means the caller couldn't be resolved (e.g. Firestore down) and gets
// 500, so the app doesn't treat an outage as being signed out.
var ErrUnauthenticated = errors.New("router: no valid credential")

// UnauthenticatedError is an ErrUnauthenticated that tells the client why, with
// a more specific code than "unauthenticated" (e.g. a Kiosk session asleep).
type UnauthenticatedError struct {
	Code   contract.ErrorCode
	Reason string
}

func (e *UnauthenticatedError) Error() string {
	return fmt.Sprintf("%v: %s (%s)", ErrUnauthenticated, e.Reason, e.Code)
}

// Is makes errors.Is(err, ErrUnauthenticated) true.
func (e *UnauthenticatedError) Is(target error) bool { return target == ErrUnauthenticated }

// Authenticator turns a request into a Caller, or an error: ErrUnauthenticated
// (wrapped or not) for a missing or invalid credential, anything else for a
// failure to resolve one.
type Authenticator interface {
	Authenticate(r *http.Request) (Caller, error)
}

// Route is one mounted route's declaration, kept so tests can walk them all.
type Route struct {
	Pattern    string   // as passed to http.ServeMux, method included: "GET /api/health"
	Accepts    []Accept // empty for an Open route
	OpenReason string   // non-empty only for an Open route
}

// Router wraps an http.ServeMux that only Open and Gated can register on.
type Router struct {
	mux    *http.ServeMux
	auth   Authenticator
	routes []Route
}

// New returns a Router. auth may be nil until a Gated route is mounted.
func New(auth Authenticator) *Router {
	return &Router{mux: http.NewServeMux(), auth: auth}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) { r.mux.ServeHTTP(w, req) }

// Routes lists every mounted route in registration order.
func (r *Router) Routes() []Route { return append([]Route(nil), r.routes...) }

// Open mounts a route anyone may call. reason says why it needs no caller.
func (r *Router) Open(pattern, reason string, h http.HandlerFunc) {
	mustHaveMethod(pattern)
	if reason == "" {
		panic(fmt.Sprintf("router: Open(%q): give a reason the route needs no caller", pattern))
	}
	r.routes = append(r.routes, Route{Pattern: pattern, OpenReason: reason})
	r.mux.Handle(pattern, h)
}

// Gated mounts a route only the accepted callers may call. Others get 401 (no
// valid credential), 403 (valid, but not accepted) or 500 (caller couldn't be
// resolved). The handler reads the
// caller with CallerFrom.
func (r *Router) Gated(pattern string, accepts []Accept, h http.HandlerFunc) {
	mustHaveMethod(pattern)
	if len(accepts) == 0 {
		panic(fmt.Sprintf("router: Gated(%q): declare at least one accepted caller", pattern))
	}
	if r.auth == nil {
		panic(fmt.Sprintf("router: Gated(%q): no Authenticator wired", pattern))
	}
	r.routes = append(r.routes, Route{Pattern: pattern, Accepts: append([]Accept(nil), accepts...)})
	r.mux.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) {
		caller, err := r.auth.Authenticate(req)
		switch {
		case errors.Is(err, ErrUnauthenticated):
			code := contract.ErrorCodeUnauthenticated
			if ue, ok := errors.AsType[*UnauthenticatedError](err); ok {
				code = ue.Code
			}
			WriteError(w, http.StatusUnauthorized, code)
			return
		case err != nil:
			WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
			return
		}
		for _, a := range accepts {
			if caller.Is(a) {
				h(w, req.WithContext(context.WithValue(req.Context(), callerKey{}, caller)))
				return
			}
		}
		WriteError(w, http.StatusForbidden, contract.ErrorCodeForbidden)
	})
}

type callerKey struct{}

// CallerFrom returns the Caller a Gated route admitted. Nil on an Open route.
func CallerFrom(ctx context.Context) Caller {
	c, _ := ctx.Value(callerKey{}).(Caller)
	return c
}

// WriteJSON writes v as a JSON body with status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes a contract.ErrorBody with status.
func WriteError(w http.ResponseWriter, status int, code contract.ErrorCode) {
	WriteJSON(w, status, contract.ErrorBody{Code: code})
}

func mustHaveMethod(pattern string) {
	method, _, ok := strings.Cut(pattern, " ")
	if !ok || method == "" || strings.HasPrefix(method, "/") {
		panic(fmt.Sprintf("router: %q: patterns must start with a method, e.g. \"GET /api/x\"", pattern))
	}
}
