package service

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Der kombinierte CSV-Import ordnet eine Zeile über die ISBN einem Titel zu. Die Datenbank
// speichert eine zehnstellige ISBN dreizehnstellig (Migration 157); nennt die Datei dasselbe
// Buch in beiden Längen, ist es ein Titel mit zwei Exemplaren. Als zwei Schlüssel ergab es
// zwei INSERTs, und der zweite scheiterte am UNIQUE-Index — der ganze Import.
func TestImportDynamic_BeideLaengenDerISBNSindEinTitel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const zehn, dreizehn = "0306406152", "9780306406157"
	barcodes := []string{"ISBN-LAENGE-157-1", "ISBN-LAENGE-157-2", "ISBN-LAENGE-157-3"}
	raeume := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE barcode_id = ANY($1)`, barcodes); err != nil {
			t.Errorf("aufräumen: Probe-Exemplare löschen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = ANY($1)`, []string{zehn, dreizehn}); err != nil {
			t.Errorf("aufräumen: Probe-Titel löschen: %v", err)
		}
	}
	raeume()
	t.Cleanup(raeume)

	kopf := map[string]int{"titel": 0, "isbn": 1, "barcode": 2}
	neueTitel, neueExemplare, err := NewImportService(nil, pool).ImportDynamic(ctx, [][]string{
		{"Titel", "ISBN", "Barcode"},
		{"Advanced Organic Chemistry", "0-306-40615-2", barcodes[0]},
		{"Advanced Organic Chemistry, Part A", "978-0-306-40615-7", barcodes[1]},
	}, kopf)
	if err != nil {
		t.Fatalf("ImportDynamic: %v", err)
	}
	if neueTitel != 1 || neueExemplare != 2 {
		t.Errorf("beide Längen in einer Datei: %d neue Titel, %d neue Exemplare, erwartet 1 und 2", neueTitel, neueExemplare)
	}

	// Ein zweiter Lauf nennt das Buch nur zehnstellig, unter einem dritten Titeltext: Das
	// Exemplar kommt an den vorhandenen Titel.
	neueTitel, neueExemplare, err = NewImportService(nil, pool).ImportDynamic(ctx, [][]string{
		{"Titel", "ISBN", "Barcode"},
		{"Organic Chemistry", zehn, barcodes[2]},
	}, kopf)
	if err != nil {
		t.Fatalf("zweiter Lauf: %v", err)
	}
	if neueTitel != 0 || neueExemplare != 1 {
		t.Errorf("zweiter Lauf: %d neue Titel, %d neue Exemplare, erwartet 0 und 1", neueTitel, neueExemplare)
	}

	var titel, exemplare int
	if err := pool.QueryRow(ctx, `
		SELECT count(DISTINCT t.id)::int, count(e.id)::int
		FROM buecher_titel t JOIN buecher_exemplare e ON e.titel_id = t.id
		WHERE e.barcode_id = ANY($1) AND t.isbn = $2`, barcodes, dreizehn).Scan(&titel, &exemplare); err != nil {
		t.Fatal(err)
	}
	if titel != 1 || exemplare != 3 {
		t.Errorf("%d Titel unter %s mit %d Exemplaren, erwartet 1 mit 3", titel, dreizehn, exemplare)
	}
}
