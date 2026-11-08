// Package rsvp owns rsvps/{eventId}_{scoutId} (ADR 0004): one doc per
// (Event, Scout) holding the current answer and the history of changes. It is
// the only code that writes those docs, and it keeps answer equal to the last
// history entry's answer.
//
// The interface is two-phase so several RSVP docs can change inside one
// Firestore transaction, which must do every read before any write:
//
//	d, err := rsvp.Get(tx, fs, eventID, scoutID) // reads only
//	d.Record(change)                             // in memory
//	err = d.Save(tx)                             // writes only
//
// SetByLeader wraps that for the Leader's Event page. The Kiosk sets answers
// the same way with an AttendanceID; Undo check-in loads that Scout's docs
// and drops the check-in's entries (both arrive with the Kiosk tickets).
package rsvp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
)

// Collection is the rsvps collection.
const Collection = "rsvps"

var (
	ErrInvalidAnswer = errors.New("rsvp: answer must be going, notGoing or null")
	ErrEventNotFound = errors.New("rsvp: no such Event")
	ErrScoutNotFound = errors.New("rsvp: no such Scout")
)

var validAnswers = []contract.RSVPAnswer{contract.RSVPAnswerGoing, contract.RSVPAnswerNotGoing}

// DocID is the ID of the RSVP doc for an Event and a Scout.
func DocID(eventID, scoutID string) string { return eventID + "_" + scoutID }

// Change is one answer to record. A nil Answer clears the RSVP so the Kiosk
// asks again. AttendanceID is the Kiosk check-in it was given in; nil for a Leader.
type Change struct {
	Answer       *contract.RSVPAnswer
	At           time.Time
	By           string // a Leader's uid or a kioskSessionId
	AttendanceID *string
}

// Doc is one RSVP doc loaded in a transaction. A Doc that doesn't exist yet
// is created by Save once something is recorded.
type Doc struct {
	ref   *firestore.DocumentRef
	data  contract.RSVP
	dirty bool
}

// Get reads the RSVP doc for (eventID, scoutID) in tx.
func Get(tx *firestore.Transaction, fs *firestore.Client, eventID, scoutID string) (*Doc, error) {
	ref := fs.Collection(Collection).Doc(DocID(eventID, scoutID))
	d := &Doc{ref: ref, data: contract.RSVP{EventID: eventID, ScoutID: scoutID, History: []contract.RSVPEntry{}}}
	snap, err := tx.Get(ref)
	if status.Code(err) == codes.NotFound {
		return d, nil
	}
	if err != nil {
		return nil, fmt.Errorf("rsvp: read %s: %w", ref.ID, err)
	}
	if err := snap.DataTo(&d.data); err != nil {
		return nil, fmt.Errorf("rsvp: decode %s: %w", ref.ID, err)
	}
	return d, nil
}

// Answer is the current answer, nil when there is none.
func (d *Doc) Answer() *contract.RSVPAnswer { return d.data.Answer }

// Record appends c to the history, unless it would leave the answer
// unchanged (a repeat answer, or clearing an RSVP that has no answer), and
// reports whether it did. It fails on an answer other than going or notGoing.
func (d *Doc) Record(c Change) (bool, error) {
	if c.Answer != nil && !slices.Contains(validAnswers, *c.Answer) {
		return false, ErrInvalidAnswer
	}
	if sameAnswer(d.data.Answer, c.Answer) {
		return false, nil
	}
	d.setHistory(append(slices.Clone(d.data.History), contract.RSVPEntry{
		Answer:       c.Answer,
		At:           c.At,
		By:           c.By,
		AttendanceID: c.AttendanceID,
	}))
	return true, nil
}

// setHistory replaces the history and derives answer from it: the last
// entry's answer, or null when no entry is left. Every change goes through here.
func (d *Doc) setHistory(h []contract.RSVPEntry) {
	d.data.History = h
	d.data.Answer = nil
	if len(h) > 0 {
		d.data.Answer = h[len(h)-1].Answer
	}
	d.dirty = true
}

// Save writes the doc in tx if anything was recorded.
func (d *Doc) Save(tx *firestore.Transaction) error {
	if !d.dirty {
		return nil
	}
	if err := tx.Set(d.ref, d.data); err != nil {
		return fmt.Errorf("rsvp: write %s: %w", d.ref.ID, err)
	}
	return nil
}

// SetByLeader records a Leader's answer (nil clears it) for the Scout at
// scout on the Event at event, both of which must exist. It runs its own
// transaction, so it serializes with deleting the Event.
func SetByLeader(ctx context.Context, fs *firestore.Client, event, scout *firestore.DocumentRef, answer *contract.RSVPAnswer, uid string) error {
	now := clock.Now(ctx)
	return fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if err := mustExist(tx, event, ErrEventNotFound); err != nil {
			return err
		}
		if err := mustExist(tx, scout, ErrScoutNotFound); err != nil {
			return err
		}
		d, err := Get(tx, fs, event.ID, scout.ID)
		if err != nil {
			return err
		}
		if _, err := d.Record(Change{Answer: answer, At: now, By: uid}); err != nil {
			return err
		}
		return d.Save(tx)
	})
}

func mustExist(tx *firestore.Transaction, ref *firestore.DocumentRef, notFound error) error {
	_, err := tx.Get(ref)
	if status.Code(err) == codes.NotFound {
		return notFound
	}
	if err != nil {
		return fmt.Errorf("rsvp: read %s: %w", ref.Path, err)
	}
	return nil
}

func sameAnswer(a, b *contract.RSVPAnswer) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
