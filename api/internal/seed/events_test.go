package seed_test

import (
	"context"
	"testing"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/seed"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

func TestLoadEventsAddsPastUpcomingAndCancelledEventsOnce(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	ctx := context.Background()
	today := "2026-11-09"

	if err := seed.LoadEvents(ctx, env.Firestore, today); err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}
	if err := seed.LoadEvents(ctx, env.Firestore, today); err != nil {
		t.Fatalf("second LoadEvents: %v", err)
	}

	snaps, err := env.Firestore.Collection("events").Documents(ctx).GetAll()
	if err != nil {
		t.Fatal(err)
	}
	var past, upcoming, cancelled int
	for _, s := range snaps {
		var e contract.Event
		if err := s.DataTo(&e); err != nil {
			t.Fatal(err)
		}
		if e.Name == "" || e.EndDate < e.StartDate {
			t.Errorf("events/%s = %+v, want a name and endDate >= startDate", s.Ref.ID, e)
		}
		switch {
		case e.Cancelled:
			cancelled++
		case e.EndDate < today:
			past++
		default:
			upcoming++
		}
	}
	if len(snaps) != 5 || past == 0 || upcoming == 0 || cancelled == 0 {
		t.Fatalf("seeded %d Events (%d past, %d upcoming, %d cancelled); want 5 once, with each kind", len(snaps), past, upcoming, cancelled)
	}
}
