package system

import (
	"fmt"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// mahnUebersicht sind die Kennzahlen der überfälligen Ausleihen für die Statistik-Seite.
//
// Bewusst ohne personenbezogene Einzeldaten: Diese Zusammenfassung speist die
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

// MahnUebersicht liefert die Kennzahlen der überfälligen Ausleihen, nur zusammengefasst und
// ohne Namen (siehe mahnUebersicht).
func (t *Tueren) MahnUebersicht() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Derselbe Bestandsfilter wie bei /api/statistiken: Die Kachel steht neben gefilterten
		// Zahlen und teilt durch eine davon.
		typFilter, _ := repository.BestandsFilterBedingung(r.URL.Query().Get("type"))
		k, err := repository.LadeMahnKennzahlen(ctx, t.DB.Pool, typFilter)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("fehler beim Laden der Mahnkennzahlen: %w", err))
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
