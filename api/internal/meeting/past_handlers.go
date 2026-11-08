package meeting

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/markgoho/troop-813/api/internal/auth"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
)

// CreatePast is POST /api/meetings (Leader): 201 PastMeetingCreated; 400
// bad_request for a bad body, a non-date, today or a future date; 409
// meeting_exists.
func (h Handlers) CreatePast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := auth.UID(router.CallerFrom(ctx))
	var req contract.PastMeetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
		return
	}
	err := CreatePast(ctx, h.FS, uid, req.Date)
	switch {
	case errors.Is(err, ErrNotPast):
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
	case errors.Is(err, ErrMeetingExists):
		router.WriteError(w, http.StatusConflict, contract.ErrorCodeMeetingExists)
	case err != nil:
		router.InternalError(ctx, h.Logger, w, "create past meeting failed", err, "meetingId", req.Date)
	default:
		router.Logger(h.Logger).InfoContext(ctx, "past meeting created", "meetingId", req.Date, "uid", uid)
		router.WriteJSON(w, http.StatusCreated, contract.PastMeetingCreated{ID: req.Date})
	}
}

// DeleteMeeting is DELETE /api/meetings/{date} (Leader): 204; 404 not_found;
// 409 in_use while any Scout is present.
func (h Handlers) DeleteMeeting(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := auth.UID(router.CallerFrom(ctx))
	date := r.PathValue("date")
	err := Delete(ctx, h.FS, date)
	switch {
	case errors.Is(err, ErrNoMeeting):
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	case errors.Is(err, ErrHasAttendance):
		router.WriteError(w, http.StatusConflict, contract.ErrorCodeInUse)
	case err != nil:
		router.InternalError(ctx, h.Logger, w, "delete meeting failed", err, "meetingId", date)
	default:
		router.Logger(h.Logger).InfoContext(ctx, "meeting deleted", "meetingId", date, "uid", uid)
		w.WriteHeader(http.StatusNoContent)
	}
}
