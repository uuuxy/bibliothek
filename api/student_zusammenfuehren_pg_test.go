package api

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Zusammenführen zweier Datensätze — das Sicherheitsnetz hinter der Vorschau-Paarung,
// am echten Postgres: Vorgänge wandern, die Quelle verschwindet, das Ziel trägt die
// LUSD-frischen Stammdaten, Sperre und Abgänger-Stempel werden bereinigt, und die
// Protokollspuren zeigen auf den verbliebenen Datensatz.

func zfAuftrag(ziel, quelle string) repository.ZusammenfuehrenAuftrag {
	return repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle, AbgaengerJahr: calculateAbgaengerJahr}
}

func zfZaehle(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestZusammenfuehren_QuelleGehtImZielAuf(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Ziel: der alte Datensatz mit Ausweis — vom letzten Jahr bestätigt, jetzt Abgänger in
	// der Karenz (der Export hat ihn unter neuem Namen nicht wiedererkannt).
	ziel := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Alt", nachname: "Name", klasse: "07A", barcode: "ZF-1", geb: datum(2012, 3, 3), abgaenger: true})
	// Quelle: der frisch importierte Datensatz — von der LUSD soeben bestätigt, mit
	// Anschrift und Klasse, und mit einer offenen Ausleihe, die schon an ihm hängt.
	quelle := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Neu", nachname: "Name", klasse: "08A", barcode: "ZF-2", geb: datum(2012, 3, 3)})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET lusd_bestaetigt_am = NOW(), strasse = 'Neuweg', plz = '61381', ort = 'Friedrichsdorf' WHERE id = $1`, quelle); err != nil {
		t.Fatal(err)
	}
	seedOffeneAusleihe(t, pool, quelle, "ZFQ")
	// Protokollspur der Quelle (Lesehistorie) — muss danach auf das Ziel zeigen.
	if _, err := pool.Exec(ctx, `INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details)
		VALUES ('ausleihen', 'CREATE', gen_random_uuid(), 'USER', jsonb_build_object('schueler_id', $1::text, 'entleiher', 'Neu Name'))`, quelle); err != nil {
		t.Fatal(err)
	}

	erg, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, quelle))
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
	if vorname != "Neu" || !klassenGleich(klasse, "08A") || strasse != "Neuweg" {
		t.Errorf("Stammdaten nicht von der LUSD-frischen Seite: vorname=%q klasse=%q strasse=%q", vorname, klasse, strasse)
	}
	// Kein Abgänger mehr; die Sperre bleibt wegen der offenen Ausleihe — mit sachlichem Grund.
	if abg || seit || !gesperrt || grund != "Sperre wegen offener Vorgänge" {
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
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Alt", nachname: "Frei", klasse: "07A", barcode: "ZF-3", geb: datum(2012, 4, 4), abgaenger: true})
	quelle := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Alt", nachname: "Frei-Neu", klasse: "08A", barcode: "ZF-4", geb: datum(2012, 4, 4)})

	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, quelle)); err != nil {
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
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	a := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "A", nachname: "Eins", klasse: "07A", barcode: "ZF-5", geb: datum(2012, 5, 5)})
	anon := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Abgänger", nachname: "Anonymisiert-x", klasse: "ABG", barcode: "ANON-x"})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET anonymized_at = NOW() WHERE id = $1`, anon); err != nil {
		t.Fatal(err)
	}

	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(a, a)); !errors.Is(err, repository.ErrZusammenfuehrenGleich) {
		t.Errorf("gleicher Datensatz: %v", err)
	}
	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(a, anon)); !errors.Is(err, repository.ErrZusammenfuehrenAnonymisiert) {
		t.Errorf("anonymisierte Quelle: %v", err)
	}
	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(a, "00000000-0000-0000-0000-000000000000")); !errors.Is(err, repository.ErrZusammenfuehrenNichtGefunden) {
		t.Errorf("unbekannte Quelle: %v", err)
	}
	if n := zfZaehle(t, pool, `SELECT count(*) FROM schueler WHERE deleted_at IS NULL`); n != 2 {
		t.Errorf("Abweisungen dürfen nichts verändern (n=%d)", n)
	}
}

// Die Kandidatensuche findet Abgänger und Gesperrte, nie den Ausgangsdatensatz selbst
// und nie Anonymisierte.
func TestZusammenfuehren_KandidatenSucheSiehtAbgaenger(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	ich := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Mira", nachname: "Suchwort", klasse: "07A", barcode: "ZF-6", geb: datum(2012, 6, 6)})
	legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Mira", nachname: "Suchwort-Alt", klasse: "06A", barcode: "ZF-7", geb: datum(2012, 6, 6), abgaenger: true})
	anon := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Mira", nachname: "Suchwort-Anon", klasse: "ABG", barcode: "ZF-8"})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET anonymized_at = NOW() WHERE id = $1`, anon); err != nil {
		t.Fatal(err)
	}

	treffer, err := repository.SucheZusammenfuehrenKandidaten(ctx, pool, ich, "Suchwort", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(treffer) != 1 || treffer[0].Nachname != "Suchwort-Alt" || !treffer[0].IstAbgaenger || !treffer[0].IstGesperrt {
		t.Errorf("erwartet genau den gesperrten Abgänger: %+v", treffer)
	}
}

// Gate über ALLE Tabellen, die an einem Schüler hängen (dieselbe Liste wie
// dsgvo_paar_vollstaendigkeit_test.go): Der erste Test säte nur die Ausleihe (FK RESTRICT
// — ein ausgelassener Schritt bräche laut). Vormerkungen und Foto hängen per CASCADE:
// Fällt ihr UPDATE weg, löscht der DELETE der Quelle sie STILL mit. Deshalb hier jede
// Tabelle befüllt und nach dem Zusammenführen gezählt (Rasterdurchgang 02.09.2026).
func TestZusammenfuehren_JedeTabelleWandert(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Ziel", nachname: "Tabelle", klasse: "07A", barcode: "ZF-T1", geb: datum(2012, 7, 7)})
	quelle := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Quelle", nachname: "Tabelle", klasse: "07A", barcode: "ZF-T2", geb: datum(2012, 7, 7)})

	seedOffeneAusleihe(t, pool, quelle, "ZFT")
	seedOffenerSchaden(t, pool, quelle, "ZFT")
	titelID := titelMitMeldebestand(t, pool, "Vormerktitel-ZFT", 1)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql[:40], err)
		}
	}
	exec(`INSERT INTO vormerkungen (titel_id, schueler_id) VALUES ($1, $2)`, titelID, quelle)
	exec(`INSERT INTO schueler_fotos (schueler_id, foto_encrypted) VALUES ($1, '\x00'::bytea)`, quelle)
	exec(`INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details)
		VALUES ('ausleihen', 'CREATE', gen_random_uuid(), 'USER', jsonb_build_object('schueler_id', $1::text))`, quelle)
	exec(`INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details)
		VALUES ('schueler', 'UPDATE', $1::uuid, 'USER', '{"feld":"klasse"}'::jsonb)`, quelle)
	exec(`INSERT INTO audit_logs (aktion, details) VALUES ('LUSD_ID_NACHGETRAGEN', jsonb_build_object('schueler_id', $1::text))`, quelle)

	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, quelle)); err != nil {
		t.Fatalf("Zusammenführen: %v", err)
	}

	zaehlungen := []struct {
		was, sql string
	}{
		{"ausleihen", `SELECT count(*) FROM ausleihen WHERE schueler_id = $1`},
		{"schadensfaelle", `SELECT count(*) FROM schadensfaelle WHERE schueler_id = $1`},
		{"vormerkungen", `SELECT count(*) FROM vormerkungen WHERE schueler_id = $1`},
		{"schueler_fotos", `SELECT count(*) FROM schueler_fotos WHERE schueler_id = $1`},
		{"audit_log Lesehistorie", `SELECT count(*) FROM audit_log WHERE tabelle = 'ausleihen' AND details->>'schueler_id' = $1`},
		{"audit_log Datensatz-Historie", `SELECT count(*) FROM audit_log WHERE tabelle = 'schueler' AND aktion = 'UPDATE' AND datensatz_id = $1::uuid`},
		{"audit_logs", `SELECT count(*) FROM audit_logs WHERE aktion = 'LUSD_ID_NACHGETRAGEN' AND details->>'schueler_id' = $1`},
	}
	for _, z := range zaehlungen {
		if n := zfZaehle(t, pool, z.sql, ziel); n != 1 {
			t.Errorf("%s: erwartet 1 Zeile am Ziel, gefunden %d", z.was, n)
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
// gewanderten Zeilen — dieser Test geht den Rückweg am ERGEBNIS: Quelle aus dem Eintrag
// neu anlegen, Ausleihe zurückschlüsseln, Ziel auf den alten Stand setzen.
func TestZusammenfuehren_RueckwegAusDemProtokoll(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Alt", nachname: "Rueckweg", klasse: "07A", barcode: "ZF-R1", geb: datum(2012, 8, 8), abgaenger: true})
	quelle := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Falsch", nachname: "Rueckweg", klasse: "08B", barcode: "ZF-R2", geb: datum(2012, 8, 8)})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET lusd_bestaetigt_am = NOW(), strasse = 'Irrweg', plz = '61381', ort = 'Friedrichsdorf' WHERE id = $1`, quelle); err != nil {
		t.Fatal(err)
	}
	seedOffeneAusleihe(t, pool, quelle, "ZFR")

	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle, AbgaengerJahr: calculateAbgaengerJahr}); err != nil {
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

// Eine manuelle Sperre der QUELLE (Ausweis verloren, Hausverbot …) darf beim Zusammen-
// führen nicht still verschwinden — sie gehört zur Person, nicht zum Datensatz
// (Rasterdurchgang 02.09.2026). Das Ziel trägt sie danach mit demselben Grund.
func TestZusammenfuehren_ManuelleSperreDerQuelleBleibt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Ziel", nachname: "Sperre", klasse: "07A", barcode: "ZF-M1", geb: datum(2012, 9, 9)})
	quelle := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Quelle", nachname: "Sperre", klasse: "07A", barcode: "ZF-M2", geb: datum(2012, 9, 9)})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET ist_gesperrt = true, is_manually_blocked = true, block_reason = 'Manuell: Ausweis als gestohlen gemeldet' WHERE id = $1`, quelle); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, quelle)); err != nil {
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

// Führend ist der Datensatz, den die LUSD zuletzt bestätigt hat — aber ein AKTIVER
// Datensatz schlägt einen Abgänger, auch wenn er nie bestätigt wurde: Das Kind kam nach
// der Umbenennung an die Theke, wurde von Hand neu angelegt (Handanlage, aktuelle Klasse),
// und der Admin behält den alten Datensatz wegen des Ausweises. Bis 02.09.2026 gewannen
// dann Name und Klasse des ABGÄNGERS — und der nächste Namensmodus-Import spaltete das
// Kind erneut (Rasterdurchgang, Frage 8).
func TestZusammenfuehren_AktiveHandanlageSchlaegtAbgaenger(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Alt", nachname: "Fuehrend", klasse: "07A", barcode: "ZF-F1", geb: datum(2012, 10, 10), abgaenger: true})
	quelle := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Neu", nachname: "Fuehrend-Neu", klasse: "08A", barcode: "ZF-F2", geb: datum(2012, 10, 10)})
	// Die Handanlage wurde nie von der LUSD bestätigt.
	if _, err := pool.Exec(ctx, `UPDATE schueler SET lusd_bestaetigt_am = NULL WHERE id = $1`, quelle); err != nil {
		t.Fatal(err)
	}
	erg, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, quelle))
	if err != nil {
		t.Fatal(err)
	}
	if erg.Nachname != "Fuehrend-Neu" || !klassenGleich(erg.Klasse, "08A") {
		t.Errorf("aktive Handanlage muss die Stammdaten stellen, nicht der Abgänger: %+v", erg)
	}
}

// Vormerkungs-Dublette: Beide haben denselben Titel vorgemerkt, die Quelle ist schon
// „abholbereit" (Exemplar liegt bereit), das Ziel noch „wartend". Bis 02.09.2026 fiel
// die Vormerkung der QUELLE — das Kind verlor den bereitgestellten Platz, das Exemplar
// stand frei, niemand wurde bedient. Es bleibt die weiter fortgeschrittene Vormerkung,
// egal auf welcher Seite sie steht (Rasterdurchgang, Frage 5/8).
func TestZusammenfuehren_FortgeschritteneVormerkungBleibt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Ziel", nachname: "Vormerk", klasse: "07A", barcode: "ZF-V1", geb: datum(2012, 11, 11)})
	quelle := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Quelle", nachname: "Vormerk", klasse: "07A", barcode: "ZF-V2", geb: datum(2012, 11, 11)})
	titelID := titelMitMeldebestand(t, pool, "Vormerktitel-ZFV", 1)
	exID := exemplar(t, pool, titelID, "EX-ZFV", true, "")
	if _, err := pool.Exec(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status) VALUES ($1, $2, 'wartend')`, titelID, ziel); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_exemplar_id, bereitgestellt_bis)
		VALUES ($1, $2, 'abholbereit', $3, CURRENT_TIMESTAMP + INTERVAL '2 days')`, titelID, quelle, exID); err != nil {
		t.Fatal(err)
	}
	erg, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, quelle))
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
		t.Errorf("genau eine Vormerkung je Titel erwartet, gefunden %d", n)
	}
}

// Foto: kommt nie aus der LUSD — das JÜNGERE gewinnt, egal auf welcher Seite. Bis
// 03.09.2026 blieb immer das Foto des Ziels, das jüngere der Quelle starb im CASCADE.
func TestZusammenfuehren_JuengeresFotoGewinnt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Ziel", nachname: "Foto", klasse: "07A", barcode: "ZF-P1", geb: datum(2012, 12, 12)})
	quelle := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Quelle", nachname: "Foto", klasse: "07A", barcode: "ZF-P2", geb: datum(2012, 12, 12)})
	for _, f := range []struct {
		id, bytes, alter string
	}{{ziel, `\x01`, "2 years"}, {quelle, `\x02`, "1 day"}} {
		if _, err := pool.Exec(ctx, `INSERT INTO schueler_fotos (schueler_id, foto_encrypted, aktualisiert_am) VALUES ($1, $2::bytea, NOW() - $3::interval)`, f.id, f.bytes, f.alter); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, quelle)); err != nil {
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
		t.Errorf("genau ein Foto erwartet, gefunden %d", n)
	}
}
