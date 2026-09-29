package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Befristung erledigter Klassensatz-Reservierungen am echten Postgres (entschieden am
// 28.09.2026, docs/OFFEN.md 5.19): dieselbe Frist wie die erledigten Anliegen.
//
// Erwartung nach dem Lauf mit der Vorgabe (365 Tage):
//
//	ALT-ERLEDIGT       vor 400 Tagen erledigt              → gelöscht
//	JUNG-ERLEDIGT      vor 10 Tagen erledigt               → bleibt
//	ALT-OFFEN          vor 800 Tagen angelegt, offen       → bleibt (laufende Sache, keine Frist)
//	OHNE-ZEITPUNKT     erledigt vor Migration 089, ohne
//	                   erledigt_am                         → bleibt (kein Zeitpunkt, keiner erfunden)
//	WIEDER-OFFEN       offen, trägt aber einen alten
//	                   erledigt_am                         → bleibt (heute schreibt das kein Weg;
//	                                                         wird eine Reservierung je wieder
//	                                                         geöffnet, zählt „erledigt")
func TestKlassensatzBefristung_LoeschtNurErledigteNachFrist(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "klassensatz")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	var titelID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, ist_lernmittel) VALUES ('Klassensatz-Titel', true) RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel: %v", err)
	}
	lege := func(notiz string, erledigt bool, erledigtVorTagen *int) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, notiz, erledigt, erstellt_am, erledigt_am)
			VALUES ($1, '7a', 30, $2, $3, NOW() - interval '800 days',
			        CASE WHEN $4::int IS NULL THEN NULL ELSE NOW() - make_interval(days => $4::int) END)
			RETURNING id`, titelID, notiz, erledigt, erledigtVorTagen).Scan(&id); err != nil {
			t.Fatalf("Reservierung %s: %v", notiz, err)
		}
		return id
	}
	tage := func(n int) *int { return &n }

	ids := map[string]string{
		"ALT-ERLEDIGT":   lege("ALT-ERLEDIGT", true, tage(400)),
		"JUNG-ERLEDIGT":  lege("JUNG-ERLEDIGT", true, tage(10)),
		"ALT-OFFEN":      lege("ALT-OFFEN", false, nil),
		"OHNE-ZEITPUNKT": lege("OHNE-ZEITPUNKT", true, nil),
		"WIEDER-OFFEN":   lege("WIEDER-OFFEN", false, tage(400)),
	}

	NewScheduler(pool, repository.NewAuditRepository(pool)).RunKlassensatzBefristung()

	for name, bleibt := range map[string]bool{
		"ALT-ERLEDIGT": false, "JUNG-ERLEDIGT": true, "ALT-OFFEN": true, "OHNE-ZEITPUNKT": true,
		"WIEDER-OFFEN": true,
	} {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM klassensatz_reservierungen WHERE id = $1`, ids[name]).Scan(&n); err != nil {
			t.Fatalf("lesen: %v", err)
		}
		if got := n > 0; got != bleibt {
			t.Errorf("%s: vorhanden = %v, erwartet %v", name, got, bleibt)
		}
	}

	// Der Lauf zeigt sich im Protokoll, mit der Zahl.
	var geloescht string
	if err := pool.QueryRow(ctx, `
		SELECT coalesce(details->>'geloescht', '') FROM audit_log
		WHERE tabelle = 'klassensatz_reservierungen' AND aktion = 'DELETE'`).Scan(&geloescht); err != nil {
		t.Fatalf("Protokolleintrag des Laufs: %v", err)
	}
	if geloescht != "1" {
		t.Errorf("Protokoll nennt %q gelöschte Reservierungen, erwartet 1", geloescht)
	}
}

// TestKlassensatzBefristung_NullSchaltetAb: 0 in anliegen_tage heißt „aus", auch für die
// Reservierungen.
func TestKlassensatzBefristung_NullSchaltetAb(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "klassensatzaus")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	var id string
	if err := pool.QueryRow(ctx, `
		WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('UR-ALT') RETURNING id)
		INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, erledigt, erstellt_am, erledigt_am)
		SELECT id, '7a', 30, true, NOW() - interval '2000 days', NOW() - interval '1900 days' FROM t
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Reservierung: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO system_einstellungen (schluessel, wert) VALUES ('anliegen_tage', '0')
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`); err != nil {
		t.Fatalf("Einstellung: %v", err)
	}

	NewScheduler(pool, repository.NewAuditRepository(pool)).RunKlassensatzBefristung()

	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM klassensatz_reservierungen WHERE id = $1`, id).Scan(&n); err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if n == 0 {
		t.Error("0 Tage muss die Befristung abschalten — die Reservierung wurde trotzdem gelöscht")
	}
}
