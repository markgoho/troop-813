package roster

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
)

const (
	meetings   = "meetings"   // duplicates meeting.Meetings: meeting may import roster
	attendance = "attendance" // the subcollection, queried across all Meetings
)

// ErrScoutNotFound means no Scout has that ID.
var ErrScoutNotFound = errors.New("roster: scout not found")

// History is the Scout's Meeting-by-Meeting attendance, newest first.
func History(ctx context.Context, fs *firestore.Client, scoutID string) (contract.ScoutHistory, error) {
	snap, err := fs.Collection(Scouts).Doc(scoutID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return contract.ScoutHistory{}, ErrScoutNotFound
	}
	if err != nil {
		return contract.ScoutHistory{}, fmt.Errorf("roster: read scouts/%s: %w", scoutID, err)
	}
	var s contract.Scout
	if err := snap.DataTo(&s); err != nil {
		return contract.ScoutHistory{}, fmt.Errorf("roster: decode scouts/%s: %w", scoutID, err)
	}
	present, _, err := presentDates(ctx, fs.CollectionGroup(attendance).Where("scoutId", "==", scoutID))
	if err != nil {
		return contract.ScoutHistory{}, err
	}
	all, err := meetingDates(ctx, fs)
	if err != nil {
		return contract.ScoutHistory{}, err
	}
	return buildHistory(clock.LocalDate(s.CreatedAt), present[scoutID], all), nil
}

// Attendances is every Scout's X of Y, keyed by Scout ID.
func Attendances(ctx context.Context, fs *firestore.Client) (map[string]contract.AttendanceCount, error) {
	scouts, err := fs.Collection(Scouts).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("roster: read scouts: %w", err)
	}
	present, recorded, err := presentDates(ctx, fs.CollectionGroup(attendance).Query)
	if err != nil {
		return nil, err
	}
	all, err := meetingDates(ctx, fs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]contract.AttendanceCount, len(scouts))
	for _, snap := range scouts {
		var s contract.Scout
		if err := snap.DataTo(&s); err != nil {
			return nil, fmt.Errorf("roster: decode scouts/%s: %w", snap.Ref.ID, err)
		}
		h := buildHistory(clock.LocalDate(s.CreatedAt), present[snap.Ref.ID], all)
		out[snap.Ref.ID] = contract.AttendanceCount{Present: h.Present, Total: h.Total, HasAttendance: recorded[snap.Ref.ID]}
	}
	return out, nil
}

// presentDates maps Scout ID to the set of Meeting dates with a non-undone
// Attendance, over the given collection-group query. recorded is the Scouts
// with any Attendance record, undone ones included (what blocks a Delete).
func presentDates(ctx context.Context, q firestore.Query) (map[string]map[string]bool, map[string]bool, error) {
	it := q.Documents(ctx)
	defer it.Stop()
	out := map[string]map[string]bool{}
	recorded := map[string]bool{}
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return out, recorded, nil
		}
		if err != nil {
			return nil, nil, fmt.Errorf("roster: read attendance: %w", err)
		}
		var a contract.Attendance
		if err := snap.DataTo(&a); err != nil {
			return nil, nil, fmt.Errorf("roster: decode %s: %w", snap.Ref.Path, err)
		}
		recorded[a.ScoutID] = true
		if a.UndoneAt != nil {
			continue
		}
		if out[a.ScoutID] == nil {
			out[a.ScoutID] = map[string]bool{}
		}
		out[a.ScoutID][snap.Ref.Parent.Parent.ID] = true
	}
}

// meetingDates is the ID (date) of every Meeting that exists.
func meetingDates(ctx context.Context, fs *firestore.Client) ([]string, error) {
	snaps, err := fs.Collection(meetings).Select().Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("roster: read meetings: %w", err)
	}
	out := make([]string, 0, len(snaps))
	for _, s := range snaps {
		out = append(out, s.Ref.ID)
	}
	return out, nil
}

// buildHistory applies the start rule: counting starts at the earlier of
// createdDate and the first present date, and covers every Meeting from then
// on. A present date always counts as a Meeting, even if its meetings doc is
// missing, so X never exceeds Y.
func buildHistory(createdDate string, present map[string]bool, all []string) contract.ScoutHistory {
	since := createdDate
	dates := map[string]bool{}
	for _, d := range all {
		dates[d] = true
	}
	for d := range present {
		since = min(since, d)
		dates[d] = true
	}
	h := contract.ScoutHistory{Since: since, Meetings: []contract.MeetingPresence{}}
	for d := range dates {
		if d >= since {
			h.Meetings = append(h.Meetings, contract.MeetingPresence{Date: d, Present: present[d]})
		}
	}
	slices.SortFunc(h.Meetings, func(a, b contract.MeetingPresence) int {
		return cmp.Compare(b.Date, a.Date)
	})
	h.Total = len(h.Meetings)
	for _, m := range h.Meetings {
		if m.Present {
			h.Present++
		}
	}
	return h
}
