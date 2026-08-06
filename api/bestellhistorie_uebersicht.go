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
		//
		// „Wartet auf Bestätigung" hängt am TOKEN der Bestellung, nicht am heutigen Haken
		// des Lieferanten. Der Token entsteht beim Bestellen genau dann, wenn der Lieferant
		// den Bestelllink trägt (insertBestellverlauf), und bleibt danach unverändert —
		// die Bestellung IST mit Link rausgegangen, daran ändert sich später nichts.
		//
		// Über den Haken gezählt, verschwänden diese Bestellungen still aus der Zahl,
		// sobald der Bestelllink an einen anderen Händler wandert: Genau eine Zeile hält
		// ihn (idx_lieferanten_ein_bestelllink), also verliert ihn der bisherige dabei —
		// samt seiner offenen Bestellungen, auf deren Bestätigung weiterhin gewartet wird.
		// Die Listenansicht rechnet aus demselben Grund schon länger über den Token.
		err := s.DB.Pool.QueryRow(r.Context(), `
			SELECT count(*), coalesce(sum(b.gesamtbetrag), 0), coalesce(sum(b.anzahl_exemplare), 0),
			       count(*) FILTER (
			           WHERE b.bestaetigt_am IS NULL AND b.bestaetigungs_token_hash IS NOT NULL
			       )
			FROM bestellungen_verlauf b
		`).Scan(&u.Gesamt, &u.Gesamtbetrag, &u.GesamtExemplare, &u.OffeneBestaetigungen)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, u)
	}
}
