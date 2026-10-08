package service

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Der Sammelimport legt einen Titel ohne Lernmittel-Signatur ohne Jahrgang an: NULL in beiden
// Spalten (Migration 162), keine Vorgabe.
func TestImportDynamic_OhneJahrgangBleibtUnbekannt(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const barcode = "JAHRGANG-PROBE-162-1"

	t.Cleanup(func() {
		var titelID string
		if err := pool.QueryRow(ctx, `SELECT coalesce((SELECT titel_id::text FROM buecher_exemplare
			WHERE barcode_id = $1), '')`, barcode).Scan(&titelID); err != nil {
			t.Errorf("aufräumen: Probe-Exemplar lesen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE barcode_id = $1`, barcode); err != nil {
			t.Errorf("aufräumen: Probe-Exemplar löschen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE id::text = $1`, titelID); err != nil {
			t.Errorf("aufräumen: Probe-Titel löschen: %v", err)
		}
	})

	rows := [][]string{
		{"Titel", "Barcode"},
		{"Jahrgang-Probe ohne Angabe", barcode},
	}
	neueTitel, _, err := NewImportService(nil, pool).ImportDynamic(ctx, rows, map[string]int{"titel": 0, "barcode": 1})
	if err != nil {
		t.Fatalf("ImportDynamic: %v", err)
	}
	if neueTitel != 1 {
		t.Fatalf("%d neue Titel, erwartet 1", neueTitel)
	}
	var unbekannt bool
	if err := pool.QueryRow(ctx, `
		SELECT t.jahrgang_von IS NULL AND t.jahrgang_bis IS NULL
		FROM buecher_titel t JOIN buecher_exemplare e ON e.titel_id = t.id
		WHERE e.barcode_id = $1`, barcode).Scan(&unbekannt); err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}
	if !unbekannt {
		t.Error("der importierte Titel trägt eine Jahrgangsspanne, erwartet NULL und NULL")
	}
}

// Nennt die Lernmittel-Signatur mehrere Jahrgänge, steht die Spanne als „von" und „bis" am
// Titel. Mit einem einzigen Jahrgang fiele nicht auf, wenn die zwei Werte vertauscht ankämen.
func TestImportDynamic_SignaturMitSpanne(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const barcode = "JAHRGANG-PROBE-163-1"

	t.Cleanup(func() {
		var titelID string
		if err := pool.QueryRow(ctx, `SELECT coalesce((SELECT titel_id::text FROM buecher_exemplare
			WHERE barcode_id = $1), '')`, barcode).Scan(&titelID); err != nil {
			t.Errorf("aufräumen: Probe-Exemplar lesen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE barcode_id = $1`, barcode); err != nil {
			t.Errorf("aufräumen: Probe-Exemplar löschen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE id::text = $1`, titelID); err != nil {
			t.Errorf("aufräumen: Probe-Titel löschen: %v", err)
		}
	})

	rows := [][]string{
		{"Titel", "Barcode", "Signatur"},
		{"Jahrgang-Probe mit Spanne", barcode, "LMF Bio 7-9"},
	}
	neueTitel, _, err := NewImportService(nil, pool).ImportDynamic(ctx, rows, map[string]int{"titel": 0, "barcode": 1, "signatur": 2})
	if err != nil {
		t.Fatalf("ImportDynamic: %v", err)
	}
	if neueTitel != 1 {
		t.Fatalf("%d neue Titel, erwartet 1", neueTitel)
	}
	var von, bis int
	var lernmittel bool
	if err := pool.QueryRow(ctx, `
		SELECT coalesce(t.jahrgang_von, 0), coalesce(t.jahrgang_bis, 0), t.ist_lernmittel
		FROM buecher_titel t JOIN buecher_exemplare e ON e.titel_id = t.id
		WHERE e.barcode_id = $1`, barcode).Scan(&von, &bis, &lernmittel); err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}
	if von != 7 || bis != 9 || !lernmittel {
		t.Errorf("der importierte Titel trägt %d bis %d (Lernmittel: %v), erwartet 7 bis 9 als Lernmittel", von, bis, lernmittel)
	}
}
