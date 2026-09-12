package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/repository"
)

// Der Topf-Filter des Berichts (#596, Bauplan 7.3 Schritt 4).
//
// Die Lieferantenabrechnung ist das Blatt, gegen das die Rechnung des Händlers geprüft
// wird. Kommen zwei Rechnungen — eine für die Lernmittel, eine für die Schülerbücherei —,
// muss sich das Blatt auf den Topf einengen lassen, sonst prüft das Sekretariat jede
// Rechnung gegen eine Liste, in der auch der andere Topf steht.
//
// Am echten Postgres, weil die Einengung im SQL steht.
func TestBestellberichtFiltertNachTopf(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
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

	anlegen(t, "Land", repository.MittelLand, 120.00)
	anlegen(t, "Kreis", repository.MittelSchultraeger, 45.50)
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
		{"nur Lernmittelfreiheit", repository.MittelLand, 1, 120.00, repository.MittelLand},
		{"nur Schülerbücherei", repository.MittelSchultraeger, 1, 45.50, repository.MittelSchultraeger},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			orders, _, err := srv.ladeBestellungen(ctx, von, bis, "", f.mittel)
			if err != nil {
				t.Fatalf("Bestellungen lesen: %v", err)
			}
			if len(orders) != f.anzahl {
				t.Fatalf("%d Bestellungen, erwartet %d", len(orders), f.anzahl)
			}
			betrag, _ := summiereBestellungen(orders)
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

	// Die Alt-Bestellung ohne Zuordnung gehört in KEINEN der beiden gefilterten Berichte —
	// sie wird nie geraten (Migration 109, Backfill).
	fuerLand, _, err := srv.ladeBestellungen(ctx, von, bis, "", repository.MittelLand)
	if err != nil {
		t.Fatal(err)
	}
	fuerKreis, _, err := srv.ladeBestellungen(ctx, von, bis, "", repository.MittelSchultraeger)
	if err != nil {
		t.Fatal(err)
	}
	if len(fuerLand)+len(fuerKreis) != 2 {
		t.Errorf("die Alt-Bestellung ohne Zuordnung ist einem Topf zugeschlagen worden (%d + %d von 3)",
			len(fuerLand), len(fuerKreis))
	}
}
