package api

import (
	"context"
	"testing"

	"bibliothek/db"
)

// Handanlage „Anna Müller", Export „Anna Mueller" (LANIS-Klassenliste, nur Name): derselbe
// Mensch. Bis 05.09.2026 war der Namensschlüssel lower+trim — „müller" ≠ „mueller" —, und
// der Import legte „Anna Mueller" neu an und meldete „Anna Müller" als „nicht im Export".
// Der Rückweg war das Zusammenführen über die Akte, sofern es jemand bemerkte. Seit dem
// Schlüssel in der Normalform suchnorm (Migration 054) wird zugeordnet: Klasse übernommen,
// bestätigt, KEIN Duplikat (Befund-Register, Entscheidung 1 vom 05.09.2026).
//
// Gegenprobe im selben Lauf: Zwei verschiedene Bestandsschüler, deren Namen erst in der
// Normalform zusammenfallen („Bauer" und „Baur"), werden NICHT zugeordnet, sondern als
// mehrdeutig gemeldet und nicht angefasst — weder neu angelegt noch zu Abgängern gemacht.
func TestNurNameModus_UmlautSchreibweisenSindDerselbeMensch(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}

	muellerID := legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Anna", nachname: "Müller", klasse: "5a", barcode: "UM-1"})
	bauerID := legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Tom", nachname: "Bauer", klasse: "7b", barcode: "UM-2", bestaetigt: true})
	baurID := legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Tom", nachname: "Baur", klasse: "7c", barcode: "UM-3", bestaetigt: true})

	datei := lusdDatei{Modus: lusdModusNurName, Zeilen: []parsedStudentRow{
		nmZeile(2, "Anna", "Mueller", "06A", nil),
		nmZeile(3, "Tom", "Bauer", "08B", nil),
	}}
	prev, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(prev.NewStudents) != 0 || len(prev.NichtImExport) != 0 {
		t.Errorf("Anna Mueller muss Anna Müller zugeordnet werden — neu=%d, nichtImExport=%d (Schlüssel nur lower+trim?)",
			len(prev.NewStudents), len(prev.NichtImExport))
	}
	if len(prev.Mehrdeutig) == 0 || len(prev.Graduates) != 0 {
		t.Errorf("Bauer/Baur müssen als mehrdeutig gemeldet werden und keine Abgänger sein — mehrdeutig=%d, abgaenger=%d",
			len(prev.Mehrdeutig), len(prev.Graduates))
	}

	if _, err := s.computeLusd(ctx, datei, true, true); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if n := zaehle(t, pool, "suchnorm(nachname) = suchnorm('Müller')"); n != 1 {
		t.Errorf("erwartet genau eine Müller/Mueller-Zeile, waren %d — Duplikat angelegt", n)
	}
	var klasse string
	var bestaetigt bool
	if err := pool.QueryRow(ctx, `SELECT klasse, lusd_bestaetigt_am IS NOT NULL FROM schueler WHERE id=$1`, muellerID).Scan(&klasse, &bestaetigt); err != nil {
		t.Fatal(err)
	}
	if !klassenGleich(klasse, "06A") || !bestaetigt {
		t.Errorf("Anna Müller: Klasse %q (erwartet 06A), bestätigt=%v", klasse, bestaetigt)
	}
	for _, id := range []string{bauerID, baurID} {
		var abgaenger bool
		if err := pool.QueryRow(ctx, `SELECT ist_abgaenger FROM schueler WHERE id=$1`, id).Scan(&abgaenger); err != nil {
			t.Fatal(err)
		}
		if abgaenger {
			t.Errorf("mehrdeutiger Bestandsschüler %s wurde zum Abgänger gemacht", id)
		}
	}
}
