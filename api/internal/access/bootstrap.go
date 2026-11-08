// Package access manages who may use HQ: users/{uid} roles (ADR 0003).
package access

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	fbauth "firebase.google.com/go/v4/auth"

	"github.com/markgoho/troop-813/api/internal/auth"
	"github.com/markgoho/troop-813/api/internal/contract"
)

// BootstrapAdmin sets the role of the user with email to admin. It's the only
// way to make an Admin; cmd/bootstrap-admin runs it. The user must have
// signed in to HQ once, so users/{uid} exists. Returns the user's uid.
func BootstrapAdmin(ctx context.Context, fs *firestore.Client, authClient *fbauth.Client, email string) (string, error) {
	rec, err := authClient.GetUserByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("access: no Auth user with that email (sign in to HQ first): %w", err)
	}
	ref := fs.Collection(auth.Users).Doc(rec.UID)
	// Update fails if the doc is missing: the user hasn't signed in to HQ.
	if _, err := ref.Update(ctx, []firestore.Update{{Path: "role", Value: contract.RoleAdmin}}); err != nil {
		return "", fmt.Errorf("access: set users/%s.role (has the user signed in to HQ?): %w", rec.UID, err)
	}
	return rec.UID, nil
}
