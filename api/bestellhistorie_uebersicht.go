package api

import (
	"context"
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
	// NachMittel teilt dieselben Zahlen auf die Töpfe auf (Migration 109): Lernmittel
	// aus Landesmitteln, Schülerbücherei aus Mitteln des Schulträgers, dazu die
	// Alt-Bestellungen ohne eindeutige Zuordnung. Immer alle drei Einträge, auch mit
	// Null — eine fehlende Zeile läse sich wie „nichts bestellt", und die Aufteilung
	// muss zusammen wieder die Gesamtzahl darüber ergeben.
	NachMittel []BestellhistorieTopf `json:"nach_mittel"`
}

// BestellhistorieTopf sind die Kennzahlen EINES Topfes.
type BestellhistorieTopf struct {
	// Mittel ist der Wert der Spalte; leer = ohne Zuordnung.
	Mittel          string  `json:"mittel"`
	Gesamt          int     `json:"gesamt"`
	Gesamtbetrag    float64 `json:"gesamtbetrag"`
	GesamtExemplare int     `json:"gesamt_exemplare"`
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

		u.NachMittel, err = s.kennzahlenJeTopf(r.Context())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, u)
	}
}

// kennzahlenJeTopf teilt die Kennzahlen auf die Töpfe auf — in der Reihenfolge, in der
// sie überall stehen (Warenkorb, Bericht, Chips): Lernmittel zuerst, dann die
// Schülerbücherei, zuletzt die Alt-Bestellungen ohne Zuordnung.
//
// Eine eigene Abfrage statt eines zweiten Aggregats in der ersten: Die Gesamtzahlen
// darüber sollen nicht davon abhängen, dass die Gruppierung gelingt.
func (s *Server) kennzahlenJeTopf(ctx context.Context) ([]BestellhistorieTopf, error) {
	rows, err := s.DB.Pool.Query(ctx, `
		SELECT coalesce(mittel, ''), count(*), coalesce(sum(gesamtbetrag), 0),
		       coalesce(sum(anzahl_exemplare), 0)
		FROM bestellungen_verlauf
		GROUP BY coalesce(mittel, '')
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	gezaehlt := map[string]BestellhistorieTopf{}
	for rows.Next() {
		var t BestellhistorieTopf
		if err := rows.Scan(&t.Mittel, &t.Gesamt, &t.Gesamtbetrag, &t.GesamtExemplare); err != nil {
			return nil, err
		}
		gezaehlt[t.Mittel] = t
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	aus := make([]BestellhistorieTopf, 0, len(mittelReihenfolge))
	for _, topf := range mittelReihenfolge {
		if t, ok := gezaehlt[topf]; ok {
			aus = append(aus, t)
			continue
		}
		aus = append(aus, BestellhistorieTopf{Mittel: topf})
	}
	return aus, nil
}
