package meeting

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/markgoho/troop-813/api/internal/auth"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
)

// LeaderCheckIn is POST /api/meetings/{date}/check-ins (Leader): 200
// CheckInResponse (new or existing Attendance); 400 bad_request for no
// scoutId; 404 not_found for an unknown Scout or Meeting.
func (h Handlers) LeaderCheckIn(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := auth.UID(router.CallerFrom(ctx))
	date := r.PathValue("date")
	var req contract.LeaderCheckInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ScoutID == "" {
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
		return
	}
	res, err := CheckInByLeader(ctx, h.FS, uid, date, req.ScoutID)
	switch {
	case errors.Is(err, ErrScoutNotFound), errors.Is(err, ErrNoMeeting):
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	case err != nil:
		router.InternalError(ctx, h.Logger, w, "leader check-in failed", err, "meetingId", date)
	default:
		router.Logger(h.Logger).InfoContext(ctx, "checked in",
			"meetingId", date, "scoutId", req.ScoutID, "attendanceId", res.AttendanceID,
			"method", contract.AttendanceMethodLeader, "uid", uid, "repeat", res.Repeat)
		router.WriteJSON(w, http.StatusOK, contract.CheckInResponse{AttendanceID: res.AttendanceID})
	}
}

// RSVPRollbackPreview is GET /api/meetings/{date}/attendance/{id}/rsvp-rollback
// (Leader): 200 RSVPRollbackPreview, the answers Undo would roll back; 404
// not_found for an unknown Meeting or Attendance.
func (h Handlers) RSVPRollbackPreview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	date, id := r.PathValue("date"), r.PathValue("id")
	items, err := PreviewUndo(ctx, h.FS, date, id)
	switch {
	case errors.Is(err, ErrNoMeeting), errors.Is(err, ErrNoAttendance):
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	case err != nil:
		router.InternalError(ctx, h.Logger, w, "rsvp rollback preview failed", err, "meetingId", date, "attendanceId", id)
	default:
		router.WriteJSON(w, http.StatusOK, contract.RSVPRollbackPreview{Items: items})
	}
}

// UndoCheckIn is POST /api/meetings/{date}/attendance/{id}/undo (Leader):
// 204; 404 not_found for an unknown Meeting or Attendance.
func (h Handlers) UndoCheckIn(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := auth.UID(router.CallerFrom(ctx))
	date, id := r.PathValue("date"), r.PathValue("id")
	err := UndoCheckIn(ctx, h.FS, uid, date, id)
	switch {
	case errors.Is(err, ErrNoMeeting), errors.Is(err, ErrNoAttendance):
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	case err != nil:
		router.InternalError(ctx, h.Logger, w, "undo check-in failed", err, "meetingId", date, "attendanceId", id)
	default:
		router.Logger(h.Logger).InfoContext(ctx, "check-in undone", "meetingId", date, "attendanceId", id, "uid", uid)
		w.WriteHeader(http.StatusNoContent)
	}
}
