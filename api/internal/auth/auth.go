// Package auth turns a request into a caller (ADR 0003). It is the only
// place that verifies ID tokens or reads roles; handlers say which callers
// they accept through router.Gated and read the caller with router.CallerFrom.
//
// A request resolves to:
//   - Leader{UID, IsAdmin}: users/{uid}.role is leader or admin.
//   - Parent{UID}: signed in with any other role, or signing in for the first
//     time (users/{uid} is then created with role parent).
//   - rejected (router.ErrUnauthenticated): no token, an invalid one, or a
//     Kiosk token (the Kiosk caller arrives with Start Meeting).
//
// The role is read on every call, so a role change applies to the next request.
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	fbauth "firebase.google.com/go/v4/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
)

// Leader is a Leader caller; the Admin is a Leader with IsAdmin.
type Leader struct {
	UID     string
	IsAdmin bool
}

// Is implements router.Caller.
func (l Leader) Is(a router.Accept) bool {
	return a == router.Leader || (a == router.Admin && l.IsAdmin)
}

// Parent is a signed-in caller who isn't a Leader.
type Parent struct {
	UID string
}

// Is implements router.Caller.
func (Parent) Is(a router.Accept) bool { return a == router.Parent }

// UID returns the uid of a Leader or Parent caller, or "" for any other.
func UID(c router.Caller) string {
	switch c := c.(type) {
	case Leader:
		return c.UID
	case Parent:
		return c.UID
	default:
		return ""
	}
}

// Users is the users collection.
const Users = "users"

// Authenticator implements router.Authenticator against Firebase Auth and Firestore.
type Authenticator struct {
	auth   *fbauth.Client
	fs     *firestore.Client
	logger *slog.Logger
}

// New returns an Authenticator. logger may be nil.
func New(authClient *fbauth.Client, fs *firestore.Client, logger *slog.Logger) *Authenticator {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Authenticator{auth: authClient, fs: fs, logger: logger}
}

// Authenticate implements router.Authenticator.
func (a *Authenticator) Authenticate(r *http.Request) (router.Caller, error) {
	ctx := r.Context()
	raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || raw == "" {
		return nil, router.ErrUnauthenticated
	}
	// Token freshness is checked against wall time, never the injected clock.
	tok, err := a.auth.VerifyIDToken(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", router.ErrUnauthenticated, err)
	}
	if role, _ := tok.Claims["role"].(string); role == "kiosk" {
		return nil, fmt.Errorf("%w: kiosk tokens are not accepted yet", router.ErrUnauthenticated)
	}
	role, err := a.role(ctx, tok)
	if err != nil {
		return nil, err
	}
	switch role {
	case contract.RoleAdmin:
		return Leader{UID: tok.UID, IsAdmin: true}, nil
	case contract.RoleLeader:
		return Leader{UID: tok.UID}, nil
	case contract.RoleParent:
		return Parent{UID: tok.UID}, nil
	default:
		// An unknown role gets the least access.
		return Parent{UID: tok.UID}, nil
	}
}

// role reads users/{uid}.role, creating the doc as a Parent on first sign-in.
func (a *Authenticator) role(ctx context.Context, tok *fbauth.Token) (contract.Role, error) {
	ref := a.fs.Collection(Users).Doc(tok.UID)
	snap, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return a.createParent(ctx, ref, tok)
	}
	if err != nil {
		return "", fmt.Errorf("auth: read users/%s: %w", tok.UID, err)
	}
	var u contract.User
	if err := snap.DataTo(&u); err != nil {
		return "", fmt.Errorf("auth: decode users/%s: %w", tok.UID, err)
	}
	return u.Role, nil
}

func (a *Authenticator) createParent(ctx context.Context, ref *firestore.DocumentRef, tok *fbauth.Token) (contract.Role, error) {
	email, _ := tok.Claims["email"].(string)
	name, _ := tok.Claims["name"].(string)
	if email == "" || name == "" {
		// Emulator and some providers leave these out of the token; the user record has them.
		rec, err := a.auth.GetUser(ctx, tok.UID)
		if err != nil {
			return "", fmt.Errorf("auth: get user %s: %w", tok.UID, err)
		}
		email, name = rec.Email, rec.DisplayName
	}
	doc := contract.User{Email: email, DisplayName: name, Role: contract.RoleParent, CreatedAt: clock.Now(ctx)}
	// Create, not Set: if a concurrent first request won, keep its doc and re-read the role.
	_, err := ref.Create(ctx, doc)
	if status.Code(err) == codes.AlreadyExists {
		return a.role(ctx, tok)
	}
	if err != nil {
		return "", fmt.Errorf("auth: create users/%s: %w", tok.UID, err)
	}
	a.logger.InfoContext(ctx, "first sign-in", "uid", tok.UID)
	return contract.RoleParent, nil
}

var _ router.Authenticator = (*Authenticator)(nil)
