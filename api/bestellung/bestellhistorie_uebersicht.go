package bestellung

import (
	"context"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

// bestellhistorieUebersicht sind die Kennzahlen über den GESAMTEN Bestellverlauf.
//
// Sie haben einen eigenen Endpunkt, weil die Liste gedeckelt ist (siehe
// bestellhistorieVorgabeGrenze). Würde die Oberfläche ihre Summen aus den geladenen
// Zeilen rechnen, stünde nach dem Deckeln eine zu kleine Zahl im Kopf — und zwar eine,
// die aussieht wie eine Gesamtsumme. Eine falsche Zahl ist schlimmer als keine.
type bestellhistorieUebersicht struct {
	Gesamt               int     `json:"gesamt"`
	Gesamtbetrag         float64 `json:"gesamtbetrag"`
	GesamtExemplare      int     `json:"gesamt_exemplare"`
	OffeneBestaetigungen int     `json:"offene_bestaetigungen"`
	// NachMittel teilt dieselben Zahlen auf die Töpfe auf (Migration 109): Lernmittel
	// aus Landesmitteln, Schülerbücherei aus Mitteln des Schulträgers, dazu die
	// Alt-Bestellungen ohne eindeutige Zuordnung. Immer alle drei Einträge, auch mit
	// Null — eine fehlende Zeile läse sich wie „nichts bestellt", und die Aufteilung
	// muss zusammen wieder die Gesamtzahl darüber ergeben.
	NachMittel []bestellhistorieTopf `json:"nach_mittel"`
}

// bestellhistorieTopf sind die Kennzahlen EINES Topfes.
type bestellhistorieTopf struct {
	// Mittel ist der Wert der Spalte; leer = ohne Zuordnung.
	Mittel          string  `json:"mittel"`
	Gesamt          int     `json:"gesamt"`
	Gesamtbetrag    float64 `json:"gesamtbetrag"`
	GesamtExemplare int     `json:"gesamt_exemplare"`
}

// BestellhistorieUebersicht liefert die Kennzahlen über alle Bestellungen.
func (t *Tueren) BestellhistorieUebersicht() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		k, err := repository.LadeBestellKennzahlen(r.Context(), t.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		u := bestellhistorieUebersicht{
			Gesamt: k.Gesamt, Gesamtbetrag: k.Gesamtbetrag, GesamtExemplare: k.GesamtExemplare,
			OffeneBestaetigungen: k.OffeneBestaetigungen,
		}

		u.NachMittel, err = t.kennzahlenJeTopf(r.Context())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, u)
	}
}

// kennzahlenJeTopf teilt die Kennzahlen auf die Töpfe auf — in der Reihenfolge, in der
// sie überall stehen (Warenkorb, Bericht, Chips): Lernmittel zuerst, dann die
// Schülerbücherei, zuletzt die Alt-Bestellungen ohne Zuordnung.
//
// Eine eigene Abfrage statt eines zweiten Aggregats in der ersten: Die Gesamtzahlen
// darüber sollen nicht davon abhängen, dass die Gruppierung gelingt.
func (t *Tueren) kennzahlenJeTopf(ctx context.Context) ([]bestellhistorieTopf, error) {
	gezaehlt, err := repository.LadeBestellKennzahlenJeTopf(ctx, t.DB.Pool)
	if err != nil {
		return nil, err
	}

	aus := make([]bestellhistorieTopf, 0, len(mitteltopf.Reihenfolge()))
	for _, topf := range mitteltopf.Reihenfolge() {
		if zahlen, ok := gezaehlt[topf]; ok {
			aus = append(aus, bestellhistorieTopf(zahlen))
			continue
		}
		aus = append(aus, bestellhistorieTopf{Mittel: topf})
	}
	return aus, nil
}
