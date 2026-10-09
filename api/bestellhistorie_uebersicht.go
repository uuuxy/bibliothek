package api

import (
	"context"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
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
		k, err := repository.LadeBestellKennzahlen(r.Context(), s.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		u := BestellhistorieUebersicht{
			Gesamt: k.Gesamt, Gesamtbetrag: k.Gesamtbetrag, GesamtExemplare: k.GesamtExemplare,
			OffeneBestaetigungen: k.OffeneBestaetigungen,
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
	gezaehlt, err := repository.LadeBestellKennzahlenJeTopf(ctx, s.DB.Pool)
	if err != nil {
		return nil, err
	}

	aus := make([]BestellhistorieTopf, 0, len(mitteltopf.Reihenfolge()))
	for _, topf := range mitteltopf.Reihenfolge() {
		if t, ok := gezaehlt[topf]; ok {
			aus = append(aus, BestellhistorieTopf(t))
			continue
		}
		aus = append(aus, BestellhistorieTopf{Mittel: topf})
	}
	return aus, nil
}
