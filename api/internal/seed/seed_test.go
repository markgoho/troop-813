package seed_test

import (
	"context"
	"strings"
	"testing"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/seed"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

func TestTargetRefusesAnythingButADemoEmulatorProject(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		env  map[string]string
		ok   bool
	}{
		{"emulator demo project", map[string]string{"FIRESTORE_EMULATOR_HOST": "127.0.0.1:21010", "GCLOUD_PROJECT": "demo-troop-813"}, true},
		{"no emulator host", map[string]string{"GCLOUD_PROJECT": "demo-troop-813"}, false},
		{"real project", map[string]string{"FIRESTORE_EMULATOR_HOST": "127.0.0.1:21010", "GCLOUD_PROJECT": "troop-813"}, false},
		{"real project via GOOGLE_CLOUD_PROJECT", map[string]string{"FIRESTORE_EMULATOR_HOST": "127.0.0.1:21010", "GOOGLE_CLOUD_PROJECT": "troop-813", "GCLOUD_PROJECT": "demo-troop-813"}, false},
		{"no project", map[string]string{"FIRESTORE_EMULATOR_HOST": "127.0.0.1:21010"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			project, err := seed.Target(func(k string) string { return c.env[k] })
			if c.ok && (err != nil || project != "demo-troop-813") {
				t.Fatalf("Target = %q, %v; want demo-troop-813, nil", project, err)
			}
			if !c.ok && err == nil {
				t.Fatalf("Target = %q, nil; want a refusal", project)
			}
		})
	}
}

func TestLoadAddsFictionalScoutsOnce(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	ctx := context.Background()

	added, err := seed.Load(ctx, env.Firestore)
	if err != nil || added == 0 {
		t.Fatalf("Load = %d, %v; want some Scouts added", added, err)
	}
	again, err := seed.Load(ctx, env.Firestore)
	if err != nil || again != 0 {
		t.Fatalf("second Load = %d, %v; want 0 (already seeded)", again, err)
	}

	snaps, err := env.Firestore.Collection("scouts").Documents(ctx).GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != added {
		t.Fatalf("%d scouts stored, want %d", len(snaps), added)
	}
	var active, inactive int
	firsts := map[string]int{}
	for _, s := range snaps {
		var sc contract.Scout
		if err := s.DataTo(&sc); err != nil {
			t.Fatal(err)
		}
		if sc.Active {
			active++
			firsts[strings.ToLower(sc.FirstName)]++
		} else {
			inactive++
		}
	}
	if active == 0 || inactive == 0 {
		t.Fatalf("seeded %d active, %d inactive; want both", active, inactive)
	}
	shared := false
	for _, n := range firsts {
		shared = shared || n > 1
	}
	if !shared {
		t.Fatal("no two active seeded Scouts share a first name (the Kiosk's last-initial rule needs one)")
	}
}
