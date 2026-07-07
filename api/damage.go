package api

import (
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
			LoanID       string  `json:"loan_id"`
			SchuelerID   string  `json:"schueler_id"`
			CopyID       string  `json:"copy_id"`
			Beschreibung string  `json:"beschreibung"`
			Betrag       float64 `json:"betrag"`
		}
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}

		schadensID, err := damageRepo.ReportDamage(r.Context(), req.CopyID, req.LoanID, req.SchuelerID, claims.UserID, req.Beschreibung, req.Betrag)
		if err != nil {
			return apierrors.Internal("Fehler beim Melden des Schadens", err)
		}

		RespondJSON(w, http.StatusOK, map[string]string{"status": "ok", "schadens_id": schadensID})
		return nil
	})
}
