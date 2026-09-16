package api

import (
	"context"
	"testing"

	"bibliothek/db"
)

// Ein ausgesondertes Exemplar bekommt kein Etikett mehr.
//
// Fund (OFFEN.md 5.5): Der Titel-Etikettendruck lud ALLE Exemplare eines Titels. Wer die
// Etiketten eines Titels druckt, klebt sie auf Bücher, die im Regal stehen — ein
// ausgesondertes gibt es dort nicht mehr. Sein Etikett ist ein Blatt Papier für ein Buch,
// das niemand findet, und auf einem Bogen mit fortlaufenden Plätzen verschiebt es alle
// folgenden: Das Etikett des nächsten Buches klebt dann auf dem falschen Platz.
//
// Die Gegenprobe steht daneben: Die übrigen Exemplare desselben Titels bleiben.
func TestEtiketten_OhneAusgesonderteExemplare(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	titel := titelMitMeldebestand(t, pool, "Etikett-Aussonderung", 0)
	for _, b := range []struct {
		barcode      string
		ausgesondert bool
	}{
		{"B-ETI-1", false},
		{"B-ETI-2", true},
		{"B-ETI-3", false},
	} {
		// Der Grund ist Pflicht, sobald ausgesondert wird (chk_aussonderung_grund) — die
		// Datenbank lässt eine Aussonderung ohne Begründung gar nicht erst zu.
		if _, err := pool.Exec(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund)
			VALUES ($1, $2, NOT $3, $3, CASE WHEN $3 THEN 'AUSSORTIERT' ELSE NULL END)`,
			titel, b.barcode, b.ausgesondert); err != nil {
			t.Fatalf("Exemplar %s anlegen: %v", b.barcode, err)
		}
	}

	items, err := srv.queryLabelItems(ctx, titel)
	if err != nil {
		t.Fatalf("Etiketten laden: %v", err)
	}

	var barcodes []string
	for _, i := range items {
		barcodes = append(barcodes, i.BarcodeID)
	}
	if len(items) != 2 {
		t.Fatalf("%d Etiketten (%v), erwartet 2 — das ausgesonderte Exemplar ist mitgedruckt "+
			"und verschiebt alle folgenden Plätze auf dem Bogen", len(items), barcodes)
	}
	for _, b := range barcodes {
		if b == "B-ETI-2" {
			t.Errorf("das ausgesonderte Exemplar %q bekommt ein Etikett", b)
		}
	}
	if barcodes[0] != "B-ETI-1" || barcodes[1] != "B-ETI-3" {
		t.Errorf("die übrigen Exemplare fehlen oder stehen falsch: %v", barcodes)
	}
}
