package meeting

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
)

// Live is tonight's Meeting as the roster sees it: today's meetings doc, read
// inside the caller's transaction so a Roster change and the handoffOrder
// change commit together. A nil *Live means there is no live Meeting and
// every method is a no-op, so callers need no branch.
type Live struct {
	ref *firestore.DocumentRef
	m   contract.Meeting
}

// ReadLive reads today's Meeting (America/New_York date) in tx. It returns
// nil when there is none or it is past 22:00. Like all reads in a Firestore
// transaction it must come before the transaction's first write.
func ReadLive(tx *firestore.Transaction, fs *firestore.Client, now time.Time) (*Live, error) {
	id := clock.LocalDate(now)
	ends, err := clock.MeetingEnds(id)
	if err != nil {
		return nil, err
	}
	if !now.Before(ends) {
		return nil, nil //nolint:nilnil // nil Live is the "no live Meeting" value
	}
	ref := fs.Collection(Meetings).Doc(id)
	snap, err := tx.Get(ref)
	if status.Code(err) == codes.NotFound {
		return nil, nil //nolint:nilnil // see above
	}
	if err != nil {
		return nil, fmt.Errorf("meeting: read meetings/%s: %w", id, err)
	}
	var m contract.Meeting
	if err := snap.DataTo(&m); err != nil {
		return nil, fmt.Errorf("meeting: decode meetings/%s: %w", id, err)
	}
	return &Live{ref: ref, m: m}, nil
}

// AddScout inserts scoutID at a random gap among the Scouts not yet checked
// in, so the Kiosk's Handoff reaches them tonight. Already in the order:
// nothing happens.
func (l *Live) AddScout(tx *firestore.Transaction, scoutID string) error {
	if l == nil || slices.Contains(l.m.HandoffOrder, scoutID) {
		return nil
	}
	order := insertAt(l.m.HandoffOrder, l.m.CheckedIn, scoutID, func(n int) int {
		return rand.IntN(n) //nolint:gosec // not security-sensitive
	})
	return tx.Update(l.ref, []firestore.Update{{Path: "handoffOrder", Value: order}})
}

// RemoveScout takes a deleted Scout out of tonight's handoffOrder and notFound.
func (l *Live) RemoveScout(tx *firestore.Transaction, scoutID string) error {
	if l == nil {
		return nil
	}
	var updates []firestore.Update
	if slices.Contains(l.m.HandoffOrder, scoutID) {
		updates = append(updates, firestore.Update{Path: "handoffOrder", Value: slices.DeleteFunc(slices.Clone(l.m.HandoffOrder), func(s string) bool { return s == scoutID })})
	}
	if slices.Contains(l.m.NotFound, scoutID) {
		updates = append(updates, firestore.Update{Path: "notFound", Value: slices.DeleteFunc(slices.Clone(l.m.NotFound), func(s string) bool { return s == scoutID })})
	}
	if len(updates) == 0 {
		return nil
	}
	return tx.Update(l.ref, updates)
}

// insertAt returns order with id placed in one of the gaps between Scouts not
// yet checked in: just before the pick'th such Scout, or after the last one
// (n = unchecked + 1 gaps; pick(n) returns 0..n-1). With nobody unchecked it
// appends. order is not modified; an id already present is left alone.
func insertAt(order []string, checkedIn map[string]string, id string, pick func(n int) int) []string {
	if slices.Contains(order, id) {
		return order
	}
	var unchecked []int // indexes into order
	for i, s := range order {
		if _, in := checkedIn[s]; !in {
			unchecked = append(unchecked, i)
		}
	}
	at := len(order)
	if len(unchecked) > 0 {
		if g := pick(len(unchecked) + 1); g < len(unchecked) {
			at = unchecked[g]
		} else {
			at = unchecked[len(unchecked)-1] + 1
		}
	}
	return slices.Insert(slices.Clone(order), at, id)
}
