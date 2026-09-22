package service

import (
	"context"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Der Zulauf am echten Postgres: Die drei Bestellspalten sind NULL bei Altbestand ohne
// bestellung_id — ein Scan in string statt *string bräche dort die Iteration ab, und der
// ganze Wareneingang wäre ein 500 (Bugklasse „NULL-Scan", die nur ein PG-Test sieht).
// Dazu die Gruppierung selbst: zwei Töpfe am selben Tag beim selben Händler sind zwei
// Gruppen, das Datum ist der Kalendertag der Schule.
func TestGetIncomingShipments_AmPostgres(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	var titelID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ('Zulauf-Probe', '978-9-99-200000-1')
		ON CONFLICT (isbn) DO UPDATE SET titel = EXCLUDED.titel RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		// Exemplare fallen per CASCADE mit dem Titel; die Bestellungen kennt ihre Adresse.
		for _, sql := range []string{
			`DELETE FROM buecher_titel WHERE id = '` + titelID + `'`,
			`DELETE FROM bestellungen_verlauf WHERE lieferant_email = 'zulauf-probe@example.org'`,
		} {
			if _, err := pool.Exec(context.Background(), sql); err != nil {
				t.Logf("Aufräumen: %v", err)
			}
		}
	})

	// Zwei Bestellungen am selben Abend (22:30 UTC = 00:30 Schulzeit des Folgetags).
	bestellungen := make([]string, 0, 2)
	for i := 0; i < 2; i++ {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, bestelldatum)
			VALUES ('Buchhandlung Probe', 'zulauf-probe@example.org', '2026-09-29 22:30:00+00') RETURNING id`).Scan(&id); err != nil {
			t.Fatalf("Bestellung anlegen: %v", err)
		}
		bestellungen = append(bestellungen, id)
	}
	exemplare := []struct {
		barcode, notiz string
		bestellung     *string
	}{
		{"ZUL-PROBE-1", "Im Zulauf - Buchhandlung Probe", &bestellungen[0]},
		{"ZUL-PROBE-2", "Im Zulauf - Buchhandlung Probe", &bestellungen[0]},
		{"ZUL-PROBE-3", "Bestellt (ohne Vorab-Barcode) - Buchhandlung Probe", &bestellungen[1]},
		{"ZUL-PROBE-4", "Im Zulauf - Altlieferant", nil}, // Altbestand vor Migration 063
	}
	for _, e := range exemplare {
		if _, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus, bestellung_id, zustand_notiz)
			VALUES ($1, $2, false, 'im_zulauf', $3, $4)`, titelID, e.barcode, e.bestellung, e.notiz); err != nil {
			t.Fatalf("Exemplar %s anlegen: %v", e.barcode, err)
		}
	}

	// Ein Exemplar im Zulauf ohne Notiz: zustand_notiz ist nullbar. Die Bestellung schreibt
	// sie immer, aber ein anderer Schreiber muss es nicht — bis zum 22.09.2026 machte
	// dieses eine Exemplar den ganzen Wareneingang zum 500 („cannot scan NULL into *string").
	if _, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus)
		VALUES ($1, 'ZUL-PROBE-5', false, 'bestellt')`, titelID); err != nil {
		t.Fatalf("Exemplar ohne Notiz anlegen: %v", err)
	}

	groups, err := GetIncomingShipments(ctx, pool)
	if err != nil {
		t.Fatalf("GetIncomingShipments: %v", err)
	}
	// Nur die Gruppen dieses Titels — andere Tests desselben Pakets dürfen eigene Zeilen halten.
	menge := map[string]int{}
	datum := map[string]string{}
	for _, g := range groups {
		for _, it := range g.Items {
			if it.TitelID == titelID {
				menge[g.SupplierName+"/"+g.ID] += it.Menge
				datum[g.SupplierName+"/"+g.ID] = g.Date
			}
		}
	}
	erwartet := map[string]int{
		"Buchhandlung Probe/" + bestellungen[0]: 2,
		"Buchhandlung Probe/" + bestellungen[1]: 1,
	}
	for schluessel, soll := range erwartet {
		if menge[schluessel] != soll {
			t.Errorf("%s: %d Exemplare, erwartet %d (Gruppen: %v)", schluessel, menge[schluessel], soll, menge)
		}
		if datum[schluessel] != "30.09.2026" {
			t.Errorf("%s: Datum %q, erwartet den Kalendertag der Schule 30.09.2026", schluessel, datum[schluessel])
		}
	}
	altGefunden := false
	for schluessel, n := range menge {
		if strings.HasPrefix(schluessel, "Altlieferant/") && n == 1 {
			altGefunden = true
		}
	}
	if !altGefunden {
		t.Errorf("Altbestand ohne Bestellung fehlt als eigene Gruppe „Altlieferant\": %v", menge)
	}
	ohneNotiz := false
	for schluessel, n := range menge {
		if strings.HasPrefix(schluessel, "Unbekannter Lieferant/") && n >= 1 {
			ohneNotiz = true
		}
	}
	if !ohneNotiz {
		t.Errorf("Exemplar ohne Notiz fehlt unter „Unbekannter Lieferant\": %v", menge)
	}
}
