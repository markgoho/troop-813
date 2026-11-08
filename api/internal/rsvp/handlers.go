package rsvp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"

	"github.com/markgoho/troop-813/api/internal/auth"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
)

// Handlers serves the Kiosk's RSVP route.
type Handlers struct {
	FS     *firestore.Client
	Logger *slog.Logger // nil discards; log IDs, never names
}

func (h Handlers) logger() *slog.Logger {
	if h.Logger == nil {
		return slog.New(slog.DiscardHandler)
	}
	return h.Logger
}

func validID(id string) bool { return id != "" && !strings.Contains(id, "/") }

// KioskSet is PUT /api/kiosk/rsvps (Kiosk only): 204; 400 bad_request for a
// bad body, an answer other than going/notGoing, or an Event that is
// cancelled or has begun; 403 forbidden for a Scout this session didn't
// check in; 404 not_found for an unknown Event or Meeting.
func (h Handlers) KioskSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	k, ok := router.CallerFrom(ctx).(auth.Kiosk)
	if !ok {
		router.WriteError(w, http.StatusForbidden, contract.ErrorCodeForbidden)
		return
	}
	var req contract.KioskSetRSVPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validID(req.EventID) || !validID(req.ScoutID) {
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
		return
	}
	err := SetAtKiosk(ctx, h.FS, Kiosk{SessionID: k.SessionID, MeetingID: k.MeetingID}, req.EventID, req.ScoutID, req.Answer)
	switch {
	case errors.Is(err, ErrInvalidAnswer), errors.Is(err, ErrEventClosed):
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
	case errors.Is(err, ErrNotCheckedIn):
		router.WriteError(w, http.StatusForbidden, contract.ErrorCodeForbidden)
	case errors.Is(err, ErrEventNotFound), errors.Is(err, ErrNoMeeting):
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	case err != nil:
		h.logger().ErrorContext(ctx, "kiosk rsvp failed", "err", err, "meetingId", k.MeetingID, "kioskSessionId", k.SessionID)
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
	default:
		h.logger().InfoContext(ctx, "rsvp recorded",
			"meetingId", k.MeetingID, "eventId", req.EventID, "scoutId", req.ScoutID,
			"answer", req.Answer, "kioskSessionId", k.SessionID)
		w.WriteHeader(http.StatusNoContent)
	}
}
