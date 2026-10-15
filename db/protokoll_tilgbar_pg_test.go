package db

// audit_log trägt keinen Trigger, der Änderungen sperrt: Die Tilgung personenbezogener Angaben
// nach Art. 17 DSGVO überschreibt audit_log.details. Der Test führt beide Zustände vor: Mit dem
// Trigger aus Migration 003 (prevent_audit_log_modification) scheitert die Tilgung, nach
// Migration 083 geht sie durch. Käme ein solcher Trigger wieder, würde der Test rot, bevor die
// Löschung still bricht.

import (
	"context"
	"os"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5"
)

func TestProtokoll_TilgungGehtDurch(t *testing.T) {
	dsn := os.Getenv(pgtest.Variable)
	if dsn == "" {
		t.Skipf("%s nicht gesetzt — DB-Integrationstest übersprungen", pgtest.Variable)
	}
	ctx := context.Background()

	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Admin-Verbindung: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close(ctx) }) //nolint:errcheck
	pool := paritaetsDB(t, ctx, admin, dsn, "bibliothek_test_083_append_only")
	fuehreDateiAus(t, ctx, pool, "../schema.sql")

	// Ein Eintrag im Protokoll, an dem die Tilgung ansetzt.
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, details, akteur)
		VALUES ('schueler', 'UPDATE', gen_random_uuid(), jsonb_build_object('nachname','KLARNAME'), 'USER')`); err != nil {
		t.Fatalf("Audit-Eintrag anlegen: %v", err)
	}

	tilgung := `UPDATE audit_log SET details = jsonb_build_object('anonymisiert', true) WHERE tabelle = 'schueler'`

	// Zustand 1: der Trigger aus Migration 003, der nur Anhängen erlaubt; die Tilgung scheitert.
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

	// Zustand 2: Migration 083 angewendet; danach geht die Tilgung durch.
	fuehreDateiAus(t, ctx, pool, "../migrations/083_audit_log_append_only_aufloesen.sql")
	if _, err := pool.Exec(ctx, tilgung); err != nil {
		t.Fatalf("nach Migration 083 muss die DSGVO-Tilgung durchgehen: %v", err)
	}
}
