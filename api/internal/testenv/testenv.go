// Package testenv runs the real API router against the Firestore and Auth
// emulators, for HTTP tests at the API seam.
//
// Run tests through `bun run test` (or `bun run test:go`), which boots the
// emulators on this worktree's ports and exports FIRESTORE_EMULATOR_HOST,
// FIREBASE_AUTH_EMULATOR_HOST and GCLOUD_PROJECT. Without them Setup fails
// the test rather than skipping it.
//
// Each Setup gets its own emulator project ID, so tests (and `go test`
// packages) run in parallel without seeing each other's docs or users.
package testenv

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/server"
)

// DefaultNow is where every Env's clock starts: Monday 2026-11-09, 19:00 in
// America/New_York, the start of a Meeting.
var DefaultNow = time.Date(2026, 11, 9, 19, 0, 0, 0, clock.NY)

// Env is one test's API server plus direct emulator clients for arranging
// state and asserting stored docs.
type Env struct {
	ProjectID string
	Server    *httptest.Server
	Firestore *firestore.Client
	Auth      *auth.Client
	Clock     *FakeClock

	authHost string
	logs     *syncBuffer
}

// Option adjusts the server's Deps before the router is built.
type Option func(*server.Deps)

// Setup starts an Env for t on a fresh emulator project. Everything is torn
// down when t ends.
func Setup(t testing.TB, opts ...Option) *Env {
	t.Helper()
	fsHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	authHost := os.Getenv("FIREBASE_AUTH_EMULATOR_HOST")
	if fsHost == "" || authHost == "" {
		t.Fatal("testenv: FIRESTORE_EMULATOR_HOST and FIREBASE_AUTH_EMULATOR_HOST are unset; run Go tests with `bun run test:go` (or `bun run test`) so the emulators are running")
	}
	base := os.Getenv("GCLOUD_PROJECT")
	if base == "" {
		base = "demo-troop-813"
	}
	projectID := base + "-" + randomHex(t, 4)
	ctx := context.Background()

	fs, err := firestore.NewClient(ctx, projectID, option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("testenv: firestore client: %v", err)
	}
	t.Cleanup(func() { _ = fs.Close() })

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID}, option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("testenv: firebase app: %v", err)
	}
	authClient, err := app.Auth(ctx)
	if err != nil {
		t.Fatalf("testenv: auth client: %v", err)
	}

	env := &Env{
		ProjectID: projectID,
		Firestore: fs,
		Auth:      authClient,
		Clock:     NewFakeClock(DefaultNow),
		authHost:  authHost,
		logs:      &syncBuffer{},
	}
	deps := server.Deps{
		Now:       env.Clock.Now,
		Firestore: fs,
		Auth:      authClient,
		Logger:    slog.New(slog.NewJSONHandler(env.logs, nil)),
	}
	for _, opt := range opts {
		opt(&deps)
	}
	env.Server = httptest.NewServer(server.New(deps))
	t.Cleanup(env.Server.Close)
	return env
}

// Logs is everything the server logged so far (JSON lines).
func (e *Env) Logs() string { return e.logs.String() }

// Request calls the server. token, if non-empty, is sent as a Bearer ID
// token. body, if non-nil, is sent as JSON. The caller closes the body.
func (e *Env) Request(t testing.TB, method, path, token string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("testenv: marshal body: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, e.Server.URL+path, r)
	if err != nil {
		t.Fatalf("testenv: new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := e.Server.Client().Do(req)
	if err != nil {
		t.Fatalf("testenv: %s %s: %v", method, path, err)
	}
	return resp
}

// IDToken signs uid in on the Auth emulator (creating the user if needed)
// with custom claims, and returns a Firebase ID token the server verifies
// as real. Use it for Leader, Admin and Parent callers (whose role lives in
// users/{uid}, which the test writes itself) and for Kiosk tokens
// (claims role, kioskSessionId, meetingId).
//
// The emulator maps API-key requests to its default project, so this signs
// in as the emulator's "owner" admin with targetProjectId to land the user
// in this Env's own project.
func (e *Env) IDToken(t testing.TB, uid string, claims map[string]any) string {
	t.Helper()
	ctx := context.Background()
	custom, err := e.Auth.CustomTokenWithClaims(ctx, uid, claims)
	if err != nil {
		t.Fatalf("testenv: custom token: %v", err)
	}
	reqBody, err := json.Marshal(map[string]any{"token": custom, "returnSecureToken": true, "targetProjectId": e.ProjectID})
	if err != nil {
		t.Fatalf("testenv: marshal: %v", err)
	}
	url := fmt.Sprintf("http://%s/identitytoolkit.googleapis.com/v1/accounts:signInWithCustomToken", e.authHost)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("testenv: new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer owner")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("testenv: signInWithCustomToken: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("testenv: signInWithCustomToken: %d %s", resp.StatusCode, raw)
	}
	var out struct {
		IDToken string `json:"idToken"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.IDToken == "" {
		t.Fatalf("testenv: signInWithCustomToken: no idToken in %s", raw)
	}
	return out.IDToken
}

// GoogleToken creates an Auth user with an email and display name, as a
// Google sign-in would, and returns its ID token. No users/{uid} doc is
// written: the first API call with the token is the user's first sign-in.
func (e *Env) GoogleToken(t testing.TB, uid, email, displayName string) string {
	t.Helper()
	user := (&auth.UserToCreate{}).UID(uid).Email(email).DisplayName(displayName)
	if _, err := e.Auth.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("testenv: create user %s: %v", uid, err)
	}
	return e.IDToken(t, uid, nil)
}

// RoleToken writes users/{uid} with role (as if uid had signed in before)
// and returns an ID token for uid. Use it for Leader, Admin and Parent callers.
func (e *Env) RoleToken(t testing.TB, uid string, role contract.Role) string {
	t.Helper()
	e.SetRole(t, uid, role)
	return e.IDToken(t, uid, nil)
}

// SetRole writes users/{uid}, replacing any existing doc, with role.
func (e *Env) SetRole(t testing.TB, uid string, role contract.Role) {
	t.Helper()
	doc := contract.User{Email: uid + "@example.com", DisplayName: uid, Role: role, CreatedAt: DefaultNow}
	if _, err := e.Firestore.Collection("users").Doc(uid).Set(context.Background(), doc); err != nil {
		t.Fatalf("testenv: set users/%s: %v", uid, err)
	}
}

// User reads users/{uid}; ok is false if it doesn't exist.
func (e *Env) User(t testing.TB, uid string) (contract.User, bool) {
	t.Helper()
	snap, err := e.Firestore.Collection("users").Doc(uid).Get(context.Background())
	if status.Code(err) == codes.NotFound {
		return contract.User{}, false
	}
	if err != nil {
		t.Fatalf("testenv: get users/%s: %v", uid, err)
	}
	var u contract.User
	if err := snap.DataTo(&u); err != nil {
		t.Fatalf("testenv: decode users/%s: %v", uid, err)
	}
	return u, true
}

// FakeClock is a settable clock, safe for concurrent use.
type FakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewFakeClock returns a FakeClock reading t.
func NewFakeClock(t time.Time) *FakeClock { return &FakeClock{now: t} }

// Now returns the clock's current instant.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Set moves the clock to t.
func (c *FakeClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

// Advance moves the clock forward by d.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func randomHex(t testing.TB, n int) string {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("testenv: rand: %v", err)
	}
	return hex.EncodeToString(b)
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
