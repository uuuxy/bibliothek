package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// DamageNoteRequest holds the payload for updating a copy's damage note.
type DamageNoteRequest struct {
	Note string `json:"note"`
}

// UpdateDamageNoteHandler updates the physical condition note of a book copy.
// @Summary      Update damage note
// @Description  Updates the custom damage or condition note text of a physical book copy.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string             true  "Book copy ID (UUID)"
// @Param        body  body      DamageNoteRequest  true  "Damage note payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /buecher/exemplare/{id}/schadensnotiz [post]
func (s *Server) UpdateDamageNoteHandler(bookRepo repository.BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		var req DamageNoteRequest
		if !DecodeJSON(w, r, &req) {
			return
		}

		ctx := r.Context()

		if err := bookRepo.UpdateCopyDamageNote(ctx, id, req.Note); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondSuccess(w)
	}
}

// UpdateStatusRequest holds the payload for updating a copy's status.
type UpdateStatusRequest struct {
	IstAusleihbar   bool   `json:"ist_ausleihbar"`
	IstAusgesondert bool   `json:"ist_ausgesondert"`
	ZustandNotiz    string `json:"zustand_notiz"`
}

// UpdateCopyStatusHandler updates the status of a physical book copy.
// @Summary      Update copy status
// @Description  Updates the status (ist_ausleihbar, ist_ausgesondert) and the condition note of a physical book copy.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string                true  "Book copy ID (UUID)"
// @Param        body  body      UpdateStatusRequest   true  "New status payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /buecher/exemplare/{id}/status [put]
func (s *Server) UpdateCopyStatusHandler(bookRepo repository.BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		var req UpdateStatusRequest
		if !DecodeJSON(w, r, &req) {
			return
		}

		ctx := r.Context()

		// Wenn ein Buch manuell auf "Verfügbar" gesetzt wird, zwingend Notizen und Ausgesondert-Flag löschen
		if req.IstAusleihbar {
			req.ZustandNotiz = ""
			req.IstAusgesondert = false
		}

		if err := bookRepo.UpdateCopyStatus(ctx, id, req.IstAusleihbar, req.IstAusgesondert, req.ZustandNotiz); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondSuccess(w)
	}
}

// AussondernCopyHandler marks a physical copy as decommissioned (ausgesondert).
// Decommissioned copies are hidden from catalog, kiosk, and inventory but kept for statistics.
// @Summary      Decommission a book copy
// @Description  Marks a physical copy as decommissioned: sets ist_ausgesondert=true and ist_ausleihbar=false.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Book copy ID (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /buecher/exemplare/{id}/aussondern [post]
func (s *Server) AussondernCopyHandler(bookRepo repository.BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		ctx := r.Context()

		if err := bookRepo.DecommissionCopy(ctx, id); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondSuccess(w)
	}
}
