package testenv_test

import (
	"testing"

	"firebase.google.com/go/v4/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/markgoho/troop-813/api/internal/testenv"
)

func TestIDTokenVerifiesWithUIDAndClaims(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)

	token := env.IDToken(t, "kiosk-test", map[string]any{
		"role":           "kiosk",
		"kioskSessionId": "session-1",
		"meetingId":      "2026-11-09",
	})

	verified, err := env.Auth.VerifyIDToken(t.Context(), token)
	if err != nil {
		t.Fatalf("VerifyIDToken: %v", err)
	}
	if verified.UID != "kiosk-test" {
		t.Errorf("UID = %q, want kiosk-test", verified.UID)
	}
	for k, want := range map[string]string{"role": "kiosk", "kioskSessionId": "session-1", "meetingId": "2026-11-09"} {
		if got := verified.Claims[k]; got != want {
			t.Errorf("claim %s = %v, want %s", k, got, want)
		}
	}
}

func TestEachTestGetsItsOwnFirestoreAndAuth(t *testing.T) {
	t.Parallel()
	a := testenv.Setup(t)
	b := testenv.Setup(t)

	if _, err := a.Firestore.Doc("scouts/s1").Set(t.Context(), map[string]any{"firstName": "Avery"}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, err := a.Firestore.Doc("scouts/s1").Get(t.Context()); err != nil {
		t.Fatalf("own doc: %v", err)
	}
	if _, err := b.Firestore.Doc("scouts/s1").Get(t.Context()); status.Code(err) != codes.NotFound {
		t.Fatalf("a doc written in one Env is visible in another (err = %v)", err)
	}

	a.IDToken(t, "same-uid", nil)
	if _, err := a.Auth.GetUser(t.Context(), "same-uid"); err != nil {
		t.Fatalf("own user: %v", err)
	}
	if _, err := b.Auth.GetUser(t.Context(), "same-uid"); !auth.IsUserNotFound(err) {
		t.Fatal("a user created in one Env is visible in another")
	}
}
