package server_test

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
	"github.com/markgoho/troop-813/api/internal/server"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

type noAuth struct{}

func (noAuth) Authenticate(*http.Request) (router.Caller, error) {
	return nil, router.ErrUnauthenticated
}

var wildcard = regexp.MustCompile(`\{[^}]+\}`)

// Every Gated route rejects anonymous callers with 401, and Parents with 403
// unless it accepts Parents. Later tickets extend this for Kiosk tokens on
// Leader routes (spec story 42).
func TestEveryGatedRouteRejectsCallersItDoesNotAccept(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	parent := env.RoleToken(t, "u-parent", contract.RoleParent)

	routes := server.Routes(server.Deps{Authenticator: noAuth{}}).Routes()
	for _, rt := range routes {
		if len(rt.Accepts) == 0 {
			continue
		}
		method, path, _ := strings.Cut(rt.Pattern, " ")
		// Fill path wildcards ({id}, {date}) with placeholder values.
		path = wildcard.ReplaceAllString(path, "x")
		t.Run(rt.Pattern, func(t *testing.T) {
			t.Parallel()
			if got := status(t, env, method, path, ""); got != http.StatusUnauthorized {
				t.Errorf("anonymous: status %d, want 401", got)
			}
			if !slices.Contains(rt.Accepts, router.Parent) {
				if got := status(t, env, method, path, parent); got != http.StatusForbidden {
					t.Errorf("parent: status %d, want 403", got)
				}
			}
		})
	}
}

func status(t *testing.T, env *testenv.Env, method, path, token string) int {
	t.Helper()
	resp := env.Request(t, method, path, token, nil)
	_ = resp.Body.Close()
	return resp.StatusCode
}
