// Package roster manages the Roster: the scouts collection. Handlers and the
// seed command go through it so every write applies the same rules.
package roster

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
)

// Scouts is the scouts collection.
const Scouts = "scouts"

// ErrEmptyName means a first or last name was blank after trimming.
var ErrEmptyName = errors.New("roster: first and last name are required")

// DuplicateError means a Scout with the same trimmed, case-insensitive first +
// last name is already on the Roster. MatchID is that Scout. Only an active
// match blocks an add for now; the Reactivate offer for an inactive match
// (spec story 56) builds on Active.
type DuplicateError struct {
	MatchID string
	Active  bool
}

func (e *DuplicateError) Error() string {
	return fmt.Sprintf("roster: duplicate of scouts/%s", e.MatchID)
}

// Add puts a new active Scout on the Roster and returns its ID. Names are
// trimmed. createdAt is clock.Now(ctx).
//
// The duplicate check reads the whole collection: a troop Roster is tens of
// docs, and Firestore can't query case-insensitively without an extra field
// the schema doesn't have.
func Add(ctx context.Context, fs *firestore.Client, firstName, lastName string) (string, error) {
	first, last := strings.TrimSpace(firstName), strings.TrimSpace(lastName)
	if first == "" || last == "" {
		return "", ErrEmptyName
	}
	if match, err := findActive(ctx, fs, first, last); err != nil {
		return "", err
	} else if match != "" {
		return "", &DuplicateError{MatchID: match, Active: true}
	}
	ref, _, err := fs.Collection(Scouts).Add(ctx, contract.Scout{
		FirstName: first,
		LastName:  last,
		Active:    true,
		CreatedAt: clock.Now(ctx),
	})
	if err != nil {
		return "", fmt.Errorf("roster: add: %w", err)
	}
	return ref.ID, nil
}

// findActive returns the ID of an active Scout named first + last (trimmed,
// case-insensitive), or "".
func findActive(ctx context.Context, fs *firestore.Client, first, last string) (string, error) {
	it := fs.Collection(Scouts).Where("active", "==", true).Documents(ctx)
	defer it.Stop()
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return "", nil
		}
		if err != nil {
			return "", fmt.Errorf("roster: read scouts: %w", err)
		}
		var s contract.Scout
		if err := snap.DataTo(&s); err != nil {
			return "", fmt.Errorf("roster: decode scouts/%s: %w", snap.Ref.ID, err)
		}
		if strings.EqualFold(strings.TrimSpace(s.FirstName), first) && strings.EqualFold(strings.TrimSpace(s.LastName), last) {
			return snap.Ref.ID, nil
		}
	}
}
