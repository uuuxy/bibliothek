package api

import (
	"net/http"

	"bibliothek/apierrors"
)

// BestellhistorieUebersicht sind die Kennzahlen über den GESAMTEN Bestellverlauf.
//
// Sie haben einen eigenen Endpunkt, weil die Liste gedeckelt ist (siehe
// bestellhistorieStandardLimit). Würde die Oberfläche ihre Summen aus den geladenen
// Zeilen rechnen, stünde nach dem Deckeln eine zu kleine Zahl im Kopf — und zwar eine,
// die aussieht wie eine Gesamtsumme. Eine falsche Zahl ist schlimmer als keine.
type BestellhistorieUebersicht struct {
	Gesamt               int     `json:"gesamt"`
	Gesamtbetrag         float64 `json:"gesamtbetrag"`
	GesamtExemplare      int     `json:"gesamt_exemplare"`
	OffeneBestaetigungen int     `json:"offene_bestaetigungen"`
}

// GetBestellhistorieUebersichtHandler liefert die Kennzahlen über alle Bestellungen.
func (s *Server) GetBestellhistorieUebersichtHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u BestellhistorieUebersicht
		// Eine Abfrage, vier Zahlen: Aggregate über den Bestellkopf sind billig, die teure
		// Seite waren die Positionen — und die braucht die Übersicht nicht.
		err := s.DB.Pool.QueryRow(r.Context(), `
			SELECT count(*), coalesce(sum(b.gesamtbetrag), 0), coalesce(sum(b.anzahl_exemplare), 0),
			       count(*) FILTER (
			           WHERE b.bestaetigt_am IS NULL AND coalesce(l.bietet_bestellbestaetigung, false)
			       )
			FROM bestellungen_verlauf b
			LEFT JOIN lieferanten l ON l.id = b.lieferant_id
		`).Scan(&u.Gesamt, &u.Gesamtbetrag, &u.GesamtExemplare, &u.OffeneBestaetigungen)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, u)
	}
}
