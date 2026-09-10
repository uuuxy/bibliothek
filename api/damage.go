package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
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
func (s *Server) MarkCopyDefektHandler(damageRepo repository.DamageRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		copyID := r.PathValue("id")
		if copyID == "" {
			return apierrors.BadRequest("missing copy ID parameter", nil)
		}

		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			return apierrors.Unauthorized("missing session information", nil)
		}

		var req DefektRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		if req.Betrag < 0 {
			return apierrors.BadRequest("betrag darf nicht negativ sein", nil)
		}
		if req.Beschreibung == "" {
			req.Beschreibung = "Defekt/Schaden bei Rückgabe gemeldet"
		}

		schadensID, err := damageRepo.MarkCopyDefekt(r.Context(), copyID, req.LoanID, req.SchuelerID, claims.UserID, req.Betrag, req.Beschreibung)
		if err != nil {
			if err == pgx.ErrNoRows {
				return apierrors.NotFound("book copy not found", nil)
			}
			return apierrors.Internal("Fehler beim Markieren des Defekts", err)
		}

		RespondJSON(w, http.StatusOK, DefektResponse{Status: "ok", SchadensID: schadensID})
		return nil
	})
}

// ReportDamageHandler handles POST /api/damage/report
// Sets ist_ausgesondert = true, inserts into schadensfaelle, and ends the loan.
func (s *Server) ReportDamageHandler(damageRepo repository.DamageRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			return apierrors.Unauthorized("missing session information", nil)
		}

		var req struct {
			LoanID       string `json:"loan_id"`
			SchuelerID   string `json:"schueler_id"`
			CopyID       string `json:"copy_id"`
			Beschreibung string `json:"beschreibung"`
			// Art: Fallgruppe des Bescheids — Pflicht, ohne stillen Vorgabewert. Bis zum
			// 10.09.2026 fehlte das Feld, jede Forderung bekam den DEFAULT 'beschaedigt',
			// und der Bescheid nannte ein verlorenes Buch „beschädigt zurückgegeben".
			Art    string  `json:"art"`
			Betrag float64 `json:"betrag"`
		}
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		art := repository.SchadensArt(req.Art)
		if !art.Gueltig() {
			//nolint:staticcheck // ST1005: ganzer Satz — die Meldung steht so vor der Bibliothekskraft.
			return apierrors.BadRequest("Bitte angeben, ob das Buch nicht zurückgegeben oder beschädigt zurückgegeben wurde.",
				errors.New("art fehlt oder ist ungültig"))
		}

		schadensID, err := damageRepo.ReportDamage(r.Context(), req.CopyID, req.LoanID, req.SchuelerID, claims.UserID, req.Beschreibung, art, req.Betrag)
		if err != nil {
			// Zwischenzeitliche Neuausleihe ist ein Konflikt (409), kein Serverfehler:
			// Der Nutzer muss den Vorgang neu laden, nicht der Server ist kaputt.
			if errors.Is(err, repository.ErrExemplarNeuVerliehen) {
				return apierrors.Conflict(err.Error(), err)
			}
			return apierrors.Internal("Fehler beim Melden des Schadens", err)
		}

		RespondJSON(w, http.StatusOK, map[string]string{"status": "ok", "schadens_id": schadensID})
		return nil
	})
}
