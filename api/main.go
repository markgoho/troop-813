// Command api is the Troop 813 HQ API: the only writer to Firestore.
//
// Env: PORT (Cloud Run sets it; `bun run dev` passes this worktree's API port),
// GOOGLE_CLOUD_PROJECT or GCLOUD_PROJECT. FIRESTORE_EMULATOR_HOST and
// FIREBASE_AUTH_EMULATOR_HOST point the SDKs at the emulators in local dev.
package main

//go:generate go tool tygo generate

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"

	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("api exited", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	projectID := cmp.Or(os.Getenv("GOOGLE_CLOUD_PROJECT"), os.Getenv("GCLOUD_PROJECT"))
	if projectID == "" {
		return errors.New("set GOOGLE_CLOUD_PROJECT or GCLOUD_PROJECT")
	}
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

	srv := &http.Server{
		Addr:              ":" + cmp.Or(os.Getenv("PORT"), "8080"),
		Handler:           server.New(server.Deps{Now: clock.Real, Firestore: fs, Auth: authClient, Logger: logger}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	logger.Info("api listening", "addr", srv.Addr, "project", projectID)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
