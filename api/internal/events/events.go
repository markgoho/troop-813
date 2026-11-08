// Package events serves the Leader's Events tab: Event add, edit, cancel /
// uncancel, delete-when-unused, and the Event page's RSVP switch (which
// goes through the rsvp module). HQ reads events and rsvps itself through
// onSnapshot; these routes only write.
package events

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/markgoho/troop-813/api/internal/auth"
	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/roster"
	"github.com/markgoho/troop-813/api/internal/router"
	"github.com/markgoho/troop-813/api/internal/rsvp"
)

// Collection is the events collection.
const Collection = "events"

// Handlers serves the Events routes. Mount them Leader-only.
type Handlers struct {
	FS *firestore.Client
}

var (
	errInvalid = errors.New("events: invalid Event")
	errInUse   = errors.New("events: Event has RSVPs")
)

// Create is POST /api/events → 201 EventCreated. New Events are not cancelled.
func (h Handlers) Create(w http.ResponseWriter, r *http.Request) {
	e, ok := decodeEvent(w, r)
	if !ok {
		return
	}
	ref, _, err := h.FS.Collection(Collection).Add(r.Context(), e)
	if err != nil {
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
		return
	}
	router.WriteJSON(w, http.StatusCreated, contract.EventCreated{ID: ref.ID})
}

// Update is PUT /api/events/{id} → 204. It replaces name, dates and
// location (empty removes it) and keeps cancelled.
func (h Handlers) Update(w http.ResponseWriter, r *http.Request) {
	e, ok := decodeEvent(w, r)
	if !ok {
		return
	}
	var location any = e.Location
	if e.Location == "" {
		location = firestore.Delete
	}
	h.update(w, r, []firestore.Update{
		{Path: "name", Value: e.Name},
		{Path: "startDate", Value: e.StartDate},
		{Path: "endDate", Value: e.EndDate},
		{Path: "location", Value: location},
	})
}

// Cancel is POST /api/events/{id}/cancel → 204. The Kiosk stops asking about it.
func (h Handlers) Cancel(w http.ResponseWriter, r *http.Request) {
	h.update(w, r, []firestore.Update{{Path: "cancelled", Value: true}})
}

// Uncancel is POST /api/events/{id}/uncancel → 204.
func (h Handlers) Uncancel(w http.ResponseWriter, r *http.Request) {
	h.update(w, r, []firestore.Update{{Path: "cancelled", Value: false}})
}

func (h Handlers) update(w http.ResponseWriter, r *http.Request, u []firestore.Update) {
	_, err := h.FS.Collection(Collection).Doc(r.PathValue("id")).Update(r.Context(), u)
	writeResult(w, err)
}

// Delete is DELETE /api/events/{id} → 204, or 409 in_use while any RSVP doc
// exists for it, even a cleared one: its history would be lost (ADR 0004).
// Cancel is the way to take an answered Event off.
func (h Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ref := h.FS.Collection(Collection).Doc(id)
	err := h.FS.RunTransaction(r.Context(), func(_ context.Context, tx *firestore.Transaction) error {
		if _, err := tx.Get(ref); err != nil {
			return err
		}
		used, err := tx.Documents(h.FS.Collection(rsvp.Collection).Where("eventId", "==", id).Limit(1)).GetAll()
		if err != nil {
			return err
		}
		if len(used) > 0 {
			return errInUse
		}
		return tx.Delete(ref)
	})
	writeResult(w, err)
}

// SetRSVP is PUT /api/events/{id}/rsvps/{scoutId} with SetRSVP → 204: a
// Leader sets, changes or (answer null) clears a Scout's RSVP.
func (h Handlers) SetRSVP(w http.ResponseWriter, r *http.Request) {
	var body contract.SetRSVP
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&body); err != nil {
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
		return
	}
	event := h.FS.Collection(Collection).Doc(r.PathValue("id"))
	scout := h.FS.Collection(roster.Scouts).Doc(r.PathValue("scoutId"))
	uid := auth.UID(router.CallerFrom(r.Context()))
	writeResult(w, rsvp.SetByLeader(r.Context(), h.FS, event, scout, body.Answer, uid))
}

func writeResult(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case status.Code(err) == codes.NotFound, errors.Is(err, rsvp.ErrEventNotFound), errors.Is(err, rsvp.ErrScoutNotFound):
		router.WriteError(w, http.StatusNotFound, contract.ErrorCodeNotFound)
	case errors.Is(err, errInvalid), errors.Is(err, rsvp.ErrInvalidAnswer):
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
	case errors.Is(err, errInUse):
		router.WriteError(w, http.StatusConflict, contract.ErrorCodeInUse)
	default:
		router.WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
	}
}

// decodeEvent reads an EventInput and checks it, writing 400 if it fails.
func decodeEvent(w http.ResponseWriter, r *http.Request) (contract.Event, bool) {
	var in contract.EventInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&in); err != nil {
		router.WriteError(w, http.StatusBadRequest, contract.ErrorCodeBadRequest)
		return contract.Event{}, false
	}
	e, err := validate(in)
	if err != nil {
		writeResult(w, err)
		return contract.Event{}, false
	}
	return e, true
}

// validate trims the text fields and checks the rules HQ's form also checks:
// a name, real YYYY-MM-DD dates, and an end no earlier than the start.
func validate(in contract.EventInput) (contract.Event, error) {
	e := contract.Event{
		Name:      strings.TrimSpace(in.Name),
		StartDate: in.StartDate,
		EndDate:   in.EndDate,
		Location:  strings.TrimSpace(in.Location),
	}
	start, err1 := time.Parse(time.DateOnly, e.StartDate)
	end, err2 := time.Parse(time.DateOnly, e.EndDate)
	if e.Name == "" || err1 != nil || err2 != nil || end.Before(start) {
		return contract.Event{}, errInvalid
	}
	return e, nil
}
