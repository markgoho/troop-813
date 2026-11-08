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
		if err := tx.Update(attRef, []firestore.Update{
			{Path: "undoneAt", Value: now},
			{Path: "undoneBy", Value: leaderUID},
		}); err != nil {
			return err
		}
		if m.CheckedIn[a.ScoutID] != attendanceID {
			return nil
		}
		return tx.Update(meetingRef, []firestore.Update{
			{FieldPath: firestore.FieldPath{"checkedIn", a.ScoutID}, Value: firestore.Delete},
		})
	})
}

func validID(id string) bool { return id != "" && !strings.Contains(id, "/") }
