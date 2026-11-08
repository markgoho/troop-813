package meeting

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/rsvp"
)

// ErrNoAttendance means the Attendance doc doesn't exist.
var ErrNoAttendance = fmt.Errorf("meeting: no such Attendance")

// CheckInByLeader checks scoutID in to meetingID (a date) as a Leader:
// method leader, by the Leader's uid. A Scout already checked in gets their
// existing attendanceId back and nothing is written.
func CheckInByLeader(ctx context.Context, fs *firestore.Client, leaderUID, meetingID, scoutID string) (CheckedIn, error) {
	if !validID(meetingID) {
		return CheckedIn{}, ErrNoMeeting
	}
	return checkIn(ctx, fs, checkInArgs{
		meetingID: meetingID,
		scoutID:   scoutID,
		method:    contract.AttendanceMethodLeader,
		by:        leaderUID,
	})
}

// UndoCheckIn soft-deletes an Attendance (undoneAt/undoneBy) and, if it is
// still the Scout's current check-in, removes them from the Meeting's
// checkedIn, in one transaction. Undoing an already-undone record is a
// no-op. ErrNoMeeting / ErrNoAttendance when either doc is missing.
func UndoCheckIn(ctx context.Context, fs *firestore.Client, leaderUID, meetingID, attendanceID string) error {
	if !validID(meetingID) {
		return ErrNoMeeting
	}
	if !validID(attendanceID) {
		return ErrNoAttendance
	}
	now := clock.Now(ctx)
	meetingRef := fs.Collection(Meetings).Doc(meetingID)
	attRef := meetingRef.Collection(Attendance).Doc(attendanceID)
	return fs.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		m, err := readMeeting(tx, meetingRef)
		if err != nil {
			return err
		}
		snap, err := tx.Get(attRef)
		if status.Code(err) == codes.NotFound {
			return ErrNoAttendance
		} else if err != nil {
			return fmt.Errorf("meeting: read attendance/%s: %w", attendanceID, err)
		}
		var a contract.Attendance
		if err := snap.DataTo(&a); err != nil {
			return fmt.Errorf("meeting: decode attendance/%s: %w", attendanceID, err)
		}
		if a.UndoneAt != nil {
			return nil
		}
		// Every read comes before the first write.
		rsvps, err := rsvp.ForScout(tx, fs, a.ScoutID)
		if err != nil {
			return err
		}
		if err := tx.Update(attRef, []firestore.Update{
			{Path: "undoneAt", Value: now},
			{Path: "undoneBy", Value: leaderUID},
		}); err != nil {
			return err
		}
		for _, d := range rsvps {
			if d.RemoveAttendance(attendanceID) {
				if err := d.Save(tx); err != nil {
					return err
				}
			}
		}
		if m.CheckedIn[a.ScoutID] != attendanceID {
			return nil
		}
		return tx.Update(meetingRef, []firestore.Update{
			{FieldPath: firestore.FieldPath{"checkedIn", a.ScoutID}, Value: firestore.Delete},
		})
	})
}

// PreviewUndo lists the RSVP answers UndoCheckIn would roll back for this
// Attendance, with Event names, for the Leader's confirm. Read-only. An
// already-undone check-in has nothing left to roll back.
func PreviewUndo(ctx context.Context, fs *firestore.Client, meetingID, attendanceID string) ([]contract.RSVPRollbackItem, error) {
	if !validID(meetingID) {
		return nil, ErrNoMeeting
	}
	if !validID(attendanceID) {
		return nil, ErrNoAttendance
	}
	meetingRef := fs.Collection(Meetings).Doc(meetingID)
	if _, err := meetingRef.Get(ctx); status.Code(err) == codes.NotFound {
		return nil, ErrNoMeeting
	} else if err != nil {
		return nil, fmt.Errorf("meeting: read meetings/%s: %w", meetingID, err)
	}
	snap, err := meetingRef.Collection(Attendance).Doc(attendanceID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, ErrNoAttendance
	} else if err != nil {
		return nil, fmt.Errorf("meeting: read attendance/%s: %w", attendanceID, err)
	}
	var a contract.Attendance
	if err := snap.DataTo(&a); err != nil {
		return nil, fmt.Errorf("meeting: decode attendance/%s: %w", attendanceID, err)
	}
	rolls, err := rsvp.PreviewRollback(ctx, fs, a.ScoutID, attendanceID)
	if err != nil {
		return nil, err
	}
	items := make([]contract.RSVPRollbackItem, 0, len(rolls))
	for _, r := range rolls {
		item := contract.RSVPRollbackItem{EventID: r.EventID, From: r.From, To: r.To}
		es, err := fs.Collection("events").Doc(r.EventID).Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("meeting: read events/%s: %w", r.EventID, err)
		}
		var e contract.Event
		if err := es.DataTo(&e); err != nil {
			return nil, fmt.Errorf("meeting: decode events/%s: %w", r.EventID, err)
		}
		item.EventName = e.Name
		items = append(items, item)
	}
	return items, nil
}

func validID(id string) bool { return id != "" && !strings.Contains(id, "/") }
