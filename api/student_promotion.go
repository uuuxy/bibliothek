package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"bibliothek/apierrors"
)

// PromoteStudentsResponse liefert die Statistik des Schuljahreswechsels zurück.
type PromoteStudentsResponse struct {
	VersetzteSchueler int `json:"versetzte_schueler"`
	NeueAbgaenger     int `json:"neue_abgaenger"`
}

// PromoteStudentsHandler führt den automatischen Schuljahreswechsel durch.
// @Summary      Automatische Versetzung (Schuljahreswechsel)
// @Description  Erhöht die Klassenstufe aller aktiven Schüler um 1. Markiert Abschlussklassen (9H, 10R, 13) automatisch als Abgänger.
// @Tags         schueler
// @Accept       json
// @Produce      json
// @Success      200  {object}  PromoteStudentsResponse
// @Failure      500  {object}  map[string]string
// @Router       /students/promote [post]
func (s *Server) PromoteStudentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		query := `
			WITH parsed AS (
				SELECT id,
					   klasse,
					   (substring(klasse from '^\d+')::int + 1) AS new_grade,
					   substring(klasse from '^\d+(.*)$') AS new_suffix
				FROM schueler
				WHERE ist_abgaenger = false 
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

		var versetzt, abgaenger int
		err = tx.QueryRow(ctx, query).Scan(&versetzt, &abgaenger)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(PromoteStudentsResponse{
			VersetzteSchueler: versetzt,
			NeueAbgaenger:     abgaenger,
		})
	}
}
