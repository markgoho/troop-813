// Command seed loads fictional Scouts into the emulators. Run it through
// `bun run seed` from the repo root (with `bun run dev` running), which points
// it at this worktree's emulators. It refuses any non-emulator target.
//
// Env: FIRESTORE_EMULATOR_HOST, and GCLOUD_PROJECT (or GOOGLE_CLOUD_PROJECT)
// starting with "demo-".
package main

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"

	"github.com/markgoho/troop-813/api/internal/seed"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	project, err := seed.Target(os.Getenv)
	if err != nil {
		return err
	}
	ctx := context.Background()
	fs, err := firestore.NewClient(ctx, project, option.WithoutAuthentication())
	if err != nil {
		return err
	}
	defer func() { _ = fs.Close() }()
	added, err := seed.Load(ctx, fs)
	if err != nil {
		return err
	}
	fmt.Printf("seed: added %d Scouts to %s\n", added, project)
	return nil
}
