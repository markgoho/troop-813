// Command bootstrap-admin makes one signed-in user the Admin (ADR 0003).
// Run it through `bun run bootstrap-admin <email> [--prod]` from the repo
// root, which points it at this worktree's emulators by default, or at the
// real troop-813 project with --prod (needs gcloud application-default
// credentials with Firestore and Firebase Auth access).
//
// Env: GOOGLE_CLOUD_PROJECT or GCLOUD_PROJECT; FIRESTORE_EMULATOR_HOST and
// FIREBASE_AUTH_EMULATOR_HOST to target the emulators.
package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"

	"github.com/markgoho/troop-813/api/internal/access"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "bootstrap-admin:", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 || os.Args[1] == "" {
		return errors.New("usage: bootstrap-admin <email>")
	}
	email := os.Args[1]
	projectID := cmp.Or(os.Getenv("GOOGLE_CLOUD_PROJECT"), os.Getenv("GCLOUD_PROJECT"))
	if projectID == "" {
		return errors.New("set GOOGLE_CLOUD_PROJECT or GCLOUD_PROJECT")
	}
	ctx := context.Background()
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return err
	}
	defer func() { _ = fs.Close() }()
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		return err
	}
	authClient, err := app.Auth(ctx)
	if err != nil {
		return err
	}
	uid, err := access.BootstrapAdmin(ctx, fs, authClient, email)
	if err != nil {
		return err
	}
	fmt.Printf("users/%s is now admin (project %s)\n", uid, projectID)
	return nil
}
