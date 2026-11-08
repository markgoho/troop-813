package router_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
)

// fakeCaller counts as each Accept in its set.
type fakeCaller map[router.Accept]bool

func (c fakeCaller) Is(a router.Accept) bool { return c[a] }

// fakeAuth maps the Authorization header to a caller or an error.
type fakeAuth map[string]struct {
	caller router.Caller
	err    error
}

func (f fakeAuth) Authenticate(r *http.Request) (router.Caller, error) {
	got, ok := f[r.Header.Get("Authorization")]
	if !ok {
		return nil, router.ErrUnauthenticated
	}
	return got.caller, got.err
}

func serve(t *testing.T) *httptest.Server {
	t.Helper()
	auth := fakeAuth{
		"leader": {caller: fakeCaller{router.Leader: true}},
		"admin":  {caller: fakeCaller{router.Leader: true, router.Admin: true}},
		"parent": {caller: fakeCaller{router.Parent: true}},
		"broken": {err: errors.New("firestore unavailable")},
	}
	r := router.New(auth)
	r.Gated("GET /leader", []router.Accept{router.Leader}, func(w http.ResponseWriter, req *http.Request) {
		c := router.CallerFrom(req.Context())
		_, _ = fmt.Fprintf(w, "admin=%v", c.Is(router.Admin))
	})
	r.Gated("GET /admin", []router.Accept{router.Admin}, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "ok")
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func get(t *testing.T, srv *httptest.Server, path, credential string) (int, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if credential != "" {
		req.Header.Set("Authorization", credential)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.Header.Get("Content-Type") == "application/json" {
		var body contract.ErrorBody
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, string(body.Code)
	}
	var b [64]byte
	n, _ := resp.Body.Read(b[:])
	return resp.StatusCode, string(b[:n])
}

func TestGated(t *testing.T) {
	t.Parallel()
	srv := serve(t)
	cases := []struct {
		name, path, credential string
		wantStatus             int
		wantBody               string
	}{
		{"no credential is unauthenticated", "/leader", "", 401, "unauthenticated"},
		{"bad credential is unauthenticated", "/leader", "nonsense", 401, "unauthenticated"},
		{"a caller the route does not accept is forbidden", "/leader", "parent", 403, "forbidden"},
		{"a Leader is not the Admin", "/admin", "leader", 403, "forbidden"},
		{"an accepted caller reaches the handler with itself in context", "/leader", "leader", 200, "admin=false"},
		{"the Admin passes a Leader route", "/leader", "admin", 200, "admin=true"},
		{"a failure to resolve the caller is a server error, not a sign-out", "/leader", "broken", 500, "internal"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			status, body := get(t, srv, c.path, c.credential)
			if status != c.wantStatus || body != c.wantBody {
				t.Fatalf("GET %s as %q = %d %q, want %d %q", c.path, c.credential, status, body, c.wantStatus, c.wantBody)
			}
		})
	}
}
