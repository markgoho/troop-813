// Package seed loads fictional dev data into the emulators (`bun run seed`).
// The repo is public and the data is youth data, so every name here is made
// up, and Target refuses anything but an emulator demo- project.
package seed

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/roster"
)

// Target returns the project to seed, or an error unless Firestore points at
// an emulator (FIRESTORE_EMULATOR_HOST) and the project ID starts with
// "demo-" (which the emulators only accept offline).
func Target(getenv func(string) string) (string, error) {
	if getenv("FIRESTORE_EMULATOR_HOST") == "" {
		return "", errors.New("seed: FIRESTORE_EMULATOR_HOST is unset; seed only targets the emulators")
	}
	project := cmp.Or(getenv("GOOGLE_CLOUD_PROJECT"), getenv("GCLOUD_PROJECT"))
	if !strings.HasPrefix(project, "demo-") {
		return "", fmt.Errorf("seed: project %q is not a demo- project; seed only targets the emulators", project)
	}
	return project, nil
}

type scout struct {
	first, last string
	active      bool
}

// scouts are fictional. Two active ones share a first name, for the Kiosk's
// last-initial rule; one is inactive, for the Roster's Inactive section.
var scouts = []scout{
	{"Avery", "Quill", true},
	{"Avery", "Marsh", true},
	{"Rowan", "Pike", true},
	{"Juniper", "Holloway", true},
	{"Ezra", "Fenwick", true},
	{"Marlowe", "Birch", true},
	{"Tobias", "Wren", true},
	{"Sage", "Calloway", true},
	{"Finch", "Ashby", true},
	{"Linden", "Crowe", true},
	{"Quincy", "Harrow", false},
}

// Events are in events.go (LoadEvents).

// Load adds each fictional Scout not already on the Roster (by exact name,
// active or not) and returns how many it added. Re-running is a no-op.
func Load(ctx context.Context, fs *firestore.Client) (int, error) {
	snaps, err := fs.Collection(roster.Scouts).Documents(ctx).GetAll()
	if err != nil {
		return 0, fmt.Errorf("seed: read scouts: %w", err)
	}
	have := map[string]bool{}
	for _, s := range snaps {
		var sc contract.Scout
		if err := s.DataTo(&sc); err != nil {
			return 0, fmt.Errorf("seed: decode scouts/%s: %w", s.Ref.ID, err)
		}
		have[sc.FirstName+" "+sc.LastName] = true
	}
	added := 0
	for _, s := range scouts {
		if have[s.first+" "+s.last] {
			continue
		}
		id, err := roster.Add(ctx, fs, s.first, s.last)
		if err != nil {
			return added, err
		}
		if !s.active {
			// Deactivate isn't in roster yet; write the field directly.
			if _, err := fs.Collection(roster.Scouts).Doc(id).Update(ctx, []firestore.Update{{Path: "active", Value: false}}); err != nil {
				return added, fmt.Errorf("seed: deactivate scouts/%s: %w", id, err)
			}
		}
		added++
	}
	return added, nil
}
