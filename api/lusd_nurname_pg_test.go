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

// Prüfung 22.08.2026, Fund A1: Steht ein Name ZWEIMAL in der Datei, sind beide Zeilen
// mehrdeutig — aber ein bestätigter Bestandsschüler dieses Namens ist deshalb nicht „nicht
// im Export". Vorher setzte der Mehrdeutig-Zweig `gesehen` nicht, und sammleAbgaenger
// machte ihn zum Abgänger: Vorschau zeigte ihn unter „wird nicht angefasst" UND unter
// „Abgänger", Apply anonymisierte ihn.
func TestNurName_DoppelnameInDateiMachtBestandNichtZumAbgaenger(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}

	tom := legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Tom", nachname: "Doppelt", klasse: "5a", barcode: "NN-6", geb: datum(2014, 1, 1), bestaetigt: true})
	// Genug unbeteiligte Bestätigte, damit die Massenabgangs-Bremse nicht greift.
	for i := 0; i < 12; i++ {
		legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Ruhig", nachname: "Kind" + string(rune('A'+i)), klasse: "7a", barcode: "NN-R" + string(rune('A'+i)), geb: datum(2012, 1, 1+i), bestaetigt: true})
	}
	zeilen := []parsedStudentRow{nmZeile(2, "Tom", "Doppelt", "6a", nil), nmZeile(3, "Tom", "Doppelt", "6b", nil)}
	for i := 0; i < 12; i++ {
		zeilen = append(zeilen, nmZeile(4+i, "Ruhig", "Kind"+string(rune('A'+i)), "8a", nil))
	}
	datei := nnDatei(zeilen...)
	prev, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Graduates) != 0 {
		t.Fatalf("Bestandsschüler mit Doppelname in der Datei darf KEIN Abgänger sein: %+v", prev.Graduates)
	}
	if len(prev.Mehrdeutig) != 2 {
		t.Fatalf("erwartet 2 Mehrdeutig-Zeilen, waren %d", len(prev.Mehrdeutig))
	}
	if _, err := s.computeLusd(ctx, datei, true, true); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	var abg bool
	var vorname string
	if err := pool.QueryRow(ctx, `SELECT ist_abgaenger, vorname FROM schueler WHERE id=$1`, tom).Scan(&abg, &vorname); err != nil {
		t.Fatal(err)
	}
	if abg || vorname != "Tom" {
		t.Errorf("Tom wurde angefasst: abgaenger=%v vorname=%q", abg, vorname)
	}
}

// Prüfung 22.08.2026, Fund A2: Im Nur-Name-Modus ist ein Abgänger mit demselben Namen
// KEIN sicherer Rückkehrer — ein neuer Fünftklässler „Max Alt" landete sonst auf dem
// Datensatz (Sperre, Schulden, Lesehistorie) des abgegangenen Zehntklässlers. Jetzt:
// gemeldet als mehrdeutig, nichts angefasst, kein Neuzugang.
func TestNurName_AbgaengerNamensvetterWirdNichtReaktiviert(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}

	alt := legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Max", nachname: "Alt", klasse: "10a", barcode: "NN-7", geb: datum(2008, 3, 3), abgaenger: true, bestaetigt: true})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET ist_gesperrt=true, block_reason='Automatisierte Abgänger-Sperre (offene Vorgänge)' WHERE id=$1`, alt); err != nil {
		t.Fatal(err)
	}
	datei := nnDatei(nmZeile(2, "Max", "Alt", "5a", nil))
	prev, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Rueckkehrer) != 0 || len(prev.Mehrdeutig) != 1 || len(prev.NewStudents) != 0 {
		t.Fatalf("Nur-Name: Abgänger-Namensvetter muss mehrdeutig sein, nicht Rückkehrer/neu: %+v", prev)
	}
	if _, err := s.computeLusd(ctx, datei, true, true); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	var abg, gesperrt bool
	var klasse string
	if err := pool.QueryRow(ctx, `SELECT ist_abgaenger, ist_gesperrt, klasse FROM schueler WHERE id=$1`, alt).Scan(&abg, &gesperrt, &klasse); err != nil {
		t.Fatal(err)
	}
	if !abg || !gesperrt || klasse != "10a" {
		t.Errorf("Abgänger wurde angefasst: abgaenger=%v gesperrt=%v klasse=%q", abg, gesperrt, klasse)
	}
	if n := zaehle(t, pool, "nachname='Alt'"); n != 1 {
		t.Errorf("erwartet 1 Zeile 'Alt', waren %d", n)
	}
}
