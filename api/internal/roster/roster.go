// Package roster manages the Roster: the scouts collection. Handlers and the
// seed command go through it so every write applies the same rules.
//
// Every write is one transaction over the whole collection (a troop Roster is
// tens of docs, and Firestore can't query case-insensitively without an extra
// field the schema doesn't have), so the duplicate check, the write and the
// live-Meeting handoffOrder change commit together.
package roster

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/meeting"
	"github.com/markgoho/troop-813/api/internal/rsvp"
)

// Scouts is the scouts collection.
const Scouts = "scouts"

var (
	// ErrEmptyName means a first or last name was blank after trimming.
	ErrEmptyName = errors.New("roster: first and last name are required")
	// ErrNotFound means the Scout ID names no Scout.
	ErrNotFound = errors.New("roster: no such Scout")
	// ErrInUse means a delete was refused: the Scout has Attendance or RSVPs.
	ErrInUse = errors.New("roster: Scout has Attendance or RSVPs")
)

// DuplicateError means another Scout has the same trimmed, case-insensitive
// first + last name. MatchID is that Scout. An active match wins over an
// inactive one; when Active is false the UI offers Reactivate for MatchID.
type DuplicateError struct {
	MatchID string
	Active  bool
}

func (e *DuplicateError) Error() string {
	return fmt.Sprintf("roster: duplicate of scouts/%s", e.MatchID)
}

type entry struct {
	id string
	contract.Scout
}

// Add puts a new active Scout on the Roster and returns its ID. Names are
// trimmed. createdAt is clock.Now(ctx). If a Meeting is live, the Scout also
// joins tonight's handoffOrder.
func Add(ctx context.Context, fs *firestore.Client, firstName, lastName string) (string, error) {
	first, last, err := names(firstName, lastName)
	if err != nil {
		return "", err
	}
	now := clock.Now(ctx)
	ref := fs.Collection(Scouts).NewDoc()
	err = fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		all, err := readAll(tx, fs)
		if err != nil {
			return err
		}
		if dup := duplicate(all, first, last, "", false); dup != nil {
			return dup
		}
		live, err := meeting.ReadLive(tx, fs, now)
		if err != nil {
			return err
		}
		if err := tx.Create(ref, contract.Scout{FirstName: first, LastName: last, Active: true, CreatedAt: now}); err != nil {
			return err
		}
		return live.AddScout(tx, ref.ID)
	})
	if err != nil {
		return "", wrap("add", err)
	}
	return ref.ID, nil
}

// Rename changes a Scout's name. Another Scout with the same name, active or
// not, is a *DuplicateError. ErrNotFound if there is no such Scout.
func Rename(ctx context.Context, fs *firestore.Client, id, firstName, lastName string) error {
	first, last, err := names(firstName, lastName)
	if err != nil {
		return err
	}
	return wrap("rename", fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		all, err := readAll(tx, fs)
		if err != nil {
			return err
		}
		if find(all, id) == nil {
			return ErrNotFound
		}
		if dup := duplicate(all, first, last, id, false); dup != nil {
			return dup
		}
		return tx.Update(fs.Collection(Scouts).Doc(id), []firestore.Update{
			{Path: "firstName", Value: first},
			{Path: "lastName", Value: last},
		})
	}))
}

// Deactivate takes a Scout off the Kiosk, keeping their history. Deactivating
// an inactive Scout does nothing. ErrNotFound if there is no such Scout.
func Deactivate(ctx context.Context, fs *firestore.Client, id string) error {
	return setActive(ctx, fs, id, false)
}

// Reactivate puts an inactive Scout back on the Kiosk and, if a Meeting is
// live, into tonight's handoffOrder. An active Scout already has the name:
// *DuplicateError. ErrNotFound if there is no such Scout.
func Reactivate(ctx context.Context, fs *firestore.Client, id string) error {
	return setActive(ctx, fs, id, true)
}

func setActive(ctx context.Context, fs *firestore.Client, id string, active bool) error {
	now := clock.Now(ctx)
	return wrap("set active", fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		all, err := readAll(tx, fs)
		if err != nil {
			return err
		}
		self := find(all, id)
		if self == nil {
			return ErrNotFound
		}
		if self.Active == active {
			return nil
		}
		var live *meeting.Live
		if active {
			if dup := duplicate(all, self.FirstName, self.LastName, id, true); dup != nil {
				return dup
			}
			if live, err = meeting.ReadLive(tx, fs, now); err != nil {
				return err
			}
		}
		if err := tx.Update(fs.Collection(Scouts).Doc(id), []firestore.Update{{Path: "active", Value: active}}); err != nil {
			return err
		}
		return live.AddScout(tx, id)
	}))
}

// Delete removes a Scout who has no Attendance (undone or not) and no RSVP
// doc, even a cleared one: their history would be lost (ADR 0004). Otherwise
// ErrInUse. A live Meeting's handoffOrder and notFound drop the Scout.
func Delete(ctx context.Context, fs *firestore.Client, id string) error {
	if id == "" || strings.Contains(id, "/") {
		return ErrNotFound
	}
	now := clock.Now(ctx)
	ref := fs.Collection(Scouts).Doc(id)
	return wrap("delete", fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, err := tx.Get(ref); status.Code(err) == codes.NotFound {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		for _, q := range []firestore.Query{
			fs.CollectionGroup(meeting.Attendance).Where("scoutId", "==", id).Limit(1),
			fs.Collection(rsvp.Collection).Where("scoutId", "==", id).Limit(1),
		} {
			used, err := tx.Documents(&q).GetAll()
			if err != nil {
				return err
			}
			if len(used) > 0 {
				return ErrInUse
			}
		}
		live, err := meeting.ReadLive(tx, fs, now)
		if err != nil {
			return err
		}
		if err := tx.Delete(ref); err != nil {
			return err
		}
		return live.RemoveScout(tx, id)
	}))
}

func names(firstName, lastName string) (string, string, error) {
	first, last := strings.TrimSpace(firstName), strings.TrimSpace(lastName)
	if first == "" || last == "" {
		return "", "", ErrEmptyName
	}
	return first, last, nil
}

func readAll(tx *firestore.Transaction, fs *firestore.Client) ([]entry, error) {
	snaps, err := tx.Documents(fs.Collection(Scouts)).GetAll()
	if err != nil {
		return nil, fmt.Errorf("read scouts: %w", err)
	}
	out := make([]entry, 0, len(snaps))
	for _, snap := range snaps {
		e := entry{id: snap.Ref.ID}
		if err := snap.DataTo(&e.Scout); err != nil {
			return nil, fmt.Errorf("decode scouts/%s: %w", snap.Ref.ID, err)
		}
		out = append(out, e)
	}
	return out, nil
}

func find(all []entry, id string) *entry {
	for i := range all {
		if all[i].id == id {
			return &all[i]
		}
	}
	return nil
}

// duplicate returns the Scout other than skip who has the name (trimmed,
// case-insensitive), preferring an active one; nil if none. activeOnly
// ignores inactive Scouts.
func duplicate(all []entry, first, last, skip string, activeOnly bool) *DuplicateError {
	var inactive *DuplicateError
	for _, e := range all {
		if e.id == skip || (activeOnly && !e.Active) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(e.FirstName), strings.TrimSpace(first)) ||
			!strings.EqualFold(strings.TrimSpace(e.LastName), strings.TrimSpace(last)) {
			continue
		}
		if e.Active {
			return &DuplicateError{MatchID: e.id, Active: true}
		}
		if inactive == nil {
			inactive = &DuplicateError{MatchID: e.id}
		}
	}
	return inactive
}

// wrap adds context to infrastructure errors but leaves this package's typed
// errors (and a nil) unwrapped for errors.Is / errors.As.
func wrap(op string, err error) error {
	var dup *DuplicateError
	if err == nil || errors.Is(err, ErrEmptyName) || errors.Is(err, ErrNotFound) || errors.Is(err, ErrInUse) || errors.As(err, &dup) {
		return err
	}
	return fmt.Errorf("roster: %s: %w", op, err)
}
