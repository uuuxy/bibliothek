package buchetikett

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Ein ausgesondertes Exemplar bekommt kein Etikett mehr. Wer die Etiketten eines Titels
// druckt, klebt sie auf Bücher im Regal; auf einem Bogen mit fortlaufenden Plätzen verschöbe
// ein überzähliges Etikett alle folgenden. Die Gegenprobe: Die übrigen Exemplare bleiben.
func TestEtiketten_OhneAusgesonderteExemplare(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	titel := testdaten.LegeTitelAn(t, pool, "Etikett-Aussonderung", 0)
	for _, b := range []struct {
		barcode      string
		ausgesondert bool
	}{
		{"B-ETI-1", false},
		{"B-ETI-2", true},
		{"B-ETI-3", false},
	} {
		// Der Grund ist Pflicht, sobald ausgesondert wird (chk_aussonderung_grund).
		if _, err := pool.Exec(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund)
			VALUES ($1, $2, NOT $3, $3, CASE WHEN $3 THEN 'AUSSORTIERT' ELSE NULL END)`,
			titel, b.barcode, b.ausgesondert); err != nil {
			t.Fatalf("Exemplar %s anlegen: %v", b.barcode, err)
		}
	}

	items, err := LadeDesTitels(ctx, pool, titel)
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
