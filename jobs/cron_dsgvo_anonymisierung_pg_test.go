package jobs

import (
	"context"
	"os"
	"strings"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Eine Anonymisierung, die an einer Spalte scheitert, die es nicht gibt, scheitert erst zur
// Laufzeit, und der Fehler steht nur im Log. Die Tests mit pgxmock (cron_dsgvo_test.go) können
// das nicht sehen; erst der Lauf gegen Postgres deckt es auf.
//
// Der Test füllt einen alten Schüler im Papierkorb in jeder Spalte mit Personendaten mit
// unverwechselbaren Marken und prüft nach der Anonymisierung, dass (a) sie wirklich lief
// (anonymized_at gesetzt) und (b) keine Marke mehr in der Zeile steht. Eine neue Spalte mit
// Personendaten, die die Anonymisierung vergisst, fällt hier auf, sobald die Testdaten sie mit
// einer Marke füllen.
func TestSchuelerAnonymisierung_LeertAllePII(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
	}

	_, dsn := legeProbeDatenbankAn(t, adminDSN, "gdpranon")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Marken: unverwechselbar, damit ein Überleben eindeutig auffällt.
	const (
		vn     = "SENTINELVORNAME"
		nn     = "SENTINELNACHNAME"
		strdat = "SENTINELSTRASSE"
		ortdat = "SENTINELORT"
		email  = "sentinel@eltern.invalid"
		lusd   = "SENTINEL-LUSD-42"
		grund  = "SENTINEL Sperrgrund vertraulich"
		barc   = "SENTINEL-BARCODE"
	)
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler
		    (barcode_id, vorname, nachname, klasse, geburtsdatum, abgaenger_jahr, lusd_id,
		     strasse, hausnummer, plz, ort, eltern_email,
		     ist_gesperrt, block_reason, deleted_at, schul_eintritt_am)
		VALUES ($1, $2, $3, '7b', '2011-05-05', 2030, $4,
		        $5, '12a', '61250', $6, $7,
		        true, $8, now() - interval '200 days', '2022-08-29')
		RETURNING id`,
		barc, vn, nn, lusd, strdat, ortdat, email, grund).Scan(&id); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	z := NewZeitplan(pool, repository.NewProtokollAbfragen(pool))
	z.fuehreSchuelerAnonymisierungAus(ctx)

	// (a) Die Anonymisierung muss gelaufen sein; scheitert die Abfrage stumm, bliebe
	// anonymized_at NULL.
	var anonymisiert bool
	if err := pool.QueryRow(ctx,
		`SELECT anonymized_at IS NOT NULL FROM schueler WHERE id = $1`, id).Scan(&anonymisiert); err != nil {
		t.Fatalf("anonymized_at lesen: %v", err)
	}
	if !anonymisiert {
		t.Fatal("fuehreSchuelerAnonymisierungAus hat den fälligen Schüler NICHT anonymisiert (Query still gescheitert?)")
	}

	// (b) Keine Marke darf überleben: die ganze Zeile als Text zusammenziehen und prüfen.
	var zeile string
	if err := pool.QueryRow(ctx, `
		SELECT concat_ws('|',
			coalesce(vorname,''), coalesce(nachname,''), coalesce(klasse,''),
			coalesce(barcode_id,''), coalesce(geburtsdatum::text,''), coalesce(lusd_id,''),
			coalesce(strasse,''), coalesce(hausnummer,''), coalesce(plz,''), coalesce(ort,''),
			coalesce(eltern_email,''), coalesce(block_reason,''))
		FROM schueler WHERE id = $1`, id).Scan(&zeile); err != nil {
		t.Fatalf("Zeile lesen: %v", err)
	}
	for _, marke := range []string{vn, nn, strdat, ortdat, email, lusd, grund, barc} {
		if strings.Contains(zeile, marke) {
			t.Errorf("PII überlebt die Anonymisierung: %q steht noch in der Zeile: %s", marke, zeile)
		}
	}
	// Auch das Geburtsdatum darf nicht mehr dastehen (direkt identifizierend).
	if strings.Contains(zeile, "2011-05-05") {
		t.Errorf("Geburtsdatum überlebt die Anonymisierung: %s", zeile)
	}
	// Spalten vom Typ DATE fallen aus der Marke als Text heraus, deshalb je Spalte prüfen: Der
	// Schuleintritt ist mit abgaenger_jahr ein Merkmal, das eine Person über den Jahrgang erkennbar
	// macht, und darf die Anonymisierung ebenfalls nicht überleben.
	var eintrittWeg bool
	if err := pool.QueryRow(ctx, `SELECT schul_eintritt_am IS NULL FROM schueler WHERE id = $1`, id).Scan(&eintrittWeg); err != nil {
		t.Fatal(err)
	}
	if !eintrittWeg {
		t.Error("schul_eintritt_am überlebt die Anonymisierung")
	}
}

// Mit der Anonymisierung fällt das Passfoto. Das gilt für den Schüler, den der Lauf dieser
// Nacht anonymisiert, und für einen früher anonymisierten, dessen Foto stehen geblieben ist;
// das Foto eines Schülers, der nicht anonymisiert ist, bleibt.
func TestSchuelerAnonymisierung_LoeschtFotosAnonymisierterSchueler(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "gdprfotos")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	schueler := func(was, sql string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, sql).Scan(&id); err != nil {
			t.Fatalf("%s: %v", was, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO schueler_fotos (schueler_id, foto_encrypted) VALUES ($1, 'foto')`, id); err != nil {
			t.Fatalf("Foto zu %s: %v", was, err)
		}
		return id
	}
	heute := schueler("Schüler im Papierkorb", `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, deleted_at)
		VALUES ('S-FOTO-HEUTE', 'Paula', 'Papierkorb', '9c', 2030, NOW() - interval '200 days') RETURNING id`)
	frueher := schueler("früher anonymisierter Schüler", `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, deleted_at, anonymized_at)
		VALUES ('ANON-FRUEHER', 'x', 'Anonym', '', 2030, NOW() - interval '300 days', NOW() - interval '100 days') RETURNING id`)
	aktiv := schueler("aktiver Schüler", `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('S-FOTO-AKTIV', 'Anna', 'Aktiv', '7a', 2030) RETURNING id`)

	NewZeitplan(pool, repository.NewProtokollAbfragen(pool)).fuehreSchuelerAnonymisierungAus(ctx)

	hatFoto := func(id string) bool {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM schueler_fotos WHERE schueler_id = $1`, id).Scan(&n); err != nil {
			t.Fatalf("Foto lesen: %v", err)
		}
		return n > 0
	}
	if hatFoto(heute) {
		t.Error("Das Foto des Schülers, den dieser Lauf anonymisiert hat, steht noch")
	}
	if hatFoto(frueher) {
		t.Error("Das Foto eines früher anonymisierten Schülers steht noch")
	}
	if !hatFoto(aktiv) {
		t.Error("Das Foto eines Schülers, der nicht anonymisiert ist, wurde gelöscht")
	}
}

// Die Anonymisierung leert nicht nur die Zeile des Schülers: Klarname im audit_log, LUSD-ID im
// audit_logs und Notiz der Vormerkung überlebten sonst die „Löschung" bis zum Ende der
// Aufbewahrung des Protokolls (bis zu 24 Monate). Der Test setzt in jede dieser Tabellen eine
// Marke und prüft, dass nach der Anonymisierung keine mehr zu finden ist.
func TestSchuelerAnonymisierung_TilgtPersonendatenInAnderenTabellen(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "gdprnebendata")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	const (
		nachnameMarke = "NEBENSENTINELNACHNAME"
		lusdMarke     = "NEBEN-LUSD-4711"
		notizMarke    = "NEBENNOTIZ-VERTRAULICH"
	)

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger, aktualisiert_am)
		VALUES ('NEBEN-BC', 'Max', $1, '9c', 2030, true, now() - interval '400 days')
		RETURNING id`, nachnameMarke).Scan(&id); err != nil {
		t.Fatalf("Schüler: %v", err)
	}

	// audit_log: fachliche Historie mit Klarname (wie LegeLeserInPapierkorb schreibt).
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, details, akteur)
		VALUES ('schueler', 'UPDATE', $1, jsonb_build_object('nachname', $2::text), 'USER')`,
		id, nachnameMarke); err != nil {
		t.Fatalf("audit_log-Seed: %v", err)
	}
	// audit_logs: Eingriff der Verwaltung mit LUSD-ID (wie LUSD_ID_NACHGETRAGEN schreibt).
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_logs (aktion, details)
		VALUES ('LUSD_ID_NACHGETRAGEN', jsonb_build_object('schueler_id', $1::text, 'lusd_id', $2::text))`,
		id, lusdMarke); err != nil {
		t.Fatalf("audit_logs-Seed: %v", err)
	}
	// Vormerkung mit einer Notiz, die auf die Person schließen lässt.
	var titelID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel) VALUES ('Neben-Titel') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO vormerkungen (titel_id, schueler_id, notiz) VALUES ($1, $2, $3)`,
		titelID, id, notizMarke); err != nil {
		t.Fatalf("vormerkung-Seed: %v", err)
	}

	NewZeitplan(pool, repository.NewProtokollAbfragen(pool)).fuehreSchuelerAnonymisierungAus(ctx)

	// Die Anonymisierung muss gelaufen sein.
	var anon bool
	if err := pool.QueryRow(ctx, `SELECT anonymized_at IS NOT NULL FROM schueler WHERE id=$1`, id).Scan(&anon); err != nil || !anon {
		t.Fatalf("Schüler nicht anonymisiert (anon=%v, err=%v)", anon, err)
	}

	// audit_log: kein Klarname mehr.
	var auditLogPII int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_log WHERE datensatz_id=$1 AND details::text LIKE '%'||$2||'%'`,
		id, nachnameMarke).Scan(&auditLogPII); err != nil {
		t.Fatalf("audit_log prüfen: %v", err)
	}
	if auditLogPII > 0 {
		t.Errorf("Klarname überlebt im audit_log (%d Treffer)", auditLogPII)
	}

	// audit_logs: keine LUSD-ID mehr.
	var auditLogsPII int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs WHERE details::text LIKE '%'||$1||'%'`,
		lusdMarke).Scan(&auditLogsPII); err != nil {
		t.Fatalf("audit_logs prüfen: %v", err)
	}
	if auditLogsPII > 0 {
		t.Errorf("LUSD-ID überlebt im audit_logs (%d Treffer)", auditLogsPII)
	}

	// vormerkungen: keine Notiz mehr (Vormerkung gelöscht).
	var vormerkungen int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM vormerkungen WHERE schueler_id=$1`, id).Scan(&vormerkungen); err != nil {
		t.Fatalf("vormerkungen prüfen: %v", err)
	}
	if vormerkungen > 0 {
		t.Errorf("Vormerkung anonymisierten Schülers überlebt (%d)", vormerkungen)
	}
}
