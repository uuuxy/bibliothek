package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
)

// PromoteStudentsResponse liefert die Statistik des Schuljahreswechsels zurück.
type PromoteStudentsResponse struct {
	PromotedCount int `json:"promoted_count"`
	ArchivedCount int `json:"archived_count"`
}

// promoteStudentsRequest verlangt eine explizite Bestätigung im Body. Das ist eine
// zusätzliche serverseitige Sicherung gegen versehentliche oder automatisierte
// Aufrufe (z. B. ein wiederholter Retry) — bei einem irreversiblen Batch-Vorgang
// dieser Tragweite reicht eine reine Client-seitige Bestätigung nicht aus.
type promoteStudentsRequest struct {
	Confirm bool `json:"confirm"`
}

// PromoteStudentsHandler führt den automatischen Schuljahreswechsel durch.
// @Summary      Automatische Versetzung (Schuljahreswechsel)
// @Description  Erhöht die Klassenstufe aller aktiven Schüler um 1. Markiert Abschlussklassen (9H, 10R, 13) automatisch als Abgänger. Erfordert { "confirm": true } im Body.
// @Tags         schueler
// @Accept       json
// @Produce      json
// @Param        body body promoteStudentsRequest true "Bestätigung"
// @Success      200  {object}  PromoteStudentsResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /students/promote [post]
func (s *Server) PromoteStudentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		claims, ok := auth.GetClaims(ctx)
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}

		var req promoteStudentsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültiger oder fehlender Request-Body"))
			return
		}
		if !req.Confirm {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New(`Bestätigung erforderlich: { "confirm": true } im Body senden`))
			return
		}

		// Der gesamte Hochzähl-Vorgang läuft in einer einzigen strikten Transaktion:
		// entweder wird JEDER Schüler versetzt/archiviert, oder — bei jedem Fehler —
		// keiner. db.SafeRollback greift auf jedem Fehler- UND Panic-Pfad; nur der
		// explizite Commit ganz unten übernimmt final.
		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer db.SafeRollback(ctx, tx)

		query := `
			WITH parsed AS (
				SELECT id,
					   klasse,
					   (substring(klasse from '^\d+')::int + 1) AS new_grade,
					   substring(klasse from '^\d+(.*)$') AS new_suffix
				FROM schueler
				WHERE ist_abgaenger = false
				  AND deleted_at IS NULL
				  AND klasse ~ '^\d+'
			),
			calculated AS (
				SELECT id,
					   (new_grade::text || new_suffix) AS new_klasse,
					   CASE 
						 WHEN new_grade = 10 AND new_suffix ILIKE '%h%' THEN true
						 WHEN new_grade = 11 AND new_suffix ILIKE '%r%' THEN true
						 WHEN new_grade >= 14 THEN true
						 ELSE false
					   END AS is_graduating
				FROM parsed
			),
			updated AS (
				UPDATE schueler s
				SET 
					klasse = CASE WHEN c.is_graduating THEN NULL ELSE c.new_klasse END,
					ist_abgaenger = c.is_graduating,
					abgaenger_jahr = CASE 
						WHEN c.is_graduating THEN EXTRACT(YEAR FROM CURRENT_DATE) 
						ELSE s.abgaenger_jahr 
					END,
					aktualisiert_am = CURRENT_TIMESTAMP
				FROM calculated c
				WHERE s.id = c.id
				RETURNING c.is_graduating
			)
			SELECT 
				COUNT(*) FILTER (WHERE is_graduating = false) AS versetzt,
				COUNT(*) FILTER (WHERE is_graduating = true) AS abgaenger
			FROM updated;
		`

		var promoted, archived int
		if err := tx.QueryRow(ctx, query).Scan(&promoted, &archived); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		log.Printf("Schuljahreswechsel durchgeführt (Benutzer %s): %d Schüler versetzt, %d neue Abgänger", claims.UserID, promoted, archived)

		RespondJSON(w, http.StatusOK, PromoteStudentsResponse{
			PromotedCount: promoted,
			ArchivedCount: archived,
		})
	}
}
