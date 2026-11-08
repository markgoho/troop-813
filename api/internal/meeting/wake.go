package meeting

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"cloud.google.com/go/firestore"

	"github.com/markgoho/troop-813/api/internal/auth"
	"github.com/markgoho/troop-813/api/internal/clock"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
)

var (
	// ErrNoSession means no Kiosk session is live (none started, or all replaced).
	ErrNoSession = errors.New("meeting: no live Kiosk session")
	// ErrSessionOver means the live session's Meeting ended at 22:00.
	ErrSessionOver = errors.New("meeting: Kiosk session is past 22:00")
)

// Wake restarts the live Kiosk session's 30 idle minutes (lastActivityAt =
// now), so an asleep tablet works again. Refused with ErrSessionOver from
// 22:00 on the session's Meeting date, ErrNoSession when nothing is live. The
// live session is the one not replaced (Start keeps exactly one).
func Wake(ctx context.Context, fs *firestore.Client) error {
	now := clock.Now(ctx)
	return fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		live, err := tx.Documents(fs.Collection(auth.KioskSessions).Where("replaced", "==", false)).GetAll()
		if err != nil {
			return fmt.Errorf("meeting: read live kiosk sessions: %w", err)
		}
		if len(live) == 0 {
			return ErrNoSession
		}
		var s contract.KioskSession
		if err := live[0].DataTo(&s); err != nil {
			return fmt.Errorf("meeting: decode %s: %w", live[0].Ref.Path, err)
		}
		ends, err := clock.MeetingEnds(s.MeetingID)
		if err != nil {
			return fmt.Errorf("meeting: %s: %w", live[0].Ref.Path, err)
		}
		if !now.Before(ends) {
			return ErrSessionOver
		}
		return tx.Update(live[0].Ref, []firestore.Update{{Path: "lastActivityAt", Value: now}})
	})
}

// Wake is POST /api/kiosk/wake (Leader): 204; 404 not_found with no live
// session; 409 session_expired once the Meeting is over (22:00).
func (h Handlers) Wake(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := auth.UID(router.CallerFrom(ctx))
	err := Wake(ctx, h.FS)
	switch {
	case errors.Is(err, ErrNoSession):
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	case errors.Is(err, ErrSessionOver):
		router.WriteError(w, http.StatusConflict, contract.ErrorCodeSessionExpired)
	case err != nil:
		h.logger().ErrorContext(ctx, "wake kiosk failed", "err", err)
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
	default:
		h.logger().InfoContext(ctx, "kiosk woken", "uid", uid)
		w.WriteHeader(http.StatusNoContent)
	}
}
