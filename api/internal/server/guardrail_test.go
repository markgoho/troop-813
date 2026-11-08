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

// Every Gated route rejects anonymous callers with 401, and Parents, Leaders
// and the Kiosk with 403 unless it accepts them. So a Kiosk token is refused
// on every Leader route (spec story 42), and a Leader can't act as the Kiosk.
func TestEveryGatedRouteRejectsCallersItDoesNotAccept(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	parent := env.RoleToken(t, "u-parent", contract.RoleParent)
	leader := env.RoleToken(t, "u-leader", contract.RoleLeader)
	kiosk, _ := unlockKiosk(t, env) // a live session, so a refusal is 403, not 401

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
			for name, c := range map[string]struct {
				token  string
				accept router.Accept
			}{
				"parent": {parent, router.Parent},
				"leader": {leader, router.Leader},
				"kiosk":  {kiosk, router.Kiosk},
			} {
				if slices.Contains(rt.Accepts, c.accept) {
					continue
				}
				if got := status(t, env, method, path, c.token); got != http.StatusForbidden {
					t.Errorf("%s: status %d, want 403", name, got)
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
