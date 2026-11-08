package roster

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"cloud.google.com/go/firestore"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
)

// Handlers serves the Roster tab. Mount its routes Leader-only.
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

// History is GET /api/scouts/{id}/history: 200 ScoutHistory; 404 not_found.
func (h Handlers) History(w http.ResponseWriter, r *http.Request) {
	out, err := History(r.Context(), h.FS, r.PathValue("id"))
	switch {
	case errors.Is(err, ErrScoutNotFound):
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	case err != nil:
		h.logger().ErrorContext(r.Context(), "scout history failed", "err", err)
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
	default:
		router.WriteJSON(w, http.StatusOK, out)
	}
}

// Attendance is GET /api/scouts/attendance: 200 ScoutAttendance.
func (h Handlers) Attendance(w http.ResponseWriter, r *http.Request) {
	counts, err := Attendances(r.Context(), h.FS)
	if err != nil {
		h.logger().ErrorContext(r.Context(), "scout attendance failed", "err", err)
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
		return
	}
	router.WriteJSON(w, http.StatusOK, contract.ScoutAttendance{Scouts: counts})
}

// Add is POST /api/scouts: 201 AddScoutResponse; 400 bad_request for a bad
// body or a blank name; 409 duplicate_scout for an active exact match.
func (h Handlers) Add(w http.ResponseWriter, r *http.Request) {
	var req contract.AddScoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
		return
	}
	id, err := Add(r.Context(), h.FS, req.FirstName, req.LastName)
	var dup *DuplicateError
	switch {
	case errors.Is(err, ErrEmptyName):
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
	case errors.As(err, &dup):
		router.WriteError(w, http.StatusConflict, contract.ErrorCodeDuplicateScout)
	case err != nil:
		h.logger().ErrorContext(r.Context(), "add scout failed", "err", err)
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
	default:
		h.logger().InfoContext(r.Context(), "scout added", "scoutId", id)
		router.WriteJSON(w, http.StatusCreated, contract.AddScoutResponse{ID: id})
	}
}
