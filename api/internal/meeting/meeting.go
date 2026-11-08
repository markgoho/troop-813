// Package meeting is all Meeting and Attendance logic (spec #17 Modules): the
// meetings/{date} doc, its attendance subcollection, and the Kiosk sessions a
// Start issues. Handlers, and later the roster (live Handoff insert), call
// these functions; nothing else writes those docs.
//
// Every write here is a Firestore transaction: concurrent claims on one Scout
// yield one Attendance record, and a Start never races another into two
// Meetings or two live sessions.
//
// Built so far: Start, CheckInAtKiosk. Later tickets add Leader check-in and
// undo (#23), mark not found (#24), Wake (#27), past Meetings (#29) and the
// live Handoff insert (#28), each as one more exported function over the same
// private helpers.
package meeting

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/markgoho/troop-813/api/internal/auth"
	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
)

const (
	// Meetings is the meetings collection; doc IDs are YYYY-MM-DD.
	Meetings = "meetings"
	// Attendance is the subcollection under each Meeting doc.
	Attendance = "attendance"
	// scouts is roster.Scouts, repeated so roster can import meeting (#28's
	// live Handoff insert) without a cycle.
	scouts = "scouts"
)

var (
	// ErrTooLate means it's 22:00 or later in America/New_York: no Start today.
	ErrTooLate = errors.New("meeting: past 22:00, too late to start")
	// ErrNoMeeting means the Meeting doc doesn't exist.
	ErrNoMeeting = errors.New("meeting: no such Meeting")
	// ErrScoutNotFound means the scoutId names no Scout.
	ErrScoutNotFound = errors.New("meeting: no such Scout")
	// ErrInvalidMethod means a check-in method the caller may not use.
	ErrInvalidMethod = errors.New("meeting: invalid check-in method")
)

// Started is the outcome of Start.
type Started struct {
	MeetingID string
	SessionID string
	Created   bool // false when today's Meeting already existed
}

// Start opens today's Meeting (America/New_York date) for the Kiosk: it
// creates meetings/{today} if absent, with every active Scout in a shuffled
// handoffOrder, and issues a new Kiosk session, marking every other live
// session replaced. Starting again on the same date keeps the Meeting as is
// (same order, same check-ins) and only issues a new session. Refused with
// ErrTooLate from 22:00. adminUID is the caller; the route allows only the Admin.
func Start(ctx context.Context, fs *firestore.Client, adminUID string) (Started, error) {
	now := clock.Now(ctx)
	id := clock.LocalDate(now)
	ends, err := clock.MeetingEnds(id)
	if err != nil {
		return Started{}, err
	}
	if !now.Before(ends) {
		return Started{}, ErrTooLate
	}

	var out Started
	err = fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		out = Started{MeetingID: id}
		meetingRef := fs.Collection(Meetings).Doc(id)
		_, err := tx.Get(meetingRef)
		switch {
		case status.Code(err) == codes.NotFound:
			out.Created = true
		case err != nil:
			return fmt.Errorf("meeting: read meetings/%s: %w", id, err)
		}
		var order []string
		if out.Created {
			if order, err = activeScoutIDs(tx, fs); err != nil {
				return err
			}
			// Shuffle order isn't a secret; it only has to differ each Meeting.
			rand.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] }) //nolint:gosec // not security-sensitive
		}
		live, err := tx.Documents(fs.Collection(auth.KioskSessions).Where("replaced", "==", false)).GetAll()
		if err != nil {
			return fmt.Errorf("meeting: read live kiosk sessions: %w", err)
		}

		// All reads done; writes follow.
		if out.Created {
			if err := tx.Create(meetingRef, contract.Meeting{
				StartedAt:    now,
				StartedBy:    adminUID,
				HandoffOrder: order,
				NotFound:     []string{},
				CheckedIn:    map[string]string{},
			}); err != nil {
				return err
			}
		}
		for _, s := range live {
			if err := tx.Update(s.Ref, []firestore.Update{{Path: "replaced", Value: true}}); err != nil {
				return err
			}
		}
		sessionRef := fs.Collection(auth.KioskSessions).NewDoc()
		out.SessionID = sessionRef.ID
		return tx.Create(sessionRef, contract.KioskSession{
			MeetingID:      id,
			CreatedAt:      now,
			CreatedBy:      adminUID,
			LastActivityAt: now,
		})
	})
	if err != nil {
		return Started{}, fmt.Errorf("meeting: start %s: %w", id, err)
	}
	return out, nil
}

func activeScoutIDs(tx *firestore.Transaction, fs *firestore.Client) ([]string, error) {
	snaps, err := tx.Documents(fs.Collection(scouts).Where("active", "==", true)).GetAll()
	if err != nil {
		return nil, fmt.Errorf("meeting: read active scouts: %w", err)
	}
	ids := make([]string, 0, len(snaps))
	for _, s := range snaps {
		ids = append(ids, s.Ref.ID)
	}
	return ids, nil
}

// CheckedIn is the outcome of a check-in.
type CheckedIn struct {
	AttendanceID string
	Repeat       bool // the Scout was already checked in; nothing was written
}

// CheckInAtKiosk checks scoutID in to the Kiosk session's Meeting by name list
// or Handoff, and bumps the session's lastActivityAt in the same transaction.
// A Scout already checked in gets their existing attendanceId back and
// nothing is written.
func CheckInAtKiosk(ctx context.Context, fs *firestore.Client, k auth.Kiosk, scoutID string, method contract.AttendanceMethod) (CheckedIn, error) {
	switch method {
	case contract.AttendanceMethodNameList, contract.AttendanceMethodHandoff:
	case contract.AttendanceMethodLeader:
		return CheckedIn{}, ErrInvalidMethod
	default:
		return CheckedIn{}, ErrInvalidMethod
	}
	return checkIn(ctx, fs, checkInArgs{
		meetingID: k.MeetingID,
		scoutID:   scoutID,
		method:    method,
		by:        k.SessionID,
		session:   k.SessionID,
	})
}

type checkInArgs struct {
	meetingID, scoutID string
	method             contract.AttendanceMethod
	by                 string // Attendance.by: Leader uid or kioskSessionId
	session            string // a Kiosk session to bump, or ""
}

// checkIn is the one place Attendance is created (Kiosk now; Leader check-in
// in #23 calls it with session "").
func checkIn(ctx context.Context, fs *firestore.Client, a checkInArgs) (CheckedIn, error) {
	if a.scoutID == "" || strings.Contains(a.scoutID, "/") {
		return CheckedIn{}, ErrScoutNotFound
	}
	now := clock.Now(ctx)
	meetingRef := fs.Collection(Meetings).Doc(a.meetingID)
	scoutRef := fs.Collection(scouts).Doc(a.scoutID)
	var out CheckedIn
	err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		out = CheckedIn{}
		m, err := readMeeting(tx, meetingRef)
		if err != nil {
			return err
		}
		if id, ok := m.CheckedIn[a.scoutID]; ok {
			out = CheckedIn{AttendanceID: id, Repeat: true}
			return nil
		}
		if _, err := tx.Get(scoutRef); status.Code(err) == codes.NotFound {
			return ErrScoutNotFound
		} else if err != nil {
			return fmt.Errorf("meeting: read scouts/%s: %w", a.scoutID, err)
		}

		attRef := meetingRef.Collection(Attendance).NewDoc()
		out.AttendanceID = attRef.ID
		if err := tx.Create(attRef, contract.Attendance{ScoutID: a.scoutID, Method: a.method, At: now, By: a.by}); err != nil {
			return err
		}
		if err := tx.Update(meetingRef, []firestore.Update{{FieldPath: firestore.FieldPath{"checkedIn", a.scoutID}, Value: attRef.ID}}); err != nil {
			return err
		}
		if a.session != "" {
			return tx.Update(fs.Collection(auth.KioskSessions).Doc(a.session), []firestore.Update{{Path: "lastActivityAt", Value: now}})
		}
		return nil
	})
	if err != nil {
		return CheckedIn{}, err
	}
	return out, nil
}

// MarkNotFound records that the Kiosk couldn't find scoutID this Meeting, and
// bumps the session's lastActivityAt, in one transaction. It never blocks a
// later check-in: the Scout can still tap their own name or be checked in by
// a Leader. Marking twice, or marking a Scout already checked in, writes
// nothing but the activity bump.
func MarkNotFound(ctx context.Context, fs *firestore.Client, k auth.Kiosk, scoutID string) error {
	if scoutID == "" || strings.Contains(scoutID, "/") {
		return ErrScoutNotFound
	}
	now := clock.Now(ctx)
	meetingRef := fs.Collection(Meetings).Doc(k.MeetingID)
	scoutRef := fs.Collection(scouts).Doc(scoutID)
	return fs.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		m, err := readMeeting(tx, meetingRef)
		if err != nil {
			return err
		}
		if _, err := tx.Get(scoutRef); status.Code(err) == codes.NotFound {
			return ErrScoutNotFound
		} else if err != nil {
			return fmt.Errorf("meeting: read scouts/%s: %w", scoutID, err)
		}
		if _, in := m.CheckedIn[scoutID]; !in {
			if err := tx.Update(meetingRef, []firestore.Update{{Path: "notFound", Value: firestore.ArrayUnion(scoutID)}}); err != nil {
				return err
			}
		}
		return tx.Update(fs.Collection(auth.KioskSessions).Doc(k.SessionID), []firestore.Update{{Path: "lastActivityAt", Value: now}})
	})
}

func readMeeting(tx *firestore.Transaction, ref *firestore.DocumentRef) (contract.Meeting, error) {
	snap, err := tx.Get(ref)
	if status.Code(err) == codes.NotFound {
		return contract.Meeting{}, ErrNoMeeting
	}
	if err != nil {
		return contract.Meeting{}, fmt.Errorf("meeting: read meetings/%s: %w", ref.ID, err)
	}
	var m contract.Meeting
	if err := snap.DataTo(&m); err != nil {
		return contract.Meeting{}, fmt.Errorf("meeting: decode meetings/%s: %w", ref.ID, err)
	}
	return m, nil
}
