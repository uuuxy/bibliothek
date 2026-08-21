package api

import (
	"context"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Namensmodus: Der LUSD-Export der Schule hat keine Schüler-ID. Zuordnung über
// Vorname + Nachname + Geburtsdatum, Jahr für Jahr. Diese Tests spielen den echten
// Jahreszyklus durch — Handanlage → erster Import → zweiter Import — und belegen an der
// Datenbank, dass nichts doppelt entsteht und nichts Unbeteiligtes anonymisiert wird.

type nmSchueler struct {
	vorname, nachname, klasse, barcode string
	geb                                *time.Time
	lusdID                             *string
	abgaenger, bestaetigt              bool
}

func datum(j int, m time.Month, t int) *time.Time {
	d := time.Date(j, m, t, 0, 0, 0, 0, time.UTC)
	return &d
}

func legeNmSchuelerAn(t *testing.T, ctx context.Context, pool *pgxpool.Pool, s nmSchueler) string {
	t.Helper()
	var id string
	var bestaetigt *time.Time
	if s.bestaetigt {
		bestaetigt = datum(2025, 8, 1)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum,
		                      lusd_id, ist_abgaenger, lusd_bestaetigt_am)
		VALUES ($1,$2,$3,$4,2031,$5,$6,$7,$8) RETURNING id`,
		s.vorname, s.nachname, s.klasse, s.barcode, s.geb, s.lusdID, s.abgaenger, bestaetigt).Scan(&id); err != nil {
		t.Fatalf("Schüler %s %s anlegen: %v", s.vorname, s.nachname, err)
	}
	return id
}

func nmZeile(line int, vorname, nachname, klasse string, geb *time.Time) parsedStudentRow {
	return parsedStudentRow{Vorname: vorname, Nachname: nachname, Klasse: klasse, GebDatum: geb, LineNum: line}
}

func nmDatei(zeilen ...parsedStudentRow) lusdDatei {
	return lusdDatei{Zeilen: zeilen, Modus: lusdModusName}
}

func zaehle(t *testing.T, pool *pgxpool.Pool, where string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM schueler WHERE deleted_at IS NULL AND `+where, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Der Kernfall des Sekretariats: Schüler von Hand angelegt (mit Geburtsdatum), später
// kommt der LUSD-Export ohne ID — derselbe Mensch, neue Klasse. Erwartung: zugeordnet,
// Klasse übernommen, bestätigt, KEIN Duplikat. Der zweite Jahreslauf ändert nichts.
func TestNamensmodus_HandanlageWirdZugeordnetNichtDupliziert(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}
	litteraID := "littera:77"

	handID := legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Mia", nachname: "Hand", klasse: "5a", barcode: "NM-1", geb: datum(2013, 5, 4)})
	littID := legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Leo", nachname: "Littera", klasse: "6b", barcode: "NM-2", geb: datum(2012, 1, 2), lusdID: &litteraID})

	datei := nmDatei(
		nmZeile(2, "mia", "HAND", "06A", datum(2013, 5, 4)), // Schreibweise + führende Null: trotzdem dieselbe Person
		nmZeile(3, "Leo", "Littera", "7b", datum(2012, 1, 2)),
		nmZeile(4, "Neu", "Kind", "5c", datum(2014, 9, 9)),
	)
	prev, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if prev.Modus != "name_geburtsdatum" || len(prev.ClassChanges) != 2 || len(prev.NewStudents) != 1 ||
		len(prev.Graduates) != 0 || len(prev.Adoptions) != 0 || len(prev.Mehrdeutig) != 0 {
		t.Fatalf("Vorschau falsch: %+v", prev)
	}
	if _, err := s.computeLusd(ctx, datei, true, true); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// lower(): Der Import übernimmt die Schreibweise des Exports ("HAND") — LUSD ist die Quelle.
	if n := zaehle(t, pool, "lower(nachname) IN ('hand','littera','kind')"); n != 3 {
		t.Fatalf("erwartet 3 Zeilen (2 zugeordnet + 1 neu), waren %d — Duplikate", n)
	}
	for id, klasse := range map[string]string{handID: "06A", littID: "7b"} {
		var k string
		var bestaetigt *time.Time
		if err := pool.QueryRow(ctx, `SELECT klasse, lusd_bestaetigt_am FROM schueler WHERE id=$1`, id).Scan(&k, &bestaetigt); err != nil {
			t.Fatal(err)
		}
		// klassenGleich statt ==: Der Klassen-Trigger (Migration 079) kanonisiert "06A" auf
		// die registrierte Schreibweise ("6a"), wenn die Schule so schreibt.
		if !klassenGleich(k, klasse) || bestaetigt == nil {
			t.Errorf("Schüler %s: Klasse %q (erwartet %q), bestätigt=%v", id, k, klasse, bestaetigt != nil)
		}
	}
	var neuLusd *string
	if err := pool.QueryRow(ctx, `SELECT lusd_id FROM schueler WHERE nachname='Kind'`).Scan(&neuLusd); err != nil {
		t.Fatal(err)
	}
	if neuLusd != nil {
		t.Errorf("Neuzugang im Namensmodus muss lusd_id NULL haben, war %q", *neuLusd)
	}

	// Zweiter Jahreslauf mit derselben Datei: nichts zu tun, niemand geht ab.
	prev2, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev2.NewStudents)+len(prev2.ClassChanges)+len(prev2.Graduates)+len(prev2.NichtImExport) != 0 {
		t.Errorf("zweiter Lauf muss leer sein: %+v", prev2)
	}
}

// Abgänger nur mit Gedächtnis: Ein BESTÄTIGTER Schüler, der im Export fehlt, geht ab.
// Eine nie bestätigte Handanlage, die fehlt, bleibt stehen und wird gemeldet. Ein
// Schüler ohne Geburtsdatum ist nicht abgleichbar und bleibt ebenfalls stehen.
func TestNamensmodus_AbgaengerNurWennBestaetigt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}

	abID := legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Alt", nachname: "Bestaetigt", klasse: "10a", barcode: "NM-3", geb: datum(2009, 2, 2), bestaetigt: true})
	handID := legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Gast", nachname: "Schueler", klasse: "8a", barcode: "NM-4", geb: datum(2011, 3, 3)})
	ohneID := legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Ohne", nachname: "Datum", klasse: "8b", barcode: "NM-5"})

	datei := nmDatei(nmZeile(2, "Anders", "Jemand", "5a", datum(2014, 4, 4)))
	prev, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Graduates) != 1 || prev.Graduates[0].ID != abID {
		t.Fatalf("erwartet genau den bestätigten Schüler als Abgänger: %+v", prev.Graduates)
	}
	if len(prev.NichtImExport) != 1 || prev.NichtImExport[0].ID != handID {
		t.Fatalf("Handanlage muss als „nicht im Export“ gemeldet werden: %+v", prev.NichtImExport)
	}
	if len(prev.NichtAbgleichbar) != 1 || prev.NichtAbgleichbar[0].ID != ohneID {
		t.Fatalf("Schüler ohne Geburtsdatum muss als „nicht abgleichbar“ gemeldet werden: %+v", prev.NichtAbgleichbar)
	}
	if _, err := s.computeLusd(ctx, datei, true, true); err != nil {
		t.Fatal(err)
	}
	var abg bool
	if err := pool.QueryRow(ctx, `SELECT ist_abgaenger FROM schueler WHERE id=$1`, abID).Scan(&abg); err != nil || !abg {
		t.Errorf("bestätigter Schüler muss Abgänger sein (err=%v, abg=%v)", err, abg)
	}
	for _, id := range []string{handID, ohneID} {
		var abg bool
		var vorname string
		if err := pool.QueryRow(ctx, `SELECT ist_abgaenger, vorname FROM schueler WHERE id=$1`, id).Scan(&abg, &vorname); err != nil {
			t.Fatal(err)
		}
		if abg || vorname == "Abgänger" {
			t.Errorf("Schüler %s wurde angefasst (abgaenger=%v, vorname=%q) — er stand nie in einem Export", id, abg, vorname)
		}
	}
}

// Rückkehrer: Ein gesperrter Abgänger (offene Vorgänge → Name erhalten) steht wieder
// im Export — derselbe Datensatz wird reaktiviert, kein Duplikat, keine 23505 am
// Unique-Index unique_schueler_name_gebdatum.
func TestNamensmodus_RueckkehrerWirdReaktiviert(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}

	id := legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Rück", nachname: "Kehrer", klasse: "9a", barcode: "NM-6", geb: datum(2010, 6, 6), abgaenger: true, bestaetigt: true})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET ist_gesperrt=true, block_reason='Automatisierte Abgänger-Sperre (offene Vorgänge)' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}

	datei := nmDatei(nmZeile(2, "Rück", "Kehrer", "10a", datum(2010, 6, 6)))
	prev, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Rueckkehrer) != 1 || len(prev.NewStudents) != 0 {
		t.Fatalf("erwartet 1 Rückkehrer, 0 Neuzugänge: %+v", prev)
	}
	if _, err := s.computeLusd(ctx, datei, true, true); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if n := zaehle(t, pool, "nachname='Kehrer'"); n != 1 {
		t.Fatalf("Rückkehrer dupliziert: %d Zeilen", n)
	}
	var abg, gesperrt bool
	var klasse string
	if err := pool.QueryRow(ctx, `SELECT ist_abgaenger, ist_gesperrt, klasse FROM schueler WHERE id=$1`, id).Scan(&abg, &gesperrt, &klasse); err != nil {
		t.Fatal(err)
	}
	if abg || gesperrt || klasse != "10a" {
		t.Errorf("Rückkehrer nicht reaktiviert: abgaenger=%v gesperrt=%v klasse=%q", abg, gesperrt, klasse)
	}
}

// Mehrdeutig: zwei aktive Zeilen mit demselben Schlüssel (case-sensitiver Unique-Index
// lässt "Anna Müller"/"anna müller" zu, sofern eine eine Littera-Marke trägt). Die CSV-
// Zeile wird NICHT zugeordnet und NICHT neu angelegt; beide bleiben unverändert und
// werden gemeldet.
func TestNamensmodus_MehrdeutigWirdNichtAngefasst(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}
	marke := "littera:5"

	legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "Anna", nachname: "Müller", klasse: "7a", barcode: "NM-7", geb: datum(2012, 7, 7)})
	legeNmSchuelerAn(t, ctx, pool, nmSchueler{vorname: "anna", nachname: "müller", klasse: "7b", barcode: "NM-8", geb: datum(2012, 7, 7), lusdID: &marke})

	datei := nmDatei(nmZeile(2, "Anna", "Müller", "8a", datum(2012, 7, 7)))
	prev, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Mehrdeutig) == 0 || len(prev.NewStudents) != 0 || len(prev.ClassChanges) != 0 {
		t.Fatalf("erwartet Mehrdeutig-Meldung ohne Neuzugang/Wechsel: %+v", prev)
	}
	if _, err := s.computeLusd(ctx, datei, true, true); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if n := zaehle(t, pool, "lower(nachname)='müller'"); n != 2 {
		t.Fatalf("erwartet weiterhin 2 Zeilen, waren %d", n)
	}
	if n := zaehle(t, pool, "lower(nachname)='müller' AND klasse IN ('7a','7b')"); n != 2 {
		t.Error("mehrdeutige Schüler wurden verändert")
	}
}
