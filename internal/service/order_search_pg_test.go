package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestSearchLocalOrders_LiefertSignatur belegt, dass die lokale Bestellsuche die
// Regalsignatur eines bereits katalogisierten Titels mitliefert — Voraussetzung
// dafür, dass der Bestellkorb sie anzeigen und "die vorhandene Systematik
// übernehmen" statt sie zu verschweigen. Gated auf TEST_DATABASE_URL, siehe
// [[pg-integration-test-workflow]].
func TestSearchLocalOrders_LiefertSignatur(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL nicht gesetzt — DB-Integrationstest übersprungen")
	}
	ctx := context.Background()

	lc, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Verbindung für Advisory-Lock: %v", err)
	}
	defer lc.Close(ctx) //nolint:errcheck
	if _, err := lc.Exec(ctx, "SELECT pg_advisory_lock($1)", int64(0x42DB0001)); err != nil {
		t.Fatalf("Advisory-Lock: %v", err)
	}
	defer lc.Exec(ctx, "SELECT pg_advisory_unlock($1)", int64(0x42DB0001)) //nolint:errcheck

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	defer pool.Close()

	var dbName string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&dbName); err != nil {
		t.Fatalf("current_database: %v", err)
	}
	if !strings.Contains(strings.ToLower(dbName), "test") {
		t.Fatalf("Sicherheitsabbruch: Datenbank %q enthält nicht \"test\"", dbName)
	}
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("Schema-Reset: %v", err)
	}
	schemaSQL, err := os.ReadFile(filepath.Join("..", "..", "schema.sql"))
	if err != nil {
		t.Fatalf("schema.sql lesen: %v", err)
	}
	if _, err := pool.Exec(ctx, string(schemaSQL)); err != nil {
		t.Fatalf("schema.sql laden: %v", err)
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO buecher_titel (titel, autor, isbn, signatur) VALUES ($1, $2, $3, $4)`,
		"Effi Briest", "Fontane, Theodor", "9783150001", "Pg"); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	results := searchLocalOrders(ctx, pool, "Effi Briest")
	if len(results) != 1 {
		t.Fatalf("Treffer = %d, want 1", len(results))
	}
	if results[0].Signatur != "Pg" {
		t.Errorf("Signatur = %q, want %q", results[0].Signatur, "Pg")
	}
	if results[0].Source != "local" {
		t.Errorf("Source = %q, want %q", results[0].Source, "local")
	}
}
