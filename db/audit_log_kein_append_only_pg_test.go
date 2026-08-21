package db

// Gate für Migration 083: audit_log darf keinen Änderungen-blockierenden Trigger tragen.
//
// Der Trigger aus Migration 003 (prevent_audit_log_modification) blockiert jedes
// UPDATE/DELETE — und damit die DSGVO-PII-Tilgung, die audit_log.details überschreiben
// MUSS (Art. 17). Der Test führt beide Zustände real vor: Mit dem alten Trigger scheitert
// die Tilgung, nach Migration 083 geht sie durch. Führt jemand den Append-Only-Trigger
// je wieder ein, wird dieser Test rot, bevor die DSGVO-Löschung still bricht.

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestMigration083_AuditLogTilgungGehtDurch(t *testing.T) {
	dsn := os.Getenv(testDBEnvVar)
	if dsn == "" {
		t.Skipf("%s nicht gesetzt — DB-Integrationstest übersprungen", testDBEnvVar)
	}
	ctx := context.Background()

	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Admin-Verbindung: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close(ctx) }) //nolint:errcheck
	pool := paritaetsDB(t, ctx, admin, dsn, "bibliothek_test_083_append_only")
	fuehreDateiAus(t, ctx, pool, "../schema.sql")

	// Einen Audit-Eintrag anlegen, an dem die Tilgung ansetzt.
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, details, akteur)
		VALUES ('schueler', 'UPDATE', gen_random_uuid(), jsonb_build_object('nachname','KLARNAME'), 'USER')`); err != nil {
		t.Fatalf("Audit-Eintrag anlegen: %v", err)
	}

	tilgung := `UPDATE audit_log SET details = jsonb_build_object('anonymisiert', true) WHERE tabelle = 'schueler'`

	// Zustand 1: der alte Append-Only-Trigger (Migration 003) — die Tilgung MUSS scheitern.
	if _, err := pool.Exec(ctx, `
		CREATE OR REPLACE FUNCTION prevent_audit_log_modification() RETURNS TRIGGER AS $$
		BEGIN RAISE EXCEPTION 'Append-Only'; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_log_append_only_trigger
		BEFORE UPDATE OR DELETE ON audit_log
		FOR EACH ROW EXECUTE FUNCTION prevent_audit_log_modification();`); err != nil {
		t.Fatalf("Alt-Trigger anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, tilgung); err == nil {
		t.Fatal("mit dem Append-Only-Trigger MUSS die DSGVO-Tilgung scheitern — sie ging durch")
	} else if !strings.Contains(err.Error(), "Append-Only") {
		t.Fatalf("unerwarteter Fehler (nicht der Trigger): %v", err)
	}

	// Zustand 2: Migration 083 anwenden — danach geht die Tilgung durch.
	fuehreDateiAus(t, ctx, pool, "../migrations/083_audit_log_append_only_aufloesen.sql")
	if _, err := pool.Exec(ctx, tilgung); err != nil {
		t.Fatalf("nach Migration 083 muss die DSGVO-Tilgung durchgehen: %v", err)
	}
}
