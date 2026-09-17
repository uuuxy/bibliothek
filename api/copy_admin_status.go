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
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		ctx := r.Context()

		if err := bookRepo.UpdateCopyDamageNote(ctx, id, req.Note); err != nil {
			if errors.Is(err, repository.ErrExemplarNichtGefunden) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
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
	// ZustandAbwertungProzent ist der Beschädigungsgrad dieses Exemplars (Migration 127,
	// Anforderungsliste Nr. 2). Ein ZEIGER, weil das Feld drei Zustände hat: ein Wert
	// setzt, 0 setzt auf null zurück, und FEHLT heißt „unangetastet". Ohne diese
	// Unterscheidung löschte jeder andere Aufruf dieser Tür einen erfassten Wasserschaden.
	//
	// 0–100, weil die Spalte es auch tut (chk_zustand_abwertung_bereich): Ohne Prüfung
	// hier käme eine 140 als 500 zurück statt als Auskunft, was erlaubt ist.
	ZustandAbwertungProzent *int `json:"zustand_abwertung_prozent" validate:"omitempty,min=0,max=100"`
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
func (s *Server) UpdateCopyStatusHandler(bookRepo repository.BookRepository, bescheidRepo repository.BescheidRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		var req UpdateStatusRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		ctx := r.Context()

		// Wenn ein Buch manuell auf "Verfügbar" gesetzt wird, zwingend Notizen und Ausgesondert-Flag löschen
		//
		// Der Beschädigungsgrad wird dabei AUSDRÜCKLICH nicht geräumt: Er ist eine
		// Eigenschaft des Buchs, kein Status. Ein Band mit Wasserrand darf ausleihbar sein
		// und trägt seinen Abschlag weiter — sonst verlangte die Schule beim nächsten
		// Verlust wieder den vollen Zeitwert für ein sichtbar beschädigtes Buch.
		if req.IstAusleihbar {
			req.ZustandNotiz = ""
			req.IstAusgesondert = false
		}

		if err := bookRepo.UpdateCopyStatus(ctx, id, req.IstAusleihbar, req.IstAusgesondert,
			req.ZustandNotiz, req.ZustandAbwertungProzent); err != nil {
			if errors.Is(err, repository.ErrExemplarNochVerliehen) {
				apierrors.SendHTTPError(w, http.StatusBadRequest, err)
				return
			}
			if errors.Is(err, repository.ErrExemplarNichtGefunden) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Die Antwort trägt den NEUEN Ersatzwert zurück (OFFEN.md 9.8, Stufe 2b):
		// Wer den Wertverlust gerade eingetragen hat, sieht sofort, was das Buch damit
		// noch wert ist. Gerechnet wird dabei am Server, mit derselben Funktion wie im
		// Melde-Dialog — die Oberfläche soll diese Zahl nie selbst ausrechnen.
		//
		// Scheitert die Nachfrage, bleibt es bei der Erfolgsmeldung: Gespeichert ist
		// gespeichert, und eine fehlende Auskunft darf daraus keinen Fehler machen.
		antwort := map[string]any{"status": "success"}
		if g, groessenErr := bescheidRepo.GroessenFuerExemplar(ctx, id); groessenErr == nil {
			v := ersatzwertVorschlagAus(g, s.preisquelle(ctx))
			antwort["ersatzwert"] = v.Betrag
			antwort["ersatzwert_herleitung"] = v.Herleitung
		}
		RespondJSON(w, http.StatusOK, antwort)
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
			// Guard 31.08.2026: verliehen -> 400 mit Auskunft, unbekannt -> 404 --
			// vorher sonderte diese Tuer auch verliehene Exemplare kommentarlos aus.
			if errors.Is(err, repository.ErrExemplarNochVerliehen) {
				apierrors.SendHTTPError(w, http.StatusBadRequest, err)
				return
			}
			if errors.Is(err, repository.ErrExemplarNichtGefunden) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondSuccess(w)
	}
}
