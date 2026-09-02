package api

import (
	"context"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Umbenennung ohne Schüler-ID — der ganze Weg an der Datenbank: Vorschau schlägt das
// Paar vor, der Admin bestätigt, derselbe Datensatz trägt den neuen Namen; unbestätigt
// bleibt es Abgänger + Neuanlage (jetzt mit Karenz-Sperre statt Anonymisierung).

type umbSchueler struct {
	vorname, nachname, klasse, barcode string
	geb, eintritt                      *time.Time
	abgaenger                          bool
}

func legeUmbSchuelerAn(t *testing.T, pool *pgxpool.Pool, s umbSchueler) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum,
		                      schul_eintritt_am, ist_abgaenger, lusd_bestaetigt_am, ist_gesperrt, block_reason, abgaenger_seit)
		VALUES ($1,$2,$3,$4,2031,$5,$6,$7,NOW() - interval '1 year',$7,
		        CASE WHEN $7 THEN 'Automatisierte Abgänger-Sperre (Karenzzeit vor Anonymisierung)' END,
		        CASE WHEN $7 THEN NOW() - interval '10 days' END)
		RETURNING id`,
		s.vorname, s.nachname, s.klasse, s.barcode, s.geb, s.eintritt, s.abgaenger).Scan(&id); err != nil {
		t.Fatalf("Schüler %s %s anlegen: %v", s.vorname, s.nachname, err)
	}
	return id
}

func umbZeile(line int, vorname, nachname, klasse string, geb, eintritt *time.Time) parsedStudentRow {
	return parsedStudentRow{Vorname: vorname, Nachname: nachname, Klasse: klasse, GebDatum: geb, EintrittAm: eintritt, LineNum: line}
}

// Namensänderung mit Schuleintritt im Bericht: sicheres Paar. Bestätigt → derselbe
// Datensatz (ID, Barcode, offene Ausleihe) trägt den neuen Namen, keine zweite Zeile,
// und der nächste Lauf mit derselben Datei ist leer.
func TestUmbenennung_BestaetigtBehaeltDatensatz(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}
	geb, eintritt := datum(2013, 5, 4), datum(2024, 8, 19)

	annaID := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Anna", nachname: "Müller", klasse: "05F1", barcode: "UMB-1", geb: geb, eintritt: eintritt})
	seedOffeneAusleihe(t, pool, annaID, "UMB1")
	// Unbeteiligte, damit die Massenabgang-Bremse nicht die Sicht verstellt.
	for i := 0; i < 12; i++ {
		legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Ruhig", nachname: "Kind" + string(rune('A'+i)), klasse: "07A", barcode: "UMB-R" + string(rune('A'+i)), geb: datum(2012, 1, 1+i)})
	}
	zeilen := []parsedStudentRow{umbZeile(2, "Anna", "Mueller-Schmidt", "06F1", geb, eintritt)}
	for i := 0; i < 12; i++ {
		zeilen = append(zeilen, umbZeile(3+i, "Ruhig", "Kind"+string(rune('A'+i)), "08A", datum(2012, 1, 1+i), nil))
	}
	datei := lusdDatei{Zeilen: zeilen, Modus: lusdModusName}

	prev, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(prev.NewStudents) != 1 || len(prev.Graduates) != 1 || len(prev.Umbenennungen) != 1 {
		t.Fatalf("Vorschau: neu=%d abg=%d umb=%d — erwartet je 1", len(prev.NewStudents), len(prev.Graduates), len(prev.Umbenennungen))
	}
	p := prev.Umbenennungen[0]
	if p.Zeile != 2 || p.SchuelerID != annaID || !p.Sicher || p.WarAbgaenger || p.AltNachname != "Müller" || p.NeuNachname != "Mueller-Schmidt" {
		t.Fatalf("Paar falsch: %+v", p)
	}
	if prev.KarenzTage != 90 {
		t.Errorf("Vorschau nennt Karenz %d, erwartet Vorgabe 90", prev.KarenzTage)
	}

	res, err := s.computeLusdLauf(ctx, datei, lusdLauf{apply: true, allowMassGraduation: true,
		umbenennungen: []umbenennungWahl{{Zeile: 2, SchuelerID: annaID}}})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(res.NewStudents) != 0 || len(res.Graduates) != 0 || !res.Umbenennungen[0].Bestaetigt {
		t.Errorf("Ergebnis nach Bestätigung: neu=%d abg=%d bestaetigt=%v", len(res.NewStudents), len(res.Graduates), res.Umbenennungen[0].Bestaetigt)
	}
	if n := zaehle(t, pool, "lower(nachname) LIKE 'm%'"); n != 1 {
		t.Fatalf("erwartet EINE Zeile für Anna, waren %d", n)
	}
	var nachname, klasse, barcode string
	var abg bool
	var bestaetigt, eintrittDB *time.Time
	if err := pool.QueryRow(ctx, `SELECT nachname, klasse, barcode_id, ist_abgaenger, lusd_bestaetigt_am, schul_eintritt_am FROM schueler WHERE id=$1`, annaID).
		Scan(&nachname, &klasse, &barcode, &abg, &bestaetigt, &eintrittDB); err != nil {
		t.Fatal(err)
	}
	if nachname != "Mueller-Schmidt" || !klassenGleich(klasse, "06F1") || barcode != "UMB-1" || abg || bestaetigt == nil || time.Since(*bestaetigt) > time.Minute {
		t.Errorf("Anna: nachname=%q klasse=%q barcode=%q abg=%v bestaetigt=%v", nachname, klasse, barcode, abg, bestaetigt)
	}
	// Der Import SCHREIBT den Schuleintritt (Bestand + Neuanlage) — kein Test sah das
	// bisher, alle säten die Spalte per SQL (Rasterdurchgang 02.09.2026, Rot-Proben P16).
	if eintrittDB == nil || eintrittDB.Format("2006-01-02") != eintritt.Format("2006-01-02") {
		t.Errorf("Bestandsschüler: schul_eintritt_am aus dem Export nicht geschrieben: %v", eintrittDB)
	}
	var neuEintritt *time.Time
	if err := pool.QueryRow(ctx, `SELECT schul_eintritt_am FROM schueler WHERE nachname = 'KindA'`).Scan(&neuEintritt); err != nil {
		t.Fatal(err)
	}
	if neuEintritt != nil {
		t.Errorf("Neuanlage ohne Eintritt im Export darf keinen tragen: %v", neuEintritt)
	}
	if n := zaehle(t, pool, "id = $1 AND EXISTS (SELECT 1 FROM ausleihen a WHERE a.schueler_id = schueler.id AND a.rueckgabe_am IS NULL)", annaID); n != 1 {
		t.Error("die offene Ausleihe hängt nicht mehr an Anna")
	}

	prev2, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev2.NewStudents)+len(prev2.Graduates)+len(prev2.Umbenennungen)+len(prev2.ClassChanges) != 0 {
		t.Errorf("zweiter Lauf muss leer sein: neu=%d abg=%d umb=%d wechsel=%d", len(prev2.NewStudents), len(prev2.Graduates), len(prev2.Umbenennungen), len(prev2.ClassChanges))
	}
}

// Ohne Bestätigung bleibt es Abgänger + Neuanlage — mit Karenz-Sperre, nicht mit
// Anonymisierung. Und eine Wahl, die die Vorschau nie gemacht hat, wird abgewiesen.
func TestUmbenennung_UnbestaetigtBleibtAbgaengerMitKarenz(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}
	geb := datum(2013, 5, 4)

	annaID := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Anna", nachname: "Müller", klasse: "05F1", barcode: "UMB-2", geb: geb})
	datei := lusdDatei{Modus: lusdModusName, Zeilen: []parsedStudentRow{umbZeile(2, "Anna", "Schulz", "06F1", geb, nil)}}

	err := func() error {
		_, err := s.computeLusdLauf(ctx, datei, lusdLauf{apply: true, allowMassGraduation: true,
			umbenennungen: []umbenennungWahl{{Zeile: 2, SchuelerID: "00000000-0000-0000-0000-000000000000"}}})
		return err
	}()
	if _, ok := err.(*errUmbenennungUngueltig); !ok {
		t.Fatalf("fremde Wahl muss abgewiesen werden, bekam %v", err)
	}
	if n := zaehle(t, pool, "nachname = 'Schulz'"); n != 0 {
		t.Fatal("die abgewiesene Wahl hat trotzdem geschrieben")
	}

	if _, err := s.computeLusd(ctx, datei, true, true); err != nil {
		t.Fatalf("Apply ohne Wahl: %v", err)
	}
	if n := zaehle(t, pool, "nachname IN ('Müller','Schulz')"); n != 2 {
		t.Fatalf("erwartet Abgänger + Neuanlage (2 Zeilen), waren %d", n)
	}
	var vorname, grund string
	var abg, gesperrt, seit bool
	if err := pool.QueryRow(ctx, `SELECT vorname, ist_abgaenger, ist_gesperrt, COALESCE(block_reason,''), abgaenger_seit IS NOT NULL FROM schueler WHERE id=$1`, annaID).
		Scan(&vorname, &abg, &gesperrt, &grund, &seit); err != nil {
		t.Fatal(err)
	}
	if vorname != "Anna" || !abg || !gesperrt || grund != abgaengerSperrgrundKarenz || !seit {
		t.Errorf("Karenz-Abgänger falsch: vorname=%q abg=%v gesperrt=%v grund=%q seit=%v", vorname, abg, gesperrt, grund, seit)
	}
}

// Datumskorrektur der LUSD (Name gleich, Klasse benachbart, Datum anders): vermutliches
// Paar; bestätigt übernimmt der Datensatz das neue Geburtsdatum — sonst fände ihn der
// nächste Lauf wieder nicht.
func TestUmbenennung_DatumskorrekturUebernimmtGeburtsdatum(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}

	benID := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Ben", nachname: "Meier", klasse: "06B", barcode: "UMB-3", geb: datum(2012, 1, 1)})
	datei := lusdDatei{Modus: lusdModusName, Zeilen: []parsedStudentRow{umbZeile(2, "Ben", "Meier", "07B", datum(2012, 1, 10), nil)}}

	prev, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Umbenennungen) != 1 || prev.Umbenennungen[0].Sicher || prev.Umbenennungen[0].NeuGeburtsdatum != "2012-01-10" {
		t.Fatalf("erwartet ein vermutliches Paar mit neuem Datum: %+v", prev.Umbenennungen)
	}
	if _, err := s.computeLusdLauf(ctx, datei, lusdLauf{apply: true, allowMassGraduation: true,
		umbenennungen: []umbenennungWahl{{Zeile: 2, SchuelerID: benID}}}); err != nil {
		t.Fatal(err)
	}
	var geb time.Time
	if err := pool.QueryRow(ctx, `SELECT geburtsdatum FROM schueler WHERE id=$1`, benID).Scan(&geb); err != nil {
		t.Fatal(err)
	}
	if geb.Format("2006-01-02") != "2012-01-10" || zaehle(t, pool, "nachname='Meier'") != 1 {
		t.Errorf("Geburtsdatum nicht übernommen (%v) oder Duplikat", geb)
	}
}

// Ein Abgänger aus einem FRÜHEREN Lauf (gesperrt, in der Karenz) taucht umbenannt
// wieder auf: Paar mit war_abgaenger; bestätigt wird er reaktiviert — entsperrt, ohne
// Abgänger-Stempel, mit neuem Namen.
func TestUmbenennung_FruehererAbgaengerWirdReaktiviert(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Server{DB: &db.Database{Pool: pool}}
	geb, eintritt := datum(2011, 3, 3), datum(2022, 8, 22)

	altID := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Cem", nachname: "Yilmaz", klasse: "08C", barcode: "UMB-4", geb: geb, eintritt: eintritt, abgaenger: true})
	datei := lusdDatei{Modus: lusdModusName, Zeilen: []parsedStudentRow{umbZeile(2, "Cem", "Yılmaz-Kaya", "09C", geb, eintritt)}}

	prev, err := s.computeLusd(ctx, datei, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Umbenennungen) != 1 || !prev.Umbenennungen[0].WarAbgaenger || !prev.Umbenennungen[0].Sicher {
		t.Fatalf("erwartet sicheres Paar mit früherem Abgänger: %+v", prev.Umbenennungen)
	}
	if _, err := s.computeLusdLauf(ctx, datei, lusdLauf{apply: true, allowMassGraduation: true,
		umbenennungen: []umbenennungWahl{{Zeile: 2, SchuelerID: altID}}}); err != nil {
		t.Fatal(err)
	}
	var nachname string
	var abg, gesperrt, seit bool
	if err := pool.QueryRow(ctx, `SELECT nachname, ist_abgaenger, ist_gesperrt, abgaenger_seit IS NOT NULL FROM schueler WHERE id=$1`, altID).
		Scan(&nachname, &abg, &gesperrt, &seit); err != nil {
		t.Fatal(err)
	}
	if nachname != "Yılmaz-Kaya" || abg || gesperrt || seit || zaehle(t, pool, "vorname='Cem'") != 1 {
		t.Errorf("Rückkehrer nicht reaktiviert: nachname=%q abg=%v gesperrt=%v seit=%v", nachname, abg, gesperrt, seit)
	}
}
