package access

import (
	"net/http"

	"github.com/markgoho/troop-813/api/internal/auth"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
)

// Leaders is GET /api/leaders (any Leader): every Leader and the Admin by
// uid and name, never email, so screens can say which Leader did something.
func (h Handlers) Leaders(w http.ResponseWriter, r *http.Request) {
	snaps, err := h.FS.Collection(auth.Users).
		Where("role", "in", []string{string(contract.RoleLeader), string(contract.RoleAdmin)}).
		Documents(r.Context()).GetAll()
	if err != nil {
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
		return
	}
	out := contract.LeaderList{Leaders: make([]contract.LeaderSummary, 0, len(snaps))}
	for _, s := range snaps {
		var u contract.User
		if err := s.DataTo(&u); err != nil {
			router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
			return
		}
		out.Leaders = append(out.Leaders, contract.LeaderSummary{UID: s.Ref.ID, DisplayName: u.DisplayName})
	}
	router.WriteJSON(w, http.StatusOK, out)
}
