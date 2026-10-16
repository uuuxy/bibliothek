package leser

import (
	"context"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/lusd"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Umbenennung ohne Schüler-ID — der ganze Weg an der Datenbank: Vorschau schlägt das
// Paar vor, der Admin bestätigt, derselbe Datensatz trägt den neuen Namen; unbestätigt
// bleibt es Abgänger + Neuanlage (mit Karenz-Sperre statt Anonymisierung).

func umbZeile(line int, vorname, nachname, klasse string, geb, eintritt *time.Time) lusd.Zeile {
	return lusd.Zeile{Vorname: vorname, Nachname: nachname, Klasse: klasse, Geburtsdatum: geb, EintrittAm: eintritt, Nr: line}
}

// Namensänderung mit Schuleintritt im Bericht: sicheres Paar. Bestätigt → derselbe
// Datensatz (ID, Barcode, offene Ausleihe) trägt den neuen Namen, keine zweite Zeile,
// und der nächste Lauf mit derselben Datei ist leer.
func TestUmbenennung_BestaetigtBehaeltDatensatz(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	geb, eintritt := testdaten.Datum(2013, 5, 4), testdaten.Datum(2024, 8, 19)

	annaID := testdaten.LegeLUSDSchuelerAn(t, pool, testdaten.LUSDSchuelerNeu{Vorname: "Anna", Nachname: "Müller", Klasse: "05F1", Barcode: "UMB-1", Geburtsdatum: geb, SchulEintrittAm: eintritt})
	testdaten.LegeOffeneAusleiheAn(t, pool, annaID, "UMB1")
	// Unbeteiligte, damit die Massenabgang-Bremse nicht die Sicht verstellt.
	for i := 0; i < 12; i++ {
		testdaten.LegeLUSDSchuelerAn(t, pool, testdaten.LUSDSchuelerNeu{Vorname: "Ruhig", Nachname: "Kind" + string(rune('A'+i)), Klasse: "07A", Barcode: "UMB-R" + string(rune('A'+i)), Geburtsdatum: testdaten.Datum(2012, 1, 1+i)})
	}
	zeilen := []lusd.Zeile{umbZeile(2, "Anna", "Mueller-Schmidt", "06F1", geb, eintritt)}
	for i := 0; i < 12; i++ {
		zeilen = append(zeilen, umbZeile(3+i, "Ruhig", "Kind"+string(rune('A'+i)), "08A", testdaten.Datum(2012, 1, 1+i), nil))
	}
	datei := lusd.Datei{Zeilen: zeilen, Modus: lusd.ModusName}

	prev, err := s.berechneLUSD(ctx, datei, false, false)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(prev.Neuzugaenge) != 1 || len(prev.Abgaenger) != 1 || len(prev.Umbenennungen) != 1 {
		t.Fatalf("Vorschau: neu=%d abg=%d umb=%d — erwartet je 1", len(prev.Neuzugaenge), len(prev.Abgaenger), len(prev.Umbenennungen))
	}
	p := prev.Umbenennungen[0]
	if p.Zeile != 2 || p.SchuelerID != annaID || !p.Sicher || p.WarAbgaenger || p.AltNachname != "Müller" || p.NeuNachname != "Mueller-Schmidt" {
		t.Fatalf("Paar falsch: %+v", p)
	}
	if prev.KarenzTage != 90 {
		t.Errorf("Vorschau nennt Karenz %d, erwartet Vorgabe 90", prev.KarenzTage)
	}

	res, err := s.berechneLUSDLauf(ctx, datei, lusd.Lauf{Anwenden: true, MassenabgangBestaetigt: true,
		Umbenennungen: []lusd.UmbenennungWahl{{Zeile: 2, SchuelerID: annaID}}})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(res.Neuzugaenge) != 0 || len(res.Abgaenger) != 0 || !res.Umbenennungen[0].Bestaetigt {
		t.Errorf("Ergebnis nach Bestätigung: neu=%d abg=%d bestaetigt=%v", len(res.Neuzugaenge), len(res.Abgaenger), res.Umbenennungen[0].Bestaetigt)
	}
	if n := zaehle(t, pool, "lower(nachname) LIKE 'm%'"); n != 1 {
		t.Fatalf("zaehle(…) = %d, erwartet EINE Zeile für Anna", n)
	}
	var nachname, klasse, barcode string
	var abg bool
	var bestaetigt, eintrittDB *time.Time
	if err := pool.QueryRow(ctx, `SELECT nachname, klasse, barcode_id, ist_abgaenger, lusd_bestaetigt_am, schul_eintritt_am FROM schueler WHERE id=$1`, annaID).
		Scan(&nachname, &klasse, &barcode, &abg, &bestaetigt, &eintrittDB); err != nil {
		t.Fatal(err)
	}
	if nachname != "Mueller-Schmidt" || !lusd.IstKlasseGleich(klasse, "06F1") || barcode != "UMB-1" || abg || bestaetigt == nil || time.Since(*bestaetigt) > time.Minute {
		t.Errorf("Anna: nachname=%q klasse=%q barcode=%q abg=%v bestaetigt=%v", nachname, klasse, barcode, abg, bestaetigt)
	}
	// Der Import schreibt den Schuleintritt (Bestand und Neuanlage); ein Test, der die
	// Spalte per SQL sät, sähe das nicht.
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

	prev2, err := s.berechneLUSD(ctx, datei, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev2.Neuzugaenge)+len(prev2.Abgaenger)+len(prev2.Umbenennungen)+len(prev2.Klassenwechsel) != 0 {
		t.Errorf("zweiter Lauf muss leer sein: neu=%d abg=%d umb=%d wechsel=%d", len(prev2.Neuzugaenge), len(prev2.Abgaenger), len(prev2.Umbenennungen), len(prev2.Klassenwechsel))
	}
}

// Ohne Bestätigung bleibt es Abgänger + Neuanlage — mit Karenz-Sperre, nicht mit
// Anonymisierung. Und eine Wahl, die die Vorschau nie gemacht hat, wird abgewiesen.
func TestUmbenennung_UnbestaetigtBleibtAbgaengerMitKarenz(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	geb := testdaten.Datum(2013, 5, 4)

	annaID := testdaten.LegeLUSDSchuelerAn(t, pool, testdaten.LUSDSchuelerNeu{Vorname: "Anna", Nachname: "Müller", Klasse: "05F1", Barcode: "UMB-2", Geburtsdatum: geb})
	datei := lusd.Datei{Modus: lusd.ModusName, Zeilen: []lusd.Zeile{umbZeile(2, "Anna", "Schulz", "06F1", geb, nil)}}

	err := func() error {
		_, err := s.berechneLUSDLauf(ctx, datei, lusd.Lauf{Anwenden: true, MassenabgangBestaetigt: true,
			Umbenennungen: []lusd.UmbenennungWahl{{Zeile: 2, SchuelerID: "00000000-0000-0000-0000-000000000000"}}})
		return err
	}()
	if _, ok := err.(*lusd.UmbenennungUngueltigFehler); !ok {
		t.Fatalf("fremde Wahl muss abgewiesen werden, bekam %v", err)
	}
	if n := zaehle(t, pool, "nachname = 'Schulz'"); n != 0 {
		t.Fatal("die abgewiesene Wahl hat trotzdem geschrieben")
	}

	if _, err := s.berechneLUSD(ctx, datei, true, true); err != nil {
		t.Fatalf("Apply ohne Wahl: %v", err)
	}
	if n := zaehle(t, pool, "nachname IN ('Müller','Schulz')"); n != 2 {
		t.Fatalf("zaehle(…) = %d, erwartet Abgänger + Neuanlage (2 Zeilen)", n)
	}
	var vorname, grund string
	var abg, gesperrt, seit bool
	if err := pool.QueryRow(ctx, `SELECT vorname, ist_abgaenger, ist_gesperrt, COALESCE(block_reason,''), abgaenger_seit IS NOT NULL FROM schueler WHERE id=$1`, annaID).
		Scan(&vorname, &abg, &gesperrt, &grund, &seit); err != nil {
		t.Fatal(err)
	}
	if vorname != "Anna" || !abg || !gesperrt || grund != repository.AbgaengerSperrgrundKarenz || !seit {
		t.Errorf("Karenz-Abgänger falsch: vorname=%q abg=%v gesperrt=%v grund=%q seit=%v", vorname, abg, gesperrt, grund, seit)
	}
}

// Datumskorrektur der LUSD (Name gleich, Klasse benachbart, Datum anders): vermutliches
// Paar; bestätigt übernimmt der Datensatz das neue Geburtsdatum — sonst fände ihn der
// nächste Lauf wieder nicht.
func TestUmbenennung_DatumskorrekturUebernimmtGeburtsdatum(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	benID := testdaten.LegeLUSDSchuelerAn(t, pool, testdaten.LUSDSchuelerNeu{Vorname: "Ben", Nachname: "Meier", Klasse: "06B", Barcode: "UMB-3", Geburtsdatum: testdaten.Datum(2012, 1, 1)})
	datei := lusd.Datei{Modus: lusd.ModusName, Zeilen: []lusd.Zeile{umbZeile(2, "Ben", "Meier", "07B", testdaten.Datum(2012, 1, 10), nil)}}

	prev, err := s.berechneLUSD(ctx, datei, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Umbenennungen) != 1 || prev.Umbenennungen[0].Sicher || prev.Umbenennungen[0].NeuGeburtsdatum != "2012-01-10" {
		t.Fatalf("Vorschau.Umbenennungen = %+v, erwartet ein vermutliches Paar mit neuem Datum", prev.Umbenennungen)
	}
	if _, err := s.berechneLUSDLauf(ctx, datei, lusd.Lauf{Anwenden: true, MassenabgangBestaetigt: true,
		Umbenennungen: []lusd.UmbenennungWahl{{Zeile: 2, SchuelerID: benID}}}); err != nil {
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

// Ein Abgänger aus einem früheren Lauf (gesperrt, in der Karenz) taucht umbenannt
// wieder auf: Paar mit war_abgaenger; bestätigt wird er reaktiviert — entsperrt, ohne
// Abgänger-Stempel, mit neuem Namen.
func TestUmbenennung_FruehererAbgaengerWirdReaktiviert(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	geb, eintritt := testdaten.Datum(2011, 3, 3), testdaten.Datum(2022, 8, 22)

	altID := testdaten.LegeLUSDSchuelerAn(t, pool, testdaten.LUSDSchuelerNeu{Vorname: "Cem", Nachname: "Yilmaz", Klasse: "08C", Barcode: "UMB-4", Geburtsdatum: geb, SchulEintrittAm: eintritt, IstAbgaenger: true})
	datei := lusd.Datei{Modus: lusd.ModusName, Zeilen: []lusd.Zeile{umbZeile(2, "Cem", "Yılmaz-Kaya", "09C", geb, eintritt)}}

	prev, err := s.berechneLUSD(ctx, datei, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Umbenennungen) != 1 || !prev.Umbenennungen[0].WarAbgaenger || !prev.Umbenennungen[0].Sicher {
		t.Fatalf("Vorschau.Umbenennungen = %+v, erwartet das sichere Paar mit dem früheren Abgänger", prev.Umbenennungen)
	}
	if _, err := s.berechneLUSDLauf(ctx, datei, lusd.Lauf{Anwenden: true, MassenabgangBestaetigt: true,
		Umbenennungen: []lusd.UmbenennungWahl{{Zeile: 2, SchuelerID: altID}}}); err != nil {
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
