package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

// Der Topf-Filter des Bestellberichts.
//
// Die Lieferantenabrechnung ist das Blatt, gegen das die Rechnung des Händlers geprüft wird.
// Kommen zwei Rechnungen, eine für die Lernmittel und eine für die Schülerbücherei, muss sich
// das Blatt auf den Topf einengen lassen; sonst prüft das Sekretariat jede Rechnung gegen eine
// Liste, in der auch der andere Topf steht. An der Datenbank, weil die Einengung im SQL steht.
func TestBestellberichtFiltertNachTopf(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	anlegen := func(t *testing.T, name, mittel string, betrag float64) {
		t.Helper()
		var spalte any
		if mittel != "" {
			spalte = mittel
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO bestellungen_verlauf
			    (lieferant_name, lieferant_email, bestelldatum, gesamtbetrag, anzahl_exemplare, mittel)
			VALUES ($1, 'haendler@example.invalid', $2, $3, 1, $4)
		`, name+"-"+suffix, time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC), betrag, spalte); err != nil {
			t.Fatalf("Bestellung %s anlegen: %v", name, err)
		}
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM bestellungen_verlauf WHERE lieferant_name LIKE '%' || $1`, suffix); err != nil {
			t.Errorf("Aufräumen: %v", err)
		}
	})

	anlegen(t, "Land", mitteltopf.Land, 120.00)
	anlegen(t, "Kreis", mitteltopf.Schultraeger, 45.50)
	anlegen(t, "Alt", "", 10.00)

	von := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	bis := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	faelle := []struct {
		name         string
		mittel       string
		anzahl       int
		betrag       float64
		erwartetTopf string
	}{
		{"ohne Filter alle drei", "", 3, 175.50, ""},
		{"nur Lernmittelfreiheit", mitteltopf.Land, 1, 120.00, mitteltopf.Land},
		{"nur Schülerbücherei", mitteltopf.Schultraeger, 1, 45.50, mitteltopf.Schultraeger},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			orders, _, err := repository.LadeBerichtBestellungen(ctx, pool, von, bis, "", f.mittel)
			if err != nil {
				t.Fatalf("Bestellungen lesen: %v", err)
			}
			if len(orders) != f.anzahl {
				t.Fatalf("%d Bestellungen, erwartet %d", len(orders), f.anzahl)
			}
			var betrag float64
			for _, o := range orders {
				betrag += o.Gesamtbetrag
			}
			if betrag != f.betrag {
				t.Errorf("Summe %.2f, erwartet %.2f", betrag, f.betrag)
			}
			if f.mittel == "" {
				return
			}
			for _, o := range orders {
				if o.Mittel != f.erwartetTopf {
					t.Errorf("der Bericht führt eine Bestellung aus dem Topf %q", o.Mittel)
				}
			}
		})
	}

	// Die Bestellung ohne Zuordnung gehört in keinen der beiden gefilterten Berichte: Ihr Topf
	// wird nicht geraten.
	fuerLand, _, err := repository.LadeBerichtBestellungen(ctx, pool, von, bis, "", mitteltopf.Land)
	if err != nil {
		t.Fatal(err)
	}
	fuerKreis, _, err := repository.LadeBerichtBestellungen(ctx, pool, von, bis, "", mitteltopf.Schultraeger)
	if err != nil {
		t.Fatal(err)
	}
	if len(fuerLand)+len(fuerKreis) != 2 {
		t.Errorf("die Bestellung ohne Zuordnung ist einem Topf zugeschlagen worden (%d + %d von 3)",
			len(fuerLand), len(fuerKreis))
	}
}
