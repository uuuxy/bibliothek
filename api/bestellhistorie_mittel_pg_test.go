package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/repository"
)

// Historie und Kennzahlen nach Topf (#596, Bauplan 7.3 Schritt 4).
//
// Die Bestellhistorie ist der Ort, an dem die Bibliothekskraft eine Bestellung
// wiederfindet — und der einzige, an dem sich der Topf einer Alt-Bestellung nachtragen
// lässt (Rückweg, 7.3 Nr. 6). Ohne Filter muss sie dafür durch 200 Zeilen blättern.
//
// Die Kennzahlen im Kopf zählen ALLE Bestellungen, nicht nur die geladenen. Je Topf
// aufgeteilt müssen sie zusammen wieder die Gesamtzahl ergeben — sonst steht auf dem
// Bildschirm eine Aufteilung, die eine andere Summe hat als die Summe darüber.
func TestBestellhistorieNachTopf(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	if _, err := pool.Exec(ctx, `DELETE FROM bestellungen_verlauf`); err != nil {
		t.Fatalf("Verlauf leeren: %v", err)
	}
	anlegen := func(t *testing.T, name, mittel string, betrag float64, exemplare int) {
		t.Helper()
		var spalte any
		if mittel != "" {
			spalte = mittel
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO bestellungen_verlauf
			    (lieferant_name, lieferant_email, bestelldatum, gesamtbetrag, anzahl_exemplare, mittel)
			VALUES ($1, 'haendler@example.invalid', now(), $2, $3, $4)
		`, name+"-"+suffix, betrag, exemplare, spalte); err != nil {
			t.Fatalf("Bestellung %s anlegen: %v", name, err)
		}
	}
	anlegen(t, "Land1", repository.MittelLand, 120.00, 30)
	anlegen(t, "Land2", repository.MittelLand, 80.00, 20)
	anlegen(t, "Kreis", repository.MittelSchultraeger, 45.50, 5)
	anlegen(t, "Alt", "", 10.00, 2)

	historie := func(t *testing.T, filter string) ([]BestellVerlaufResponse, int) {
		t.Helper()
		ziel := "/api/bestellhistorie"
		if filter != "" {
			ziel += "?mittel=" + filter
		}
		rec := httptest.NewRecorder()
		srv.GetBestellhistorieHandler()(rec, httptest.NewRequest(http.MethodGet, ziel, nil))
		if rec.Code != http.StatusOK {
			return nil, rec.Code
		}
		var liste []BestellVerlaufResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &liste); err != nil {
			t.Fatalf("Antwort lesen: %v (%s)", err, rec.Body.String())
		}
		return liste, rec.Code
	}

	t.Run("Filter je Topf", func(t *testing.T) {
		faelle := []struct {
			filter string
			anzahl int
		}{
			{"", 4},
			{repository.MittelLand, 2},
			{repository.MittelSchultraeger, 1},
			// „ohne": die Alt-Bestellungen, deren Topf beim Backfill nicht eindeutig war.
			// Genau sie sucht, wer den Rückweg benutzt.
			{"ohne", 1},
		}
		for _, f := range faelle {
			liste, code := historie(t, f.filter)
			if code != http.StatusOK {
				t.Fatalf("Filter %q: Status %d", f.filter, code)
			}
			if len(liste) != f.anzahl {
				t.Errorf("Filter %q: %d Bestellungen, erwartet %d", f.filter, len(liste), f.anzahl)
			}
			for _, b := range liste {
				if f.filter == "ohne" && b.Mittel != "" {
					t.Errorf("Filter „ohne\" liefert eine Bestellung aus dem Topf %q", b.Mittel)
				}
				if f.filter != "" && f.filter != "ohne" && b.Mittel != f.filter {
					t.Errorf("Filter %q liefert eine Bestellung aus dem Topf %q", f.filter, b.Mittel)
				}
			}
		}
	})

	t.Run("unbekannter Topf ist 400 und keine stille Gesamtliste", func(t *testing.T) {
		if _, code := historie(t, "kreis"); code != http.StatusBadRequest {
			t.Errorf("Status %d, erwartet 400", code)
		}
	})

	t.Run("Kennzahlen je Topf ergeben zusammen die Gesamtzahl", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.GetBestellhistorieUebersichtHandler()(rec,
			httptest.NewRequest(http.MethodGet, "/api/bestellhistorie/uebersicht", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
		}
		var u BestellhistorieUebersicht
		if err := json.Unmarshal(rec.Body.Bytes(), &u); err != nil {
			t.Fatalf("Antwort lesen: %v", err)
		}

		if len(u.NachMittel) != 3 {
			t.Fatalf("%d Töpfe in den Kennzahlen, erwartet 3 (Land, Schulträger, ohne Zuordnung): %+v",
				len(u.NachMittel), u.NachMittel)
		}
		// Die Reihenfolge ist dieselbe wie im Warenkorb und im Bericht.
		if u.NachMittel[0].Mittel != repository.MittelLand ||
			u.NachMittel[1].Mittel != repository.MittelSchultraeger ||
			u.NachMittel[2].Mittel != "" {
			t.Errorf("Reihenfolge der Töpfe: %+v", u.NachMittel)
		}

		var summeBetrag float64
		var summeAnzahl, summeExemplare int
		for _, m := range u.NachMittel {
			summeBetrag += m.Gesamtbetrag
			summeAnzahl += m.Gesamt
			summeExemplare += m.GesamtExemplare
		}
		if summeAnzahl != u.Gesamt || summeExemplare != u.GesamtExemplare || summeBetrag != u.Gesamtbetrag {
			t.Errorf("die Aufteilung ergibt %d Bestellungen / %d Exemplare / %.2f €, die Gesamtzahlen sagen %d / %d / %.2f",
				summeAnzahl, summeExemplare, summeBetrag, u.Gesamt, u.GesamtExemplare, u.Gesamtbetrag)
		}
		if u.NachMittel[0].Gesamtbetrag != 200.00 {
			t.Errorf("Lernmittelfreiheit: %.2f €, erwartet 200,00 €", u.NachMittel[0].Gesamtbetrag)
		}
	})
}
