package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// DashboardSummary holds key metrics for the library reporting dashboard.
type DashboardSummary struct {
	TotalOverdue int              `json:"total_overdue"`
	TopOverdue   []OverdueSummary `json:"top_overdue"`
}

// OverdueSummary groups overdue loans by delay categories.
type OverdueSummary struct {
	SchuelerName string `json:"schueler_name"`
	Klasse       string `json:"klasse"`
	Titel        string `json:"titel"`
	Tage         int    `json:"tage"`
}

// GetDashboardSummaryHandler gibt aggregierte Daten für das Dashboard zurück (z.B. Mahnungen).
func (s *Server) GetDashboardSummaryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var summary DashboardSummary

		// 1. Gesamtanzahl aktuell überfälliger Ausleihen ermitteln
		err := s.DB.Pool.QueryRow(ctx, `
			SELECT count(*) FROM ausleihen 
			WHERE rueckgabe_am IS NULL AND rueckgabe_frist < CURRENT_TIMESTAMP
		`).Scan(&summary.TotalOverdue)
		if err != nil {
			http.Error(w, "Fehler beim Laden der Gesamtanzahl", http.StatusInternalServerError)
			return
		}

		// 2. Top 5 der am längsten überfälligen Bücher
		rows, err := s.DB.Pool.Query(ctx, `
			SELECT s.vorname || ' ' || s.nachname AS schueler_name, 
			       s.klasse, 
			       t.titel, 
			       GREATEST(0, EXTRACT(DAY FROM (CURRENT_TIMESTAMP - a.rueckgabe_frist))::int) AS tage
			FROM ausleihen a
			JOIN buecher_exemplare e ON a.exemplar_id = e.id
			JOIN buecher_titel t ON e.titel_id = t.id
			JOIN schueler s ON a.schueler_id = s.id
			WHERE a.rueckgabe_am IS NULL AND a.rueckgabe_frist < CURRENT_TIMESTAMP
			ORDER BY a.rueckgabe_frist ASC
			LIMIT 5
		`)
		if err != nil {
			http.Error(w, "Fehler beim Laden der Top 5", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		summary.TopOverdue = make([]OverdueSummary, 0)
		for rows.Next() {
			var o OverdueSummary
			if err := rows.Scan(&o.SchuelerName, &o.Klasse, &o.Titel, &o.Tage); err == nil {
				summary.TopOverdue = append(summary.TopOverdue, o)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(summary)
	}
}
