package access

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/markgoho/troop-813/api/internal/auth"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
)

// Handlers serves the Access tab. Mount List, SetRole and Dismiss Admin-only.
type Handlers struct {
	FS *firestore.Client
}

// List is GET /api/users: everyone who has signed in, newest first, minus
// Parents whose Waiting request the Admin dismissed.
func (h Handlers) List(w http.ResponseWriter, r *http.Request) {
	snaps, err := h.FS.Collection(auth.Users).OrderBy("createdAt", firestore.Desc).Documents(r.Context()).GetAll()
	if err != nil {
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
		return
	}
	out := contract.UserList{Users: make([]contract.UserSummary, 0, len(snaps))}
	for _, s := range snaps {
		var u contract.User
		if err := s.DataTo(&u); err != nil {
			router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
			return
		}
		if u.Dismissed && u.Role == contract.RoleParent {
			continue
		}
		out.Users = append(out.Users, contract.UserSummary{
			UID: s.Ref.ID, Email: u.Email, DisplayName: u.DisplayName, Role: u.Role,
			CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	router.WriteJSON(w, http.StatusOK, out)
}

var (
	errFixed      = errors.New("access: admin role is fixed")
	errNotWaiting = errors.New("access: only a Parent's request can be dismissed")
)

// SetRole is PUT /api/users/{uid}/role: Parent↔Leader. Never touches an
// Admin's row (so the Admin can't lock themselves out) and never sets admin.
// auth reads the role on every request, so the change applies on the next one.
func (h Handlers) SetRole(w http.ResponseWriter, r *http.Request) {
	var body contract.SetRole
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
		(body.Role != contract.RoleParent && body.Role != contract.RoleLeader) {
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
		return
	}
	uid := r.PathValue("uid")
	if uid == auth.UID(router.CallerFrom(r.Context())) {
		router.WriteError(w, http.StatusConflict, contract.ErrorCodeFixedRole)
		return
	}
	ref := h.FS.Collection(auth.Users).Doc(uid)
	err := h.FS.RunTransaction(r.Context(), func(_ context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		if role, _ := snap.DataAt("role"); role == string(contract.RoleAdmin) {
			return errFixed
		}
		return tx.Update(ref, []firestore.Update{
			{Path: "role", Value: body.Role},
			{Path: "dismissed", Value: firestore.Delete},
		})
	})
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, errFixed):
		router.WriteError(w, http.StatusConflict, contract.ErrorCodeFixedRole)
	case status.Code(err) == codes.NotFound:
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	default:
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
	}
}

// Dismiss is POST /api/users/{uid}/dismiss: hide a Parent's Waiting request
// (an accidental sign-in, someone who isn't a Leader). Their users doc stays,
// so signing in again doesn't bring the request back and Parent access is
// unchanged. Leaders and Admins are refused: demote a Leader instead.
func (h Handlers) Dismiss(w http.ResponseWriter, r *http.Request) {
	ref := h.FS.Collection(auth.Users).Doc(r.PathValue("uid"))
	err := h.FS.RunTransaction(r.Context(), func(_ context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		if role, _ := snap.DataAt("role"); role != string(contract.RoleParent) {
			return errNotWaiting
		}
		return tx.Update(ref, []firestore.Update{{Path: "dismissed", Value: true}})
	})
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, errNotWaiting):
		router.WriteError(w, http.StatusConflict, contract.ErrorCodeNotWaiting)
	case status.Code(err) == codes.NotFound:
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	default:
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
	}
}
