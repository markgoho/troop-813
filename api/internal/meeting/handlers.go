package meeting

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"cloud.google.com/go/firestore"
	fbauth "firebase.google.com/go/v4/auth"

	"github.com/markgoho/troop-813/api/internal/auth"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
)

// Handlers serves Start (Admin) and the Kiosk's check-in (Kiosk).
type Handlers struct {
	FS     *firestore.Client
	Auth   *fbauth.Client // mints the Kiosk custom token
	Logger *slog.Logger   // nil discards; log IDs, never names
}

func (h Handlers) logger() *slog.Logger {
	if h.Logger == nil {
		return slog.New(slog.DiscardHandler)
	}
	return h.Logger
}

// kioskUID is the Firebase Auth uid a Kiosk session signs in as. One per
// session, so no two sessions ever share an Auth user.
func kioskUID(sessionID string) string { return "kiosk-" + sessionID }

// Start is POST /api/kiosk/start (Admin only): 200 StartMeetingResponse with
// the Kiosk custom token; 409 too_late from 22:00.
func (h Handlers) Start(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	admin := auth.UID(router.CallerFrom(ctx))
	started, err := Start(ctx, h.FS, admin)
	if errors.Is(err, ErrTooLate) {
		router.WriteError(w, http.StatusConflict, contract.ErrorCodeTooLate)
		return
	}
	if err != nil {
		h.logger().ErrorContext(ctx, "start meeting failed", "err", err)
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
		return
	}
	token, err := h.Auth.CustomTokenWithClaims(ctx, kioskUID(started.SessionID), map[string]any{
		"role":           auth.KioskRole,
		"kioskSessionId": started.SessionID,
		"meetingId":      started.MeetingID,
	})
	if err != nil {
		// The session exists but the tablet can't use it; Start again replaces it.
		h.logger().ErrorContext(ctx, "mint kiosk token failed", "err", err, "kioskSessionId", started.SessionID)
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
		return
	}
	h.logger().InfoContext(ctx, "meeting started",
		"meetingId", started.MeetingID, "kioskSessionId", started.SessionID, "created", started.Created, "uid", admin)
	router.WriteJSON(w, http.StatusOK, contract.StartMeetingResponse{
		MeetingID:      started.MeetingID,
		KioskSessionID: started.SessionID,
		Token:          token,
	})
}

// KioskNotFound is POST /api/kiosk/not-found (Kiosk only): 204; 400
// bad_request for a bad body; 404 not_found for an unknown Scout or Meeting.
func (h Handlers) KioskNotFound(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	k, ok := router.CallerFrom(ctx).(auth.Kiosk)
	if !ok {
		router.WriteError(w, http.StatusForbidden, contract.ErrorCodeForbidden)
		return
	}
	var req contract.KioskNotFoundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ScoutID == "" {
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
		return
	}
	err := MarkNotFound(ctx, h.FS, k, req.ScoutID)
	switch {
	case errors.Is(err, ErrScoutNotFound), errors.Is(err, ErrNoMeeting):
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	case err != nil:
		h.logger().ErrorContext(ctx, "mark not found failed", "err", err, "meetingId", k.MeetingID, "kioskSessionId", k.SessionID)
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
	default:
		h.logger().InfoContext(ctx, "marked not found",
			"meetingId", k.MeetingID, "scoutId", req.ScoutID, "kioskSessionId", k.SessionID)
		w.WriteHeader(http.StatusNoContent)
	}
}

// KioskCheckIn is POST /api/kiosk/check-in (Kiosk only): 200 CheckInResponse
// (new or existing Attendance); 400 bad_request for a bad body or method;
// 404 not_found for an unknown Scout or a missing Meeting doc.
func (h Handlers) KioskCheckIn(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	k, ok := router.CallerFrom(ctx).(auth.Kiosk)
	if !ok {
		router.WriteError(w, http.StatusForbidden, contract.ErrorCodeForbidden)
		return
	}
	var req contract.KioskCheckInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ScoutID == "" {
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
		return
	}
	res, err := CheckInAtKiosk(ctx, h.FS, k, req.ScoutID, req.Method)
	switch {
	case errors.Is(err, ErrInvalidMethod):
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
	case errors.Is(err, ErrScoutNotFound), errors.Is(err, ErrNoMeeting):
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	case err != nil:
		h.logger().ErrorContext(ctx, "kiosk check-in failed", "err", err, "meetingId", k.MeetingID, "kioskSessionId", k.SessionID)
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
	default:
		h.logger().InfoContext(ctx, "checked in",
			"meetingId", k.MeetingID, "scoutId", req.ScoutID, "attendanceId", res.AttendanceID,
			"method", req.Method, "kioskSessionId", k.SessionID, "repeat", res.Repeat)
		router.WriteJSON(w, http.StatusOK, contract.CheckInResponse{AttendanceID: res.AttendanceID})
	}
}
