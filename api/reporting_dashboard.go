package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"
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

		k, err := repository.LadeMahnKennzahlen(ctx, s.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("fehler beim Laden der Mahnkennzahlen"))
			return
		}

		summary := DashboardSummary{
			TotalOverdue:   k.Ueberfaellig,
			MaxTageOverdue: k.MaxTage,
			OverdueBuckets: []OverdueBucket{
				{Label: "1–14 Tage", Count: k.Bis14},
				{Label: "15–30 Tage", Count: k.Bis30},
				{Label: "31–60 Tage", Count: k.Bis60},
				{Label: "über 60 Tage", Count: k.Ueber60},
			},
		}

		RespondJSON(w, http.StatusOK, summary)
	}
}
