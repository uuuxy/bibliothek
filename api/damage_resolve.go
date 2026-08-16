package api

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// Die Gebühren-Erledigung: Bis 16.08.2026 konnte die Anwendung Schadensfälle nur
// ANLEGEN — weder Barzahlung noch Storno hatten eine Tür. Jede reale Gebühr blieb
// damit ewig offen und blockierte Schülerlöschung und LUSD-Abgleich dauerhaft
// (pruefeSchuelerLoeschbar bzw. der Abgleich prüfen ist_bezahlt = false).
//
// Rechte: edit_students, symmetrisch zum Anlegen (POST /api/damage/report). Bewusst
// NICHT "nur eingeloggt": Kiosk-Arbeitsplätze sind dauerhaft mit der Helfer-Rolle
// angemeldet — Gebühren erledigen dürfen nur Rollen, die Schülerdaten bearbeiten.

// ListStudentSchadensfaelleHandler liefert alle Schadensfälle eines Schülers für
// die Gebühren-Sektion der Schülerakte (GET /api/schueler/{id}/schadensfaelle).
func (s *Server) ListStudentSchadensfaelleHandler(damageRepo repository.DamageRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("Schüler-ID fehlt", nil)
		}
		faelle, err := damageRepo.ListSchadensfaelleVonSchueler(r.Context(), id)
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Schadensfälle", err)
		}
		RespondJSON(w, http.StatusOK, map[string]any{"data": faelle})
		return nil
	})
}

// erledigungsFehler mappt die Sentinel-Fehler der Repo-Schicht auf HTTP-Codes.
// 409 statt 500 ist hier tragend: Am Mehrplatz-System ist "eine Kollegin war
// schneller" der Normalfall — der Sanitizer würde einen 500 zu "interner
// Datenbankfehler" verschlucken und die eigentliche Ursache verstecken.
func erledigungsFehler(err error, aktion string) error {
	if errors.Is(err, repository.ErrSchadensfallNichtGefunden) {
		return apierrors.NotFound(err.Error(), err)
	}
	if errors.Is(err, repository.ErrSchadensfallErledigt) {
		return apierrors.Conflict(err.Error(), err)
	}
	return apierrors.Internal("Fehler beim "+aktion+" der Gebühr", err)
}

// BezahltGebuehrHandler verbucht die Zahlung (POST /api/schadensfaelle/{id}/bezahlt).
// Kein Request-Body: Der Betrag kommt IMMER aus der Datenbank, nie vom Client.
func (s *Server) BezahltGebuehrHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			return apierrors.Unauthorized("missing session information", nil)
		}
		if err := auditRepo.BezahltGebuehr(r.Context(), r.PathValue("id"), claims.UserID); err != nil {
			return erledigungsFehler(err, "Verbuchen")
		}
		RespondSuccess(w)
		return nil
	})
}

// StornoGebuehrHandler storniert eine Gebühr mit zwingender Begründung
// (POST /api/schadensfaelle/{id}/storno).
func (s *Server) StornoGebuehrHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			return apierrors.Unauthorized("missing session information", nil)
		}
		var req struct {
			Grund string `json:"grund"`
		}
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		grund := strings.TrimSpace(req.Grund)
		if grund == "" {
			return apierrors.BadRequest("Ein Stornierungsgrund ist erforderlich", nil)
		}
		if err := auditRepo.StornierungGebuehr(r.Context(), r.PathValue("id"), claims.UserID, grund); err != nil {
			return erledigungsFehler(err, "Stornieren")
		}
		RespondSuccess(w)
		return nil
	})
}
