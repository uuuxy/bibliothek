package repository_test

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/lusd"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/testweg"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Zusammenführen zweier Datensätze — das Sicherheitsnetz hinter der Vorschau-Paarung,
// am echten Postgres: Vorgänge wandern, die Quelle verschwindet, das Ziel trägt die
// LUSD-frischen Stammdaten, Sperre und Abgänger-Stempel werden bereinigt, und die
// Protokollspuren zeigen auf den verbliebenen Datensatz.

func zfZaehle(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestZusammenfuehren_QuelleGehtImZielAuf(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	// Ziel: der alte Datensatz mit Ausweis — vom letzten Jahr bestätigt, jetzt Abgänger in
	// der Karenz (der Export hat ihn unter neuem Namen nicht wiedererkannt).
	ziel := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Alt", Nachname: "Name", Klasse: "07A", Barcode: "ZF-1", Geburtsdatum: testdaten.Datum(2012, 3, 3), IstAbgaenger: true})
	// Quelle: der frisch importierte Datensatz — von der LUSD soeben bestätigt, mit
	// Anschrift und Klasse, und mit einer offenen Ausleihe, die schon an ihm hängt.
	quelle := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Neu", Nachname: "Name", Klasse: "08A", Barcode: "ZF-2", Geburtsdatum: testdaten.Datum(2012, 3, 3)})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET lusd_bestaetigt_am = NOW(), strasse = 'Neuweg', plz = '61381', ort = 'Friedrichsdorf' WHERE id = $1`, quelle); err != nil {
		t.Fatal(err)
	}
	testdaten.LegeOffeneAusleiheAn(t, pool, quelle, "ZFQ")
	// Protokollspur der Quelle (Lesehistorie) — muss danach auf das Ziel zeigen.
	if _, err := pool.Exec(ctx, `INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details)
		VALUES ('ausleihen', 'CREATE', gen_random_uuid(), 'USER', jsonb_build_object('schueler_id', $1::text, 'entleiher', 'Neu Name'))`, quelle); err != nil {
		t.Fatal(err)
	}

	erg, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle})
	if err != nil {
		t.Fatalf("Zusammenführen: %v", err)
	}
	if erg.ZielID != ziel || erg.BarcodeID != "ZF-1" || erg.QuelleBarcode != "ZF-2" || erg.Ausleihen != 1 || erg.Nachname != "Name" || erg.Vorname != "Neu" {
		t.Errorf("Ergebnis falsch: %+v", erg)
	}

	if n := zfZaehle(t, pool, `SELECT count(*) FROM schueler WHERE id = $1`, quelle); n != 0 {
		t.Error("die Quelle existiert noch")
	}
	var vorname, klasse, strasse, grund string
	var abg, gesperrt, seit bool
	if err := pool.QueryRow(ctx, `SELECT vorname, klasse, COALESCE(strasse,''), ist_abgaenger, ist_gesperrt, COALESCE(block_reason,''), abgaenger_seit IS NOT NULL
		FROM schueler WHERE id = $1`, ziel).Scan(&vorname, &klasse, &strasse, &abg, &gesperrt, &grund, &seit); err != nil {
		t.Fatal(err)
	}
	// LUSD führend: Name, Klasse und Anschrift der frischer bestätigten Quelle.
	if vorname != "Neu" || !lusd.IstKlasseGleich(klasse, "08A") || strasse != "Neuweg" {
		t.Errorf("Stammdaten nicht von der LUSD-frischen Seite: vorname=%q klasse=%q strasse=%q", vorname, klasse, strasse)
	}
	// Kein Abgänger mehr; die Sperre bleibt wegen der offenen Ausleihe — mit sachlichem Grund.
	if abg || seit || !gesperrt || grund != repository.AbgaengerSperrgrundOffen {
		t.Errorf("Status falsch: abg=%v seit=%v gesperrt=%v grund=%q", abg, seit, gesperrt, grund)
	}
	if n := zfZaehle(t, pool, `SELECT count(*) FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL`, ziel); n != 1 {
		t.Errorf("offene Ausleihe nicht gewandert (n=%d)", n)
	}
	if n := zfZaehle(t, pool, `SELECT count(*) FROM audit_log WHERE tabelle='ausleihen' AND details->>'schueler_id' = $1`, ziel); n != 1 {
		t.Errorf("Protokollspur zeigt nicht auf das Ziel (n=%d)", n)
	}
}

// Ohne offene Vorgänge fällt die automatische Abgänger-Sperre des Ziels; eine manuelle
// Sperre bliebe (anderer Grund) — hier der automatische Fall.
func TestZusammenfuehren_HebtAbgaengerSperreOhneVorgaengeAuf(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Alt", Nachname: "Frei", Klasse: "07A", Barcode: "ZF-3", Geburtsdatum: testdaten.Datum(2012, 4, 4), IstAbgaenger: true})
	quelle := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Alt", Nachname: "Frei-Neu", Klasse: "08A", Barcode: "ZF-4", Geburtsdatum: testdaten.Datum(2012, 4, 4)})

	if _, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle}); err != nil {
		t.Fatal(err)
	}
	var gesperrt bool
	var grund *string
	if err := pool.QueryRow(ctx, `SELECT ist_gesperrt, block_reason FROM schueler WHERE id = $1`, ziel).Scan(&gesperrt, &grund); err != nil {
		t.Fatal(err)
	}
	if gesperrt || grund != nil {
		t.Errorf("Ziel ohne Vorgänge muss entsperrt sein: gesperrt=%v grund=%v", gesperrt, grund)
	}
}

func TestZusammenfuehren_Abweisungen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	a := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "A", Nachname: "Eins", Klasse: "07A", Barcode: "ZF-5", Geburtsdatum: testdaten.Datum(2012, 5, 5)})
	anon := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Abgänger", Nachname: "Anonymisiert-x", Klasse: "ABG", Barcode: "ANON-x"})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET anonymized_at = NOW() WHERE id = $1`, anon); err != nil {
		t.Fatal(err)
	}

	if _, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: a, QuelleID: a}); !errors.Is(err, repository.ErrZusammenfuehrenGleich) {
		t.Errorf("gleicher Datensatz: %v", err)
	}
	if _, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: a, QuelleID: anon}); !errors.Is(err, repository.ErrZusammenfuehrenAnonymisiert) {
		t.Errorf("anonymisierte Quelle: %v", err)
	}
	if _, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: a, QuelleID: "00000000-0000-0000-0000-000000000000"}); !errors.Is(err, repository.ErrZusammenfuehrenNichtGefunden) {
		t.Errorf("unbekannte Quelle: %v", err)
	}
	if n := zfZaehle(t, pool, `SELECT count(*) FROM schueler WHERE deleted_at IS NULL`); n != 2 {
		t.Errorf("Abweisungen dürfen nichts verändern (n=%d)", n)
	}
}

// Die Kandidatensuche findet Abgänger und Gesperrte, nie den Ausgangsdatensatz selbst
// und nie Anonymisierte.
func TestZusammenfuehren_KandidatenSucheSiehtAbgaenger(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	ich := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Mira", Nachname: "Suchwort", Klasse: "07A", Barcode: "ZF-6", Geburtsdatum: testdaten.Datum(2012, 6, 6)})
	testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Mira", Nachname: "Suchwort-Alt", Klasse: "06A", Barcode: "ZF-7", Geburtsdatum: testdaten.Datum(2012, 6, 6), IstAbgaenger: true})
	anon := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Mira", Nachname: "Suchwort-Anon", Klasse: "ABG", Barcode: "ZF-8"})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET anonymized_at = NOW() WHERE id = $1`, anon); err != nil {
		t.Fatal(err)
	}

	treffer, err := repository.FindeZusammenfuehrenKandidaten(ctx, pool, ich, "Suchwort", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(treffer) != 1 || treffer[0].Nachname != "Suchwort-Alt" || !treffer[0].IstAbgaenger || !treffer[0].IstGesperrt {
		t.Errorf("FindeZusammenfuehrenKandidaten(…) = %+v, erwartet genau den gesperrten Abgänger", treffer)
	}
}

// Gate über alle Tabellen, die an einem Schüler hängen: Der erste Test säte nur die
// Ausleihe (FK RESTRICT — ein ausgelassener Schritt bräche laut). Vormerkungen und Foto
// hängen per CASCADE, der Bescheid per SET NULL: Fällt ihr UPDATE weg, nimmt der DELETE
// der Quelle sie still mit bzw. lässt sie personenlos zurück.
//
// Die Tabellen kommen aus testdaten.DSGVOLeserQuellen, nicht aus einer Abschrift: Eine
// eigene Liste bleibt stehen, wenn eine Migration eine neue Tabelle in die echte Liste
// bringt, wie Migration 110 die Bescheide.
// Das Zusammenführen ließe den Bescheid der Quelle dann ohne Schüler
// zurück: weg aus der Akte, weg aus der Art.-15-Auskunft, und die Tilgung (sie sucht
// über schueler_id) fände Name und Anschrift im Snapshot nie mehr. Eine neue
// Quelle ohne Fall im switch macht diesen Test rot.
func TestZusammenfuehren_JedeTabelleWandert(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Ziel", Nachname: "Tabelle", Klasse: "07A", Barcode: "ZF-T1", Geburtsdatum: testdaten.Datum(2012, 7, 7)})
	quelle := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Quelle", Nachname: "Tabelle", Klasse: "07A", Barcode: "ZF-T2", Geburtsdatum: testdaten.Datum(2012, 7, 7)})

	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql[:40], err)
		}
	}
	type zaehlung struct{ was, sql string }
	var zaehlungen []zaehlung
	for _, q := range testdaten.DSGVOLeserQuellen {
		switch q.Tabelle {
		case "leser":
			// Die Zeile selbst wandert nicht — sie geht im Ziel auf.
		case "ausleihen":
			testdaten.LegeOffeneAusleiheAn(t, pool, quelle, "ZFT")
			zaehlungen = append(zaehlungen, zaehlung{"ausleihen", `SELECT count(*) FROM ausleihen WHERE schueler_id = $1`})
		case "schadensfaelle":
			testdaten.LegeOffenenSchadenAn(t, pool, quelle, "ZFT")
			zaehlungen = append(zaehlungen, zaehlung{"schadensfaelle", `SELECT count(*) FROM schadensfaelle WHERE schueler_id = $1`})
		case "vormerkungen":
			titelID := testdaten.LegeTitelAn(t, pool, "Vormerktitel-ZFT", 1)
			exec(`INSERT INTO vormerkungen (titel_id, schueler_id) VALUES ($1, $2)`, titelID, quelle)
			zaehlungen = append(zaehlungen, zaehlung{"vormerkungen", `SELECT count(*) FROM vormerkungen WHERE schueler_id = $1`})
		case "schueler_fotos":
			exec(`INSERT INTO schueler_fotos (schueler_id, foto_encrypted) VALUES ($1, '\x00'::bytea)`, quelle)
			zaehlungen = append(zaehlungen, zaehlung{"schueler_fotos", `SELECT count(*) FROM schueler_fotos WHERE schueler_id = $1`})
		case "schadensersatz_bescheide":
			exec(`INSERT INTO schadensersatz_bescheide
					(schueler_id, mittel, kassenjahr, laufende_nr, referenznummer, frist_bis, gesamtbetrag, empfaenger_snapshot)
				VALUES ($1, 'land', 2026, 1, '5830 2026 1234 0001', CURRENT_DATE + 28, 24.90,
					jsonb_build_object('name', 'Quelle Tabelle'))`, quelle)
			zaehlungen = append(zaehlungen, zaehlung{"schadensersatz_bescheide", `SELECT count(*) FROM schadensersatz_bescheide WHERE schueler_id = $1`})
		case "audit_log":
			exec(`INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details)
				VALUES ('ausleihen', 'CREATE', gen_random_uuid(), 'USER', jsonb_build_object('schueler_id', $1::text))`, quelle)
			exec(`INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details)
				VALUES ('schueler', 'UPDATE', $1::uuid, 'USER', '{"feld":"klasse"}'::jsonb)`, quelle)
			// Die Spur einer Forderung und einer Vormerkung, die mit ihrem Titel gelöscht
			// wurden, trägt den Leser ebenfalls in den Details (protokolliereOffeneForderungen,
			// ProtokolliereWartendeBezuege). Bliebe sie an der Quelle, fände die Tilgung des
			// Ziels den Namen darin nicht.
			exec(`INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details)
				VALUES ('schadensfaelle', 'DELETE', gen_random_uuid(), 'SYSTEM',
					jsonb_build_object('schueler_id', $1::text, 'schuldner', 'Quelle Tabelle'))`, quelle)
			exec(`INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details)
				VALUES ('vormerkungen', 'DELETE', gen_random_uuid(), 'SYSTEM',
					jsonb_build_object('schueler_id', $1::text, 'betrifft', 'Quelle Tabelle'))`, quelle)
			zaehlungen = append(zaehlungen,
				zaehlung{"audit_log Lesehistorie", `SELECT count(*) FROM audit_log WHERE tabelle = 'ausleihen' AND details->>'schueler_id' = $1`},
				zaehlung{"audit_log Datensatz-Historie", `SELECT count(*) FROM audit_log WHERE tabelle = 'schueler' AND aktion = 'UPDATE' AND datensatz_id = $1::uuid`},
				zaehlung{"audit_log Spur einer gelöschten Forderung", `SELECT count(*) FROM audit_log WHERE tabelle = 'schadensfaelle' AND details->>'schueler_id' = $1`},
				zaehlung{"audit_log Spur einer gelöschten Vormerkung", `SELECT count(*) FROM audit_log WHERE tabelle = 'vormerkungen' AND details->>'schueler_id' = $1`})
		case "audit_logs":
			exec(`INSERT INTO audit_logs (aktion, details) VALUES ('LUSD_ID_NACHGETRAGEN', jsonb_build_object('schueler_id', $1::text))`, quelle)
			zaehlungen = append(zaehlungen, zaehlung{"audit_logs", `SELECT count(*) FROM audit_logs WHERE aktion = 'LUSD_ID_NACHGETRAGEN' AND details->>'schueler_id' = $1`})
		case "nachbuch_meldungen":
			// Migration 117: die Quelle stand als Vorbesitzer in einer Meldung der Theke.
			exec(`INSERT INTO nachbuch_meldungen (idempotency_key, barcode, ergebnis, grund, vorbesitzer_schueler_id, gescannt_am)
				VALUES (gen_random_uuid(), 'B-ZFT-NB', 'umgebucht', 'Zusammenführen-Gate', $1, NOW())`, quelle)
			zaehlungen = append(zaehlungen, zaehlung{"nachbuch_meldungen", `SELECT count(*) FROM nachbuch_meldungen WHERE ausleiher_schueler_id = $1 OR vorbesitzer_schueler_id = $1`})
		case "benutzer":
			// Migration 123: Auf die Quelle zeigt ein Zugangskonto. Nach dem
			// Zusammenführen muss es auf das Ziel zeigen — sonst macht der DELETE der
			// Quelle die Anmeldung still leserlos, und die Person sieht ihre Bücher nicht
			// mehr. (Beide Seiten mit eigenem Konto bricht bewusst ab; das prüft
			// TestZusammenfuehren_ZweiKontenBrichtAb.)
			exec(`INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, leser_id)
				VALUES ('Zft', 'Konto', 'zft-konto@test.invalid', 'kollegium', true, $1::uuid)`, quelle)
			zaehlungen = append(zaehlungen, zaehlung{"benutzer", `SELECT count(*) FROM benutzer WHERE leser_id = $1`})
		default:
			t.Fatalf("testdaten.DSGVOLeserQuellen kennt %s (%s), das Zusammenführen-Gate nicht — "+
				"verschiebeVorgaenge und diesen Test nachziehen", q.Tabelle, q.Bezug)
		}
	}

	if _, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle}); err != nil {
		t.Fatalf("Zusammenführen: %v", err)
	}

	for _, z := range zaehlungen {
		if n := zfZaehle(t, pool, z.sql, ziel); n != 1 {
			t.Errorf("%s = %d, erwartet 1 Zeile am Ziel", z.was, n)
		}
	}
	// Nichts hängt mehr an der Quelle — weder als FK noch als Protokollschlüssel.
	for _, z := range zaehlungen {
		if n := zfZaehle(t, pool, z.sql, quelle); n != 0 {
			t.Errorf("%s: Quelle hat noch %d Zeile(n)", z.was, n)
		}
	}
}

// Rückweg (Rasterfrage 10): Ein falsch bestätigtes Paar muss sich wieder trennen lassen.
// Der Eintrag in audit_log (tabelle='schueler', datensatz_id=Ziel, in der Transaktion)
// trägt die Stammdaten der Quelle, den Stand des Ziels davor und die Kennungen der
// gewanderten Zeilen — dieser Test geht den Rückweg am Ergebnis: Quelle aus dem Eintrag
// neu anlegen, Ausleihe zurückschlüsseln, Ziel auf den alten Stand setzen.
func TestZusammenfuehren_RueckwegAusDemProtokoll(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Alt", Nachname: "Rueckweg", Klasse: "07A", Barcode: "ZF-R1", Geburtsdatum: testdaten.Datum(2012, 8, 8), IstAbgaenger: true})
	quelle := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Falsch", Nachname: "Rueckweg", Klasse: "08B", Barcode: "ZF-R2", Geburtsdatum: testdaten.Datum(2012, 8, 8)})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET lusd_bestaetigt_am = NOW(), strasse = 'Irrweg', plz = '61381', ort = 'Friedrichsdorf' WHERE id = $1`, quelle); err != nil {
		t.Fatal(err)
	}
	testdaten.LegeOffeneAusleiheAn(t, pool, quelle, "ZFR")

	if _, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle}); err != nil {
		t.Fatal(err)
	}

	// Der Eintrag: in der Transaktion, am Ziel, mit allem, was der Rückweg braucht.
	var details map[string]any
	if err := pool.QueryRow(ctx, `SELECT details FROM audit_log WHERE tabelle = 'schueler' AND aktion = 'ZUSAMMENGEFUEHRT' AND datensatz_id = $1::uuid`, ziel).Scan(&details); err != nil {
		t.Fatalf("Rückweg-Eintrag fehlt: %v", err)
	}
	teil := func(k string) map[string]any {
		t.Helper()
		m, ok := details[k].(map[string]any)
		if !ok {
			t.Fatalf("Eintrag ohne %q: %+v", k, details)
		}
		return m
	}
	q, vorher, gewandert := teil("quelle"), teil("ziel_vorher"), teil("gewandert")
	ausleihen, ok := gewandert["ausleihen"].([]any)
	if !ok {
		t.Fatalf("gewandert.ausleihen fehlt: %+v", gewandert)
	}
	if q["id"] != quelle || q["barcode_id"] != "ZF-R2" || q["vorname"] != "Falsch" || q["strasse"] != "Irrweg" || q["geburtsdatum"] != "2012-08-08" {
		t.Errorf("Quelle unvollständig im Eintrag: %+v", q)
	}
	if vorher["vorname"] != "Alt" || vorher["ist_abgaenger"] != true || vorher["block_reason"] == nil {
		t.Errorf("Ziel-Stand davor unvollständig: %+v", vorher)
	}
	if len(ausleihen) != 1 {
		t.Fatalf("gewanderte Ausleihe fehlt: %+v", gewandert)
	}

	// Der Rückweg von Hand, allein aus dem Eintrag — erst das Ziel zurück (sonst
	// kollidiert Name + Geburtsdatum am Unique-Index), dann die Quelle neu, dann die Zeilen.
	if _, err := pool.Exec(ctx, `UPDATE schueler SET vorname = $2, nachname = $3, klasse = $4, ist_abgaenger = true, ist_gesperrt = true, block_reason = $5 WHERE id = $1`,
		ziel, vorher["vorname"], vorher["nachname"], vorher["klasse"], vorher["block_reason"]); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO schueler (id, barcode_id, vorname, nachname, klasse, geburtsdatum, abgaenger_jahr, strasse, plz, ort)
		VALUES ($1, $2, $3, $4, $5, $6::date, 2031, $7, $8, $9)`,
		q["id"], q["barcode_id"], q["vorname"], q["nachname"], q["klasse"], q["geburtsdatum"], q["strasse"], q["plz"], q["ort"]); err != nil {
		t.Fatalf("Quelle aus dem Eintrag neu anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET schueler_id = $1 WHERE id = $2::uuid`, quelle, ausleihen[0]); err != nil {
		t.Fatal(err)
	}
	if n := zfZaehle(t, pool, `SELECT count(*) FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL`, quelle); n != 1 {
		t.Errorf("Ausleihe nach dem Rückweg nicht bei der Quelle (n=%d)", n)
	}
	if n := zfZaehle(t, pool, `SELECT count(*) FROM schueler WHERE id = $1 AND vorname = 'Alt' AND ist_abgaenger`, ziel); n != 1 {
		t.Error("Ziel nicht auf den alten Stand zurück")
	}
}

// Eine manuelle Sperre der Quelle (Ausweis verloren, Hausverbot …) darf beim Zusammen-
// führen nicht still verschwinden — sie gehört zur Person, nicht zum Datensatz.
// Das Ziel trägt sie danach mit demselben Grund.
func TestZusammenfuehren_ManuelleSperreDerQuelleBleibt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Ziel", Nachname: "Sperre", Klasse: "07A", Barcode: "ZF-M1", Geburtsdatum: testdaten.Datum(2012, 9, 9)})
	quelle := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Quelle", Nachname: "Sperre", Klasse: "07A", Barcode: "ZF-M2", Geburtsdatum: testdaten.Datum(2012, 9, 9)})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET ist_gesperrt = true, is_manually_blocked = true, block_reason = 'Manuell: Ausweis als gestohlen gemeldet' WHERE id = $1`, quelle); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle}); err != nil {
		t.Fatal(err)
	}
	var gesperrt, manuell bool
	var grund string
	if err := pool.QueryRow(ctx, `SELECT ist_gesperrt, COALESCE(is_manually_blocked, false), COALESCE(block_reason, '') FROM schueler WHERE id = $1`, ziel).Scan(&gesperrt, &manuell, &grund); err != nil {
		t.Fatal(err)
	}
	if !gesperrt || !manuell || grund != "Manuell: Ausweis als gestohlen gemeldet" {
		t.Errorf("manuelle Sperre der Quelle verloren: gesperrt=%v manuell=%v grund=%q", gesperrt, manuell, grund)
	}
}

// Führend ist der Datensatz, den die LUSD zuletzt bestätigt hat — aber ein aktiver
// Datensatz schlägt einen Abgänger, auch wenn er nie bestätigt wurde: Das Kind kam nach
// der Umbenennung an die Theke, wurde von Hand neu angelegt (Handanlage, aktuelle Klasse),
// und der Admin behält den alten Datensatz wegen des Ausweises. Gewännen
// dann Name und Klasse des Abgängers, spaltete der nächste Namensmodus-Import das
// Kind erneut.
func TestZusammenfuehren_AktiveHandanlageSchlaegtAbgaenger(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Alt", Nachname: "Fuehrend", Klasse: "07A", Barcode: "ZF-F1", Geburtsdatum: testdaten.Datum(2012, 10, 10), IstAbgaenger: true})
	quelle := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Neu", Nachname: "Fuehrend-Neu", Klasse: "08A", Barcode: "ZF-F2", Geburtsdatum: testdaten.Datum(2012, 10, 10)})
	// Die Handanlage wurde nie von der LUSD bestätigt.
	if _, err := pool.Exec(ctx, `UPDATE schueler SET lusd_bestaetigt_am = NULL WHERE id = $1`, quelle); err != nil {
		t.Fatal(err)
	}
	erg, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle})
	if err != nil {
		t.Fatal(err)
	}
	if erg.Nachname != "Fuehrend-Neu" || !lusd.IstKlasseGleich(erg.Klasse, "08A") {
		t.Errorf("aktive Handanlage muss die Stammdaten stellen, nicht der Abgänger: %+v", erg)
	}
}

// Vormerkungs-Dublette: Beide haben denselben Titel vorgemerkt, die Quelle ist schon
// „abholbereit" (Exemplar liegt bereit), das Ziel noch „wartend". Fiele
// die Vormerkung der Quelle, verlöre das Kind den bereitgestellten Platz, das Exemplar
// stünde frei, und niemand würde bedient. Es bleibt die weiter fortgeschrittene Vormerkung,
// egal auf welcher Seite sie steht.
func TestZusammenfuehren_FortgeschritteneVormerkungBleibt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Ziel", Nachname: "Vormerk", Klasse: "07A", Barcode: "ZF-V1", Geburtsdatum: testdaten.Datum(2012, 11, 11)})
	quelle := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Quelle", Nachname: "Vormerk", Klasse: "07A", Barcode: "ZF-V2", Geburtsdatum: testdaten.Datum(2012, 11, 11)})
	titelID := testdaten.LegeTitelAn(t, pool, "Vormerktitel-ZFV", 1)
	exID := testdaten.LegeExemplarAn(t, pool, titelID, "EX-ZFV", true, "")
	if _, err := pool.Exec(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status) VALUES ($1, $2, 'wartend')`, titelID, ziel); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis)
		VALUES ($1, $2, 'abholbereit', $3, CURRENT_TIMESTAMP + INTERVAL '2 days')`, titelID, quelle, exID); err != nil {
		t.Fatal(err)
	}
	erg, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle})
	if err != nil {
		t.Fatal(err)
	}
	var status string
	var exemplarID *string
	if err := pool.QueryRow(ctx, `SELECT status, bereitgestellt_exemplar_id::text FROM vormerkungen WHERE schueler_id = $1 AND titel_id = $2`, ziel, titelID).Scan(&status, &exemplarID); err != nil {
		t.Fatal(err)
	}
	if status != "abholbereit" || exemplarID == nil || *exemplarID != exID {
		t.Errorf("die abholbereite Vormerkung muss überleben: status=%q exemplar=%v (Ergebnis %+v)", status, exemplarID, erg)
	}
	if n := zfZaehle(t, pool, `SELECT count(*) FROM vormerkungen WHERE titel_id = $1`, titelID); n != 1 {
		t.Errorf("Vormerkungen des Titels = %d, erwartet genau eine", n)
	}
}

// Lag für beide Datensätze schon je ein Exemplar desselben Titels im Abholfach, bleibt die
// ältere Vormerkung. Das Exemplar der anderen ist danach frei und geht an den Nächsten in der
// Warteschlange wie beim Löschen einer Vormerkung; sonst stünde es im Regal, während er wartet.
func TestZusammenfuehren_FreiesAbholfachGehtAnDenNaechsten(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Ziel", Nachname: "Abholfach", Klasse: "07A", Barcode: "ZF-A1", Geburtsdatum: testdaten.Datum(2012, 9, 9)})
	quelle := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Quelle", Nachname: "Abholfach", Klasse: "07A", Barcode: "ZF-A2", Geburtsdatum: testdaten.Datum(2012, 9, 9)})
	dritte := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Dritte", Nachname: "Wartet", Klasse: "07B", Barcode: "ZF-A3", Geburtsdatum: testdaten.Datum(2012, 8, 8)})
	titelID := testdaten.LegeTitelAn(t, pool, "Abholfach-ZFA", 1)
	exZiel := testdaten.LegeExemplarAn(t, pool, titelID, "EX-ZFA-1", true, "")
	exQuelle := testdaten.LegeExemplarAn(t, pool, titelID, "EX-ZFA-2", true, "")
	for _, v := range []struct{ leser, status, exemplar, alter string }{
		{ziel, "abholbereit", exZiel, "2 days"},
		{quelle, "abholbereit", exQuelle, "1 day"},
		{dritte, "wartend", "", "1 hour"},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis, erstellt_am)
			VALUES ($1, $2, $3::text, NULLIF($4, '')::uuid,
			        CASE WHEN $3::text = 'abholbereit' THEN now() + interval '3 days' END, now() - $5::interval)`,
			titelID, v.leser, v.status, v.exemplar, v.alter); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle}); err != nil {
		t.Fatal(err)
	}
	for _, w := range []struct{ name, leser, exemplar string }{{"Ziel", ziel, exZiel}, {"Dritte", dritte, exQuelle}} {
		var status string
		var exemplar *string
		if err := pool.QueryRow(ctx, `SELECT status, bereitgestellt_exemplar_id::text FROM vormerkungen WHERE schueler_id = $1`,
			w.leser).Scan(&status, &exemplar); err != nil {
			t.Fatalf("Vormerkung von %s lesen: %v", w.name, err)
		}
		if status != "abholbereit" || exemplar == nil || *exemplar != w.exemplar {
			t.Errorf("%s: %s mit Exemplar %v, erwartet abholbereit mit %s", w.name, status, exemplar, w.exemplar)
		}
	}
}

// Foto: kommt nie aus der LUSD — das jüngere gewinnt, egal auf welcher Seite. Bliebe
// immer das Foto des Ziels, stürbe das jüngere der Quelle im CASCADE.
func TestZusammenfuehren_JuengeresFotoGewinnt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Ziel", Nachname: "Foto", Klasse: "07A", Barcode: "ZF-P1", Geburtsdatum: testdaten.Datum(2012, 12, 12)})
	quelle := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Quelle", Nachname: "Foto", Klasse: "07A", Barcode: "ZF-P2", Geburtsdatum: testdaten.Datum(2012, 12, 12)})
	for _, f := range []struct {
		id, bytes, alter string
	}{{ziel, `\x01`, "2 years"}, {quelle, `\x02`, "1 day"}} {
		if _, err := pool.Exec(ctx, `INSERT INTO schueler_fotos (schueler_id, foto_encrypted, aktualisiert_am) VALUES ($1, $2::bytea, NOW() - $3::interval)`, f.id, f.bytes, f.alter); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle}); err != nil {
		t.Fatal(err)
	}
	var foto []byte
	if err := pool.QueryRow(ctx, `SELECT foto_encrypted FROM schueler_fotos WHERE schueler_id = $1`, ziel).Scan(&foto); err != nil {
		t.Fatal(err)
	}
	if len(foto) != 1 || foto[0] != 0x02 {
		t.Errorf("das jüngere Foto der Quelle muss am Ziel hängen, gefunden %x", foto)
	}
	if n := zfZaehle(t, pool, `SELECT count(*) FROM schueler_fotos`); n != 1 {
		t.Errorf("Fotos = %d, erwartet genau eins", n)
	}
}

// TestZusammenfuehren_ZweiKontenBrichtAb: Haben beide Datensätze ein eigenes
// Zugangskonto, wird nicht geraten.
//
// uniq_benutzer_leser verbietet zwei Anmeldungen auf einer Leserzeile.
// Ohne die Vorprüfung liefe das Zusammenführen in eine Constraint-Meldung, die an der
// Oberfläche als „Fehler beim Zusammenführen" ankäme; mit ihr steht dort, was zu tun ist.
// Vor allem aber: Welche der beiden Anmeldungen bleiben soll, ist eine Entscheidung über
// zwei Menschen — die trifft kein UPDATE.
func TestZusammenfuehren_ZweiKontenBrichtAb(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	ziel := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Ziel", Nachname: "Konto", Klasse: "07A", Barcode: "ZK-1", Geburtsdatum: testdaten.Datum(2012, 5, 5)})
	quelle := testweg.LegeLUSDSchuelerAn(t, pool, testweg.LUSDSchuelerNeu{Vorname: "Quelle", Nachname: "Konto", Klasse: "08A", Barcode: "ZK-2", Geburtsdatum: testdaten.Datum(2012, 5, 5)})

	for i, leser := range []string{ziel, quelle} {
		if _, err := pool.Exec(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, leser_id)
			VALUES ('Zwei', 'Konten', 'zwei-konten-' || $1::uuid::text || '@test.invalid', 'kollegium', true, $1::uuid)`,
			leser); err != nil {
			t.Fatalf("Konto %d anlegen: %v", i, err)
		}
	}

	_, err := repository.FuehreLeserZusammen(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle})
	if err == nil {
		t.Fatal("das Zusammenführen ging durch — eine der beiden Anmeldungen wäre verloren")
	}
	if !errors.Is(err, repository.ErrZusammenfuehrenZweiKonten) {
		t.Errorf("Abbruch kam, aber nicht als die benannte Abweisung, an der die Route den Grund erkennt: %v", err)
	}

	// Und es ist wirklich nichts passiert: Die Quelle steht noch, beide Konten hängen noch.
	if n := zfZaehle(t, pool, `SELECT count(*) FROM schueler WHERE id = $1`, quelle); n != 1 {
		t.Error("die Quelle ist trotz Abbruch verschwunden — die Transaktion lief nicht zurück")
	}
	if n := zfZaehle(t, pool, `SELECT count(*) FROM benutzer WHERE leser_id IN ($1, $1)`, quelle); n != 1 {
		t.Errorf("das Konto der Quelle hängt nicht mehr an ihr (%d)", n)
	}
}
