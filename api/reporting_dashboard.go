package api

import (
	"bibliothek/apierrors"
	"errors"

	"net/http"
)

// DashboardSummary holds key metrics for the library reporting dashboard.
//
// Bewusst OHNE personenbezogene Einzeldaten: Diese Zusammenfassung speist die
// Statistik-Seite (Analyse-Kontext). Klarnamen der (minderjährigen) Schüler samt
// entliehenem Titel gehören dort nicht hin — das wäre Zweckentfremdung und verletzt
// die Datenminimierung (Art. 5 Abs. 1 lit. c DSGVO); zudem sind Lesegewohnheiten
// besonders schützenswert. Die namentliche Bearbeitung überfälliger Ausleihen läuft
// operativ und mit eigener Zugriffskontrolle im Mahnwesen (/api/mahnwesen). Hier nur
// die aggregierte Gesamtzahl und eine anonyme Verteilung nach Überfälligkeitsdauer.
type DashboardSummary struct {
	TotalOverdue   int             `json:"total_overdue"`
	MaxTageOverdue int             `json:"max_tage_overdue"` // längste Überfälligkeit in Tagen (anonym)
	OverdueBuckets []OverdueBucket `json:"overdue_buckets"`  // anonyme Verteilung nach Dauer
}

// OverdueBucket ist ein anonymer Zähler je Überfälligkeits-Zeitspanne.
type OverdueBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// GetDashboardSummaryHandler gibt aggregierte Daten für das Dashboard zurück (z.B. Mahnungen).
// Liefert ausschliesslich anonyme Aggregate — siehe DashboardSummary.
func (s *Server) GetDashboardSummaryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var summary DashboardSummary

		// Gesamtzahl, längste Dauer und die Dauer-Verteilung in EINEM aggregierten
		// Scan über die offenen, überfälligen Ausleihen. Keine JOINs auf schueler/titel:
		// es verlässt bewusst kein personenbezogenes Feld die Datenbank.
		var b1, b2, b3, b4 int
		err := s.DB.Pool.QueryRow(ctx, `
			WITH offen AS (
				SELECT (CURRENT_TIMESTAMP - rueckgabe_frist) AS verzug
				FROM ausleihen
				WHERE rueckgabe_am IS NULL AND rueckgabe_frist < CURRENT_TIMESTAMP
			)
			SELECT
				COUNT(*)::int,
				COALESCE(MAX(GREATEST(0, EXTRACT(DAY FROM verzug)::int)), 0)::int,
				COUNT(*) FILTER (WHERE verzug <= INTERVAL '14 days')::int,
				COUNT(*) FILTER (WHERE verzug > INTERVAL '14 days' AND verzug <= INTERVAL '30 days')::int,
				COUNT(*) FILTER (WHERE verzug > INTERVAL '30 days' AND verzug <= INTERVAL '60 days')::int,
				COUNT(*) FILTER (WHERE verzug > INTERVAL '60 days')::int
			FROM offen
		`).Scan(&summary.TotalOverdue, &summary.MaxTageOverdue, &b1, &b2, &b3, &b4)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("fehler beim Laden der Mahnkennzahlen"))
			return
		}

		summary.OverdueBuckets = []OverdueBucket{
			{Label: "1–14 Tage", Count: b1},
			{Label: "15–30 Tage", Count: b2},
			{Label: "31–60 Tage", Count: b3},
			{Label: "über 60 Tage", Count: b4},
		}

		RespondJSON(w, http.StatusOK, summary)
	}
}
