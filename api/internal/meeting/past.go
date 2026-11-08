package meeting

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
)

var (
	// ErrNotPast means the date isn't a real YYYY-MM-DD before today.
	ErrNotPast = errors.New("meeting: date is not a past day")
	// ErrMeetingExists means a Meeting doc already exists for the date.
	ErrMeetingExists = errors.New("meeting: Meeting exists")
	// ErrHasAttendance means somebody is still present, so the Meeting stays.
	ErrHasAttendance = errors.New("meeting: Scouts are present")
)

// CreatePast backfills the Meeting for date (a Leader adds it after the
// fact, e.g. when the Admin was absent). The date must be a real day before
// today in America/New_York and have no Meeting; the doc is created empty so
// the Meeting counts (a Meeting counts only if its doc exists, #30). Holidays
// are just dates nobody creates.
func CreatePast(ctx context.Context, fs *firestore.Client, leaderUID, date string) error {
	day, err := time.ParseInLocation(time.DateOnly, date, clock.NY)
	if err != nil || day.Format(time.DateOnly) != date {
		return ErrNotPast
	}
	now := clock.Now(ctx)
	if date >= clock.LocalDate(now) {
		return ErrNotPast
	}
	_, err = fs.Collection(Meetings).Doc(date).Create(ctx, contract.Meeting{
		StartedAt:    now,
		StartedBy:    leaderUID,
		HandoffOrder: []string{},
		NotFound:     []string{},
		CheckedIn:    map[string]string{},
	})
	if status.Code(err) == codes.AlreadyExists {
		return ErrMeetingExists
	}
	if err != nil {
		return fmt.Errorf("meeting: create meetings/%s: %w", date, err)
	}
	return nil
}

// Delete removes a Meeting nobody is present at, with its undone Attendance
// records (so they can't pin a Scout as in-use forever). ErrNoMeeting when
// absent, ErrHasAttendance while any Scout is checked in.
func Delete(ctx context.Context, fs *firestore.Client, meetingID string) error {
	if !validID(meetingID) {
		return ErrNoMeeting
	}
	ref := fs.Collection(Meetings).Doc(meetingID)
	return fs.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		m, err := readMeeting(tx, ref)
		if err != nil {
			return err
		}
		if len(m.CheckedIn) > 0 {
			return ErrHasAttendance
		}
		snaps, err := tx.Documents(ref.Collection(Attendance)).GetAll()
		if err != nil {
			return fmt.Errorf("meeting: list attendance of %s: %w", meetingID, err)
		}
		for _, s := range snaps {
			var a contract.Attendance
			if err := s.DataTo(&a); err != nil {
				return fmt.Errorf("meeting: decode attendance/%s: %w", s.Ref.ID, err)
			}
			if a.UndoneAt == nil {
				return ErrHasAttendance
			}
		}
		for _, s := range snaps {
			if err := tx.Delete(s.Ref); err != nil {
				return err
			}
		}
		return tx.Delete(ref)
	})
}
