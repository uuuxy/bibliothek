package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// ReportDamageHandler handles POST /api/damage/report
// Sets ist_ausgesondert = true, inserts into schadensfaelle, and ends the loan.
func (s *Server) ReportDamageHandler(damageRepo repository.DamageRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			return apierrors.Unauthorized("missing session information", nil)
		}

		var req struct {
			LoanID       string `json:"loan_id" validate:"omitempty,uuid_oder_leer"`
			SchuelerID   string `json:"schueler_id" validate:"omitempty,uuid_oder_leer"`
			CopyID       string `json:"copy_id" validate:"omitempty,uuid_oder_leer"`
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
