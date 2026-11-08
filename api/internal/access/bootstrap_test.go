package access_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/markgoho/troop-813/api/internal/access"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

func TestBootstrapAdminPromotesASignedInUser(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	token := env.GoogleToken(t, "u-taylor", "taylor.fictional@example.com", "Taylor Fictional")
	resp := env.Request(t, http.MethodGet, "/api/me", token, nil) // first sign-in
	_ = resp.Body.Close()

	uid, err := access.BootstrapAdmin(context.Background(), env.Firestore, env.Auth, "taylor.fictional@example.com")
	if err != nil || uid != "u-taylor" {
		t.Fatalf("BootstrapAdmin = %q, %v; want u-taylor, nil", uid, err)
	}

	resp = env.Request(t, http.MethodGet, "/api/me", token, nil)
	defer func() { _ = resp.Body.Close() }()
	var me contract.Me
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		t.Fatal(err)
	}
	if me.Role != contract.RoleAdmin {
		t.Fatalf("role after bootstrap = %q, want admin", me.Role)
	}
	if u, _ := env.User(t, "u-taylor"); u.Email != "taylor.fictional@example.com" {
		t.Fatalf("bootstrap clobbered the doc: %+v", u)
	}
}

func TestBootstrapAdminRefusesSomeoneWhoHasNotSignedIn(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	env.GoogleToken(t, "u-quinn", "quinn.fictional@example.com", "Quinn Fictional") // Auth user, no users doc

	for _, email := range []string{"quinn.fictional@example.com", "nobody@example.com"} {
		if _, err := access.BootstrapAdmin(context.Background(), env.Firestore, env.Auth, email); err == nil {
			t.Errorf("BootstrapAdmin(%q) succeeded, want an error", email)
		}
	}
	if _, ok := env.User(t, "u-quinn"); ok {
		t.Fatal("BootstrapAdmin created a users doc")
	}
}
