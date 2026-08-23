package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Anliegen-Befristung am echten Postgres. Geprüft wird die PAARUNG aus Frist und
// Erledigungs-Zustand — das, was ein Mock nur nachspielen würde: die
// Intervall-Rechnung auf `erledigt_am` (nicht auf `erstellt_am`) und der Wächter, der
// offene Anliegen unangetastet lässt.
//
// Erwartung nach dem Lauf mit der Vorgabe (365 Tage):
//
//	ALT-ERLEDIGT   vor 400 Tagen erledigt   → gelöscht
//	JUNG-ERLEDIGT  vor 10 Tagen erledigt    → bleibt
//	ALT-OFFEN      vor 400 Tagen angelegt,
//	               nie erledigt             → bleibt (laufende Sache, keine Frist)
func TestAnliegenBefristung_LoeschtNurErledigteNachFrist(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "anliegen")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	anlege := func(titel string, erledigtVorTagen int, offen bool) string {
		t.Helper()
		var id string
		sql := `INSERT INTO lehrer_anliegen (art, titel_text, kommentar, erstellt_am, erledigt_am)
		        VALUES ('wunsch', $1, 'Bitte anschaffen',
		                NOW() - make_interval(days => $2 + 30),
		                CASE WHEN $3 THEN NULL ELSE NOW() - make_interval(days => $2) END)
		        RETURNING id`
		if err := pool.QueryRow(ctx, sql, titel, erledigtVorTagen, offen).Scan(&id); err != nil {
			t.Fatalf("Anliegen %s: %v", titel, err)
		}
		return id
	}

	ids := map[string]string{
		"ALT-ERLEDIGT":  anlege("ALT-ERLEDIGT", 400, false),
		"JUNG-ERLEDIGT": anlege("JUNG-ERLEDIGT", 10, false),
		"ALT-OFFEN":     anlege("ALT-OFFEN", 400, true),
	}

	s := NewScheduler(pool, repository.NewAuditRepository(pool))
	s.RunAnliegenBefristung()

	existiert := func(id string) bool {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM lehrer_anliegen WHERE id = $1`, id).Scan(&n); err != nil {
			t.Fatalf("lehrer_anliegen lesen: %v", err)
		}
		return n > 0
	}

	for name, bleibt := range map[string]bool{"ALT-ERLEDIGT": false, "JUNG-ERLEDIGT": true, "ALT-OFFEN": true} {
		if got := existiert(ids[name]); got != bleibt {
			t.Errorf("%s: vorhanden = %v, erwartet %v", name, got, bleibt)
		}
	}

	// Der Lauf muss sich im Audit-Protokoll zeigen — eine Löschung ohne Spur ist genau
	// das, was bei den DSGVO-Jobs zweimal monatelang unbemerkt ins Leere lief.
	var eintraege int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_log WHERE tabelle = 'lehrer_anliegen' AND aktion = 'DELETE'`).Scan(&eintraege); err != nil {
		t.Fatalf("audit_log lesen: %v", err)
	}
	if eintraege == 0 {
		t.Error("kein Audit-Eintrag für den Löschlauf")
	}
}

// TestAnliegenBefristung_NullSchaltetAb: Die 0 ist in diesen Einstellungen ein echter
// Wert und heißt "aus". Fiele sie durch, löschte der Job mit der Vorgabe weiter — und
// die Schule hätte eine Abschaltung, die nichts abschaltet.
func TestAnliegenBefristung_NullSchaltetAb(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "anliegenaus")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO lehrer_anliegen (art, titel_text, erstellt_am, erledigt_am)
		VALUES ('meldung', 'UR-ALT', NOW() - interval '2000 days', NOW() - interval '1900 days')
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Anliegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO system_einstellungen (schluessel, wert) VALUES ('anliegen_tage', '0')
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`); err != nil {
		t.Fatalf("Einstellung: %v", err)
	}

	NewScheduler(pool, repository.NewAuditRepository(pool)).RunAnliegenBefristung()

	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM lehrer_anliegen WHERE id = $1`, id).Scan(&n); err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if n == 0 {
		t.Error("0 Tage muss die Befristung abschalten — das Anliegen wurde trotzdem gelöscht")
	}
}
