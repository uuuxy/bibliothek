package api

import (
	"context"
	"testing"

	"bibliothek/db"
)

// Nur-Name-Modus: Der Export hat weder ID noch Geburtsdatum (LANIS-Klassenliste).
// Dritte, unsicherste Stufe — deshalb gilt: Gleicher Name doppelt (in der Datei ODER
// im Bestand) wird nie zugeordnet, nur gemeldet.

func nnDatei(zeilen ...parsedStudentRow) lusdDatei {
	return lusdDatei{Zeilen: zeilen, Modus: lusdModusNurName}
}

// Handanlage (mit oder ohne Geburtsdatum) wird über den Namen zugeordnet, Klasse
// übernommen, bestätigt, kein Duplikat. Neuzugang entsteht ohne ID und ohne Datum.
func TestNurName_HandanlageWirdZugeordnet(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}

	mitDatum := legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Mia", nachname: "Hand", klasse: "5a", barcode: "NN-1", geb: datum(2013, 5, 4)})
	ohneDatum := legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Ben", nachname: "Ohne", klasse: "5b", barcode: "NN-2"})

	datei := nnDatei(
		nmZeile(2, "MIA", "hand", "06G1", nil),
		nmZeile(3, "Ben", "Ohne", "6b", nil),
		nmZeile(4, "Neu", "Kind", "5c", nil),
	)
	prev, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if prev.Modus != "name" || len(prev.ClassChanges) != 2 || len(prev.NewStudents) != 1 || len(prev.Mehrdeutig) != 0 || len(prev.NichtAbgleichbar) != 0 {
		t.Fatalf("Vorschau falsch: %+v", prev)
	}
	if _, err := s.computeLusd(ctx, datei, true, true); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if n := zaehle(t, pool, "lower(nachname) IN ('hand','ohne','kind')"); n != 3 {
		t.Fatalf("erwartet 3 Zeilen, waren %d — Duplikate", n)
	}
	for id, klasse := range map[string]string{mitDatum: "06G1", ohneDatum: "6b"} {
		var k string
		var bestaetigt bool
		if err := pool.QueryRow(ctx, `SELECT klasse, lusd_bestaetigt_am IS NOT NULL FROM schueler WHERE id=$1`, id).Scan(&k, &bestaetigt); err != nil {
			t.Fatal(err)
		}
		if !klassenGleich(k, klasse) || !bestaetigt {
			t.Errorf("Schüler %s: Klasse %q (erwartet %q), bestätigt=%v", id, k, klasse, bestaetigt)
		}
	}
}

// Gleicher Name zweimal in der DATEI: beide Zeilen gemeldet, keine angefasst, kein
// Neuzugang. Gleicher Name zweimal im BESTAND: die CSV-Zeile gemeldet, beide bleiben.
func TestNurName_GleicheNamenWerdenNieZugeordnet(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}

	// Bestand: zwei verschiedene "Max Mustermann" (verschiedene Geburtstage = zwei Menschen).
	legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Max", nachname: "Mustermann", klasse: "5a", barcode: "NN-3", geb: datum(2013, 1, 1), bestaetigt: true})
	legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Max", nachname: "Mustermann", klasse: "9c", barcode: "NN-4", geb: datum(2009, 2, 2), bestaetigt: true})
	legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Lea", nachname: "Einzig", klasse: "7a", barcode: "NN-5", geb: datum(2011, 3, 3), bestaetigt: true})

	datei := nnDatei(
		nmZeile(2, "Max", "Mustermann", "6a", nil),
		nmZeile(3, "Lea", "Einzig", "8a", nil),
		nmZeile(4, "Tom", "Doppelt", "5a", nil),
		nmZeile(5, "Tom", "Doppelt", "5b", nil),
	)
	prev, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatal(err)
	}
	// Mehrdeutig: die CSV-Zeile "Max Mustermann" + die beiden Bestandszeilen + die zwei "Tom Doppelt".
	if len(prev.Mehrdeutig) != 5 {
		t.Fatalf("erwartet 5 Mehrdeutig-Einträge, waren %d: %+v", len(prev.Mehrdeutig), prev.Mehrdeutig)
	}
	if len(prev.NewStudents) != 0 || len(prev.ClassChanges) != 1 || len(prev.Graduates) != 0 {
		t.Fatalf("Vorschau falsch: neu=%d wechsel=%d abgänger=%d", len(prev.NewStudents), len(prev.ClassChanges), len(prev.Graduates))
	}
	if _, err := s.computeLusd(ctx, datei, true, true); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if n := zaehle(t, pool, "nachname='Mustermann' AND klasse IN ('5a','9c')"); n != 2 {
		t.Errorf("die beiden Max Mustermann wurden angefasst oder dupliziert (n=%d)", n)
	}
	if n := zaehle(t, pool, "nachname='Doppelt'"); n != 0 {
		t.Errorf("mehrdeutige Neuzugänge dürfen nicht angelegt werden (n=%d)", n)
	}
	var abg bool
	if err := pool.QueryRow(ctx, `SELECT bool_or(ist_abgaenger) FROM schueler WHERE nachname='Mustermann'`).Scan(&abg); err != nil || abg {
		t.Errorf("mehrdeutige Bestandsschüler dürfen nicht als Abgänger gelten (err=%v, abg=%v)", err, abg)
	}
}
