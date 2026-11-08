package seed

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/events"
)

// LoadEvents writes fictional Events dated around today (YYYY-MM-DD): two
// past, two upcoming (the first this coming weekend, so the Kiosk's
// last-chance card shows) and one cancelled. They have fixed seed-event-*
// IDs, so re-running resets them instead of adding more.
func LoadEvents(ctx context.Context, fs *firestore.Client, today string) error {
	t, err := time.Parse(time.DateOnly, today)
	if err != nil {
		return fmt.Errorf("seed: today %q: %w", today, err)
	}
	friday := t.AddDate(0, 0, (int(time.Friday)-int(t.Weekday())+7)%7)
	day := func(offset int) string { return friday.AddDate(0, 0, offset).Format(time.DateOnly) }
	seeds := map[string]contract.Event{
		"seed-event-service":   {Name: "Food Drive Service Day", StartDate: day(-48), EndDate: day(-48), Location: "Fictional Food Pantry"},
		"seed-event-last":      {Name: "Leaf Peeper Campout", StartDate: day(-28), EndDate: day(-26), Location: "Camp Fictional"},
		"seed-event-next":      {Name: "Fall Campout", StartDate: day(0), EndDate: day(2), Location: "Lake Fictional State Park"},
		"seed-event-later":     {Name: "Winter Cabin Campout", StartDate: day(28), EndDate: day(30)},
		"seed-event-cancelled": {Name: "Klondike Derby", StartDate: day(57), EndDate: day(57), Cancelled: true},
	}
	for id, e := range seeds {
		if _, err := fs.Collection(events.Collection).Doc(id).Set(ctx, e); err != nil {
			return fmt.Errorf("seed: events/%s: %w", id, err)
		}
	}
	return nil
}
