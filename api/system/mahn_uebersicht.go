package system

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// mahnUebersicht holds key metrics for the library reporting dashboard.
//
// Bewusst OHNE personenbezogene Einzeldaten: Diese Zusammenfassung speist die
// Statistik-Seite (Analyse-Kontext). Klarnamen der (minderjährigen) Schüler samt
// entliehenem Titel gehören dort nicht hin — das wäre Zweckentfremdung und verletzt
// die Datenminimierung (Art. 5 Abs. 1 lit. c DSGVO); zudem sind Lesegewohnheiten
// besonders schützenswert. Die namentliche Bearbeitung überfälliger Ausleihen läuft
// operativ und mit eigener Zugriffskontrolle im Mahnwesen (/api/mahnwesen). Hier nur
// die aggregierte Gesamtzahl und eine anonyme Verteilung nach Überfälligkeitsdauer.
type mahnUebersicht struct {
	Ueberfaellig        int                      `json:"total_overdue"`
	MaxTageUeberfaellig int                      `json:"max_tage_overdue"` // längste Überfälligkeit in Tagen (anonym)
	Zeitspannen         []ueberfaelligZeitspanne `json:"overdue_buckets"`  // anonyme Verteilung nach Dauer
}

// ueberfaelligZeitspanne ist ein anonymer Zähler je Überfälligkeits-Zeitspanne.
type ueberfaelligZeitspanne struct {
	Bezeichnung string `json:"label"`
	Anzahl      int    `json:"count"`
}

// MahnUebersicht gibt aggregierte Daten für das Dashboard zurück (z.B. Mahnungen).
// Liefert ausschliesslich anonyme Aggregate — siehe DashboardSummary.
func (t *Tueren) MahnUebersicht() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		k, err := repository.LadeMahnKennzahlen(ctx, t.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("fehler beim Laden der Mahnkennzahlen"))
			return
		}

		uebersicht := mahnUebersicht{
			Ueberfaellig:        k.Ueberfaellig,
			MaxTageUeberfaellig: k.MaxTage,
			Zeitspannen: []ueberfaelligZeitspanne{
				{Bezeichnung: "1–14 Tage", Anzahl: k.Bis14},
				{Bezeichnung: "15–30 Tage", Anzahl: k.Bis30},
				{Bezeichnung: "31–60 Tage", Anzahl: k.Bis60},
				{Bezeichnung: "über 60 Tage", Anzahl: k.Ueber60},
			},
		}

		tuer.AntworteJSON(w, http.StatusOK, uebersicht)
	}
}
