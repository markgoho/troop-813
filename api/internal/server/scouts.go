package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"cloud.google.com/go/firestore"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/roster"
	"github.com/markgoho/troop-813/api/internal/router"
)

// addScout handles POST /api/scouts. Logs carry the new ID, never the name.
func addScout(fs *firestore.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req contract.AddScoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeInvalidRequest)
			return
		}
		id, err := roster.Add(r.Context(), fs, req.FirstName, req.LastName)
		var dup *roster.DuplicateError
		switch {
		case errors.Is(err, roster.ErrEmptyName):
			router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeInvalidRequest)
		case errors.As(err, &dup):
			router.WriteError(w, http.StatusConflict, contract.ErrorCodeDuplicateScout)
		case err != nil:
			logger.ErrorContext(r.Context(), "add scout failed", "err", err)
			router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
		default:
			logger.InfoContext(r.Context(), "scout added", "scoutId", id)
			router.WriteJSON(w, http.StatusCreated, contract.AddScoutResponse{ID: id})
		}
	}
}
