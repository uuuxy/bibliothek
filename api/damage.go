package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
)

// DefektRequest is the payload for marking a book copy as defective.
type DefektRequest struct {
	LoanID       *string `json:"loan_id,omitempty"`
	SchuelerID   *string `json:"schueler_id,omitempty"`
	Betrag       float64 `json:"betrag"`
	Beschreibung string  `json:"beschreibung"`
}

// DefektResponse is returned after successfully recording a damage case.
type DefektResponse struct {
	Status     string `json:"status"`
	SchadensID string `json:"schadens_id"`
}

// MarkCopyDefektHandler marks a book copy as defective:
//  1. Sets ist_ausleihbar = false and records a damage note on the copy.
//  2. Creates a Schadensfaelle entry linked to the responsible student (if provided).
func (s *Server) MarkCopyDefektHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		copyID := r.PathValue("id")
		if copyID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}

		var req DefektRequest
		if !DecodeJSON(w, r, &req) {
			return
		}
		if req.Betrag < 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("betrag darf nicht negativ sein"))
			return
		}
		if req.Beschreibung == "" {
			req.Beschreibung = "Defekt/Schaden bei Rückgabe gemeldet"
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		// 1. Mark copy as not lendable and record damage note.
		res, err := tx.Exec(ctx, `
			UPDATE buecher_exemplare
			SET ist_ausleihbar = false,
			    zustand_notiz = $1,
			    aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $2
		`, req.Beschreibung, copyID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if res.RowsAffected() == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("book copy not found"))
			return
		}

		// 2. Create Schadensfaelle entry.
		var schadensID string
		if req.SchuelerID != nil && *req.SchuelerID != "" {
			err = tx.QueryRow(ctx, `
				INSERT INTO schadensfaelle
				    (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
			`, copyID, req.LoanID, req.SchuelerID, req.Beschreibung, req.Betrag).Scan(&schadensID)
		} else {
			// No identified student: associate with the acting staff member.
			err = tx.QueryRow(ctx, `
				INSERT INTO schadensfaelle
				    (exemplar_id, ausleihe_id, benutzer_id, beschreibung, betrag)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
			`, copyID, req.LoanID, claims.UserID, req.Beschreibung, req.Betrag).Scan(&schadensID)
		}
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, DefektResponse{Status: "ok", SchadensID: schadensID})
	}
}

// UndoReturnHandler reverses a recent return (within 1 hour) by nullifying rueckgabe_am.
func (s *Server) UndoReturnHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loanID := r.PathValue("id")
		if loanID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing loan ID parameter"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		res, err := s.DB.Pool.Exec(ctx, `
			UPDATE ausleihen
			SET rueckgabe_am = NULL, rueckgabe_bearbeiter_id = NULL
			WHERE id = $1
			  AND rueckgabe_am IS NOT NULL
			  AND rueckgabe_am > CURRENT_TIMESTAMP - INTERVAL '1 hour'
		`, loanID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if res.RowsAffected() == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound,
				errors.New("ausleihe nicht gefunden, nicht zurückgegeben oder Zeitfenster überschritten (max. 1 Stunde)"))
			return
		}

		RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// ReportDamageHandler handles POST /api/damage/report
// Sets ist_ausgesondert = true, inserts into schadensfaelle, and ends the loan.
func (s *Server) ReportDamageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}

		var req struct {
			LoanID       string  `json:"loan_id"`
			SchuelerID   string  `json:"schueler_id"`
			CopyID       string  `json:"copy_id"`
			Beschreibung string  `json:"beschreibung"`
			Betrag       float64 `json:"betrag"`
		}
		if !DecodeJSON(w, r, &req) {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		// 1. Mark copy as decommissioned
		_, err = tx.Exec(ctx, `
			UPDATE buecher_exemplare
			SET ist_ausgesondert = true, ist_ausleihbar = false, zustand_notiz = $1, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $2
		`, req.Beschreibung, req.CopyID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 2. Create Schadensfall
		var schadensID string
		err = tx.QueryRow(ctx, `
			INSERT INTO schadensfaelle (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, req.CopyID, req.LoanID, req.SchuelerID, req.Beschreibung, req.Betrag).Scan(&schadensID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 3. End loan
		_, err = tx.Exec(ctx, `
			UPDATE ausleihen
			SET rueckgabe_am = CURRENT_TIMESTAMP, rueckgabe_bearbeiter_id = $1
			WHERE id = $2
		`, claims.UserID, req.LoanID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, map[string]string{"status": "ok", "schadens_id": schadensID})
	}
}
