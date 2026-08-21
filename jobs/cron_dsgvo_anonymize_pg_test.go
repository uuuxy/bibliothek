package jobs

import (
	"context"
	"os"
	"strings"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestGDPRAnonymizeOldData_LeertAllePII deckt einen kritischen, lange unentdeckten Fehler
// ab (PII-Lebenszyklus-Sweep 19.08.2026): RunGDPRAnonymizeOldData referenzierte eine nicht
// existente Spalte (foto_url) und scheiterte deshalb ZUR LAUFZEIT — die Anonymisierung lief
// NIE, der Fehler wurde nur geloggt. Die bestehenden gdpr_test.go nutzen pgxmock und konnten
// das prinzipiell nicht sehen; erst dieser Lauf gegen echtes Postgres deckt es auf.
//
// Der Test befüllt einen alten, soft-gelöschten Schüler mit unverwechselbaren Sentinel-Werten
// in JEDER PII-Spalte und prüft nach der Anonymisierung, dass (a) sie wirklich lief
// (anonymized_at gesetzt) und (b) KEIN Sentinel mehr in der Zeile steht. Eine neu
// hinzugefügte PII-Spalte, die der Anonymisierer vergisst, fällt hier auf, sobald der Seed
// sie mit einem Sentinel füllt.
func TestGDPRAnonymizeOldData_LeertAllePII(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}

	_, dsn := legeProbeDatenbankAn(t, adminDSN, "gdpranon")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Sentinel-Werte: unverwechselbar, damit ein Überleben eindeutig auffällt.
	const (
		vn     = "SENTINELVORNAME"
		nn     = "SENTINELNACHNAME"
		strdat = "SENTINELSTRASSE"
		ortdat = "SENTINELORT"
		email  = "sentinel@eltern.invalid"
		lusd   = "SENTINEL-LUSD-42"
		reason = "SENTINEL Sperrgrund vertraulich"
		barc   = "SENTINEL-BARCODE"
	)
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler
		    (barcode_id, vorname, nachname, klasse, geburtsdatum, abgaenger_jahr, lusd_id,
		     strasse, hausnummer, plz, ort, eltern_email,
		     ist_gesperrt, block_reason, deleted_at)
		VALUES ($1, $2, $3, '7b', '2011-05-05', 2030, $4,
		        $5, '12a', '61250', $6, $7,
		        true, $8, now() - interval '200 days')
		RETURNING id`,
		barc, vn, nn, lusd, strdat, ortdat, email, reason).Scan(&id); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	s := NewScheduler(pool, repository.NewAuditRepository(pool))
	s.RunGDPRAnonymizeOldData()

	// (a) Die Anonymisierung MUSS gelaufen sein. Vor dem Fix scheiterte die Query still
	// (foto_url/anonymized_at existierten nicht) → anonymized_at bliebe NULL.
	var anonymisiert bool
	if err := pool.QueryRow(ctx,
		`SELECT anonymized_at IS NOT NULL FROM schueler WHERE id = $1`, id).Scan(&anonymisiert); err != nil {
		t.Fatalf("anonymized_at lesen: %v", err)
	}
	if !anonymisiert {
		t.Fatal("RunGDPRAnonymizeOldData hat den fälligen Schüler NICHT anonymisiert (Query still gescheitert?)")
	}

	// (b) KEIN Sentinel darf überleben — die ganze Zeile als Text zusammenziehen und prüfen.
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
	for _, sentinel := range []string{vn, nn, strdat, ortdat, email, lusd, reason, barc} {
		if strings.Contains(zeile, sentinel) {
			t.Errorf("PII überlebt die Anonymisierung: %q steht noch in der Zeile: %s", sentinel, zeile)
		}
	}
	// Auch das Geburtsdatum darf nicht mehr dastehen (direkt identifizierend).
	if strings.Contains(zeile, "2011-05-05") {
		t.Errorf("Geburtsdatum überlebt die Anonymisierung: %s", zeile)
	}
}

// TestGDPRAnonymize_TilgtNebenTabellenPII deckt den DSGVO-Fund vom 21.08.2026 ab: Die
// Feld-Anonymisierung leerte nur die schueler-Zeile, ließ aber den Klarnamen im
// audit_log, die LUSD-ID im audit_logs und die Vormerkungs-Notiz stehen — der
// Personenbezug überlebte die „Löschung" bis zur Audit-Aufbewahrung (bis zu 24 Monate).
// Der Test setzt in jede dieser Nebentabellen einen Sentinel und prüft, dass nach der
// Anonymisierung keiner mehr auffindbar ist.
func TestGDPRAnonymize_TilgtNebenTabellenPII(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
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
		nachnameSentinel = "NEBENSENTINELNACHNAME"
		lusdSentinel     = "NEBEN-LUSD-4711"
		notizSentinel    = "NEBENNOTIZ-VERTRAULICH"
	)

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger, aktualisiert_am)
		VALUES ('NEBEN-BC', 'Max', $1, '9c', 2030, true, now() - interval '400 days')
		RETURNING id`, nachnameSentinel).Scan(&id); err != nil {
		t.Fatalf("Schüler: %v", err)
	}

	// audit_log: fachliche Historie mit Klarname (wie DeleteStudent schreibt).
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, details, akteur)
		VALUES ('schueler', 'UPDATE', $1, jsonb_build_object('nachname', $2::text), 'USER')`,
		id, nachnameSentinel); err != nil {
		t.Fatalf("audit_log-Seed: %v", err)
	}
	// audit_logs: Admin-Eingriff mit LUSD-ID (wie LUSD_ID_NACHGETRAGEN schreibt).
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_logs (aktion, details)
		VALUES ('LUSD_ID_NACHGETRAGEN', jsonb_build_object('schueler_id', $1::text, 'lusd_id', $2::text))`,
		id, lusdSentinel); err != nil {
		t.Fatalf("audit_logs-Seed: %v", err)
	}
	// vormerkung mit personenbeziehbarer Notiz.
	var titelID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel) VALUES ('Neben-Titel') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO vormerkungen (titel_id, schueler_id, notiz) VALUES ($1, $2, $3)`,
		titelID, id, notizSentinel); err != nil {
		t.Fatalf("vormerkung-Seed: %v", err)
	}

	NewScheduler(pool, repository.NewAuditRepository(pool)).RunGDPRAnonymizeOldData()

	// Anonymisierung muss gelaufen sein.
	var anon bool
	if err := pool.QueryRow(ctx, `SELECT anonymized_at IS NOT NULL FROM schueler WHERE id=$1`, id).Scan(&anon); err != nil || !anon {
		t.Fatalf("Schüler nicht anonymisiert (anon=%v, err=%v)", anon, err)
	}

	// audit_log: kein Klarname mehr.
	var auditLogPII int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_log WHERE datensatz_id=$1 AND details::text LIKE '%'||$2||'%'`,
		id, nachnameSentinel).Scan(&auditLogPII); err != nil {
		t.Fatalf("audit_log prüfen: %v", err)
	}
	if auditLogPII > 0 {
		t.Errorf("Klarname überlebt im audit_log (%d Treffer)", auditLogPII)
	}

	// audit_logs: keine LUSD-ID mehr.
	var auditLogsPII int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs WHERE details::text LIKE '%'||$1||'%'`,
		lusdSentinel).Scan(&auditLogsPII); err != nil {
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
