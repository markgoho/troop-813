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

// History is GET /api/scouts/{id}/history: 200 ScoutHistory; 404 not_found.
func (h Handlers) History(w http.ResponseWriter, r *http.Request) {
	out, err := History(r.Context(), h.FS, r.PathValue("id"))
	switch {
	case errors.Is(err, ErrScoutNotFound):
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	case err != nil:
		router.InternalError(r.Context(), h.Logger, w, "scout history failed", err)
	default:
		router.WriteJSON(w, http.StatusOK, out)
	}
}

// Attendance is GET /api/scouts/attendance: 200 ScoutAttendance.
func (h Handlers) Attendance(w http.ResponseWriter, r *http.Request) {
	counts, err := Attendances(r.Context(), h.FS)
	if err != nil {
		router.InternalError(r.Context(), h.Logger, w, "scout attendance failed", err)
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
	if err != nil {
		h.fail(w, r, "add scout", err)
		return
	}
	router.Logger(h.Logger).InfoContext(r.Context(), "scout added", "scoutId", id)
	router.WriteJSON(w, http.StatusCreated, contract.AddScoutResponse{ID: id})
}

// Edit is PUT /api/scouts/{id} with EditScoutRequest → 204; 400 bad_request,
// 404 not_found, 409 duplicate_scout (inactiveMatchId when that match is inactive).
func (h Handlers) Edit(w http.ResponseWriter, r *http.Request) {
	var req contract.EditScoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
		return
	}
	h.done(w, r, "edit scout", Rename(r.Context(), h.FS, r.PathValue("id"), req.FirstName, req.LastName))
}

// Deactivate is POST /api/scouts/{id}/deactivate → 204.
func (h Handlers) Deactivate(w http.ResponseWriter, r *http.Request) {
	h.done(w, r, "deactivate scout", Deactivate(r.Context(), h.FS, r.PathValue("id")))
}

// Reactivate is POST /api/scouts/{id}/reactivate → 204; 409 duplicate_scout
// while an active Scout has the same name.
func (h Handlers) Reactivate(w http.ResponseWriter, r *http.Request) {
	h.done(w, r, "reactivate scout", Reactivate(r.Context(), h.FS, r.PathValue("id")))
}

// Delete is DELETE /api/scouts/{id} → 204; 409 in_use while the Scout has
// Attendance or RSVPs.
func (h Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	h.done(w, r, "delete scout", Delete(r.Context(), h.FS, r.PathValue("id")))
}

func (h Handlers) done(w http.ResponseWriter, r *http.Request, op string, err error) {
	if err != nil {
		h.fail(w, r, op, err)
		return
	}
	router.Logger(h.Logger).InfoContext(r.Context(), op, "scoutId", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

func (h Handlers) fail(w http.ResponseWriter, r *http.Request, op string, err error) {
	var dup *DuplicateError
	switch {
	case errors.Is(err, ErrEmptyName):
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
	case errors.Is(err, ErrNotFound):
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	case errors.Is(err, ErrInUse):
		router.WriteError(w, http.StatusConflict, contract.ErrorCodeInUse)
	case errors.As(err, &dup):
		body := contract.ErrorBody{Code: contract.ErrorCodeDuplicateScout}
		if !dup.Active {
			body.InactiveMatchID = dup.MatchID
		}
		router.WriteJSON(w, http.StatusConflict, body)
	default:
		router.InternalError(r.Context(), h.Logger, w, op+" failed", err)
	}
}
