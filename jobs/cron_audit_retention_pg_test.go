package jobs

import (
	"context"
	"os"
	"strings"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Aufbewahrung am echten Postgres: Alt fliegt raus, Neu bleibt, und die
// Löschung hinterlässt genau einen Meta-Eintrag mit den Zahlen. Das Intervall-SQL
// (make_interval über beide Tabellen mit verschiedenen Zeitspalten) ist genau die
// Sorte, die ein Mock nur nachspielen würde.
func TestAuditAufbewahrung_LoeschtAltesUndProtokolliert(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}

	_, dsn := legeProbeDatenbankAn(t, adminDSN, "retention")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Ein Bearbeiter für die FK-Spalten + je Tabelle ein alter und ein junger Eintrag.
	var benutzerID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ('RET-B', 'Retention', 'Kraft', 'ret@example.org', 'admin', true) RETURNING id`).Scan(&benutzerID); err != nil {
		t.Fatalf("Benutzer: %v", err)
	}
	seed := func(query string, alterMonate int) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, benutzerID, alterMonate); err != nil {
			t.Fatalf("Seed (%d Monate): %v", alterMonate, err)
		}
	}
	seed(`INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, timestamp)
	      VALUES ('schueler', 'UPDATE', gen_random_uuid(), $1, now() - make_interval(months => $2))`, 30)
	seed(`INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, timestamp)
	      VALUES ('schueler', 'UPDATE', gen_random_uuid(), $1, now() - make_interval(months => $2))`, 3)
	seed(`INSERT INTO audit_logs (admin_id, aktion, ip_adresse, zeitstempel)
	      VALUES ($1, 'LOGIN', '203.0.113.7', now() - make_interval(months => $2))`, 30)
	seed(`INSERT INTO audit_logs (admin_id, aktion, ip_adresse, zeitstempel)
	      VALUES ($1, 'LOGIN', '203.0.113.7', now() - make_interval(months => $2))`, 3)

	s := NewScheduler(pool, repository.NewAuditRepository(pool))
	s.RunAuditAufbewahrung()

	zaehle := func(query string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, query).Scan(&n); err != nil {
			t.Fatalf("Zählen: %v", err)
		}
		return n
	}
	// Vorgabe 24 Monate (keine Einstellung gesetzt): 30 Monate alt fliegt, 3 bleibt.
	if n := zaehle(`SELECT count(*) FROM audit_log WHERE aktion = 'UPDATE'`); n != 1 {
		t.Errorf("audit_log: %d UPDATE-Einträge, erwartet 1 (der junge)", n)
	}
	if n := zaehle(`SELECT count(*) FROM audit_logs WHERE aktion = 'LOGIN'`); n != 1 {
		t.Errorf("audit_logs: %d LOGIN-Einträge, erwartet 1 (der junge)", n)
	}

	// Meta-Audit: genau eine RETENTION-Zeile mit beiden Zahlen.
	var details string
	if err := pool.QueryRow(ctx,
		`SELECT details::text FROM audit_log WHERE aktion = 'RETENTION'`).Scan(&details); err != nil {
		t.Fatalf("Meta-Eintrag fehlt: %v", err)
	}
	for _, muss := range []string{`"geloescht_audit_log": 1`, `"geloescht_admin_log": 1`, `"frist_monate": 24`} {
		if !containsJSON(details, muss) {
			t.Errorf("Meta-Eintrag ohne %s: %s", muss, details)
		}
	}

	// Untergrenze: Eine versehentliche 0 in der Einstellung darf nicht alles wegräumen.
	if _, err := pool.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert)
		VALUES ('audit_aufbewahrung_monate', '0')`); err != nil {
		t.Fatalf("Einstellung setzen: %v", err)
	}
	if m := repository.NewBetriebszustandRepository(pool).AuditAufbewahrungMonate(ctx); m != repository.StandardAuditAufbewahrungMonate {
		t.Errorf("Frist bei Fehlkonfiguration 0 = %d, erwartet Vorgabe %d", m, repository.StandardAuditAufbewahrungMonate)
	}
}

// containsJSON prüft tolerant gegen Leerzeichenvarianten der JSON-Ausgabe.
func containsJSON(haystack, needle string) bool {
	entferne := strings.NewReplacer(" ", "")
	return strings.Contains(entferne.Replace(haystack), entferne.Replace(needle))
}
