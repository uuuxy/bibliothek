package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Integrationstests gegen echtes Postgres (gated auf TEST_DATABASE_URL, wie im
// db-Paket). Lokal ohne die Variable werden sie übersprungen; in CI setzt der
// Workflow sie auf den Postgres-Service-Container.
//
// Warum echtes PG und nicht pgxmock: Die Inventur-Session-Logik lebt fast vollständig
// im SQL (Scope-Bedingungen, partielle Unique-Indizes, Verlust-UPDATE mit
// Erfassungs-Join). pgxmock würde nur nachgespielte Antworten prüfen, nicht die
// eigentliche Korrektheit — genau die Lücke, um die es hier geht.

const testDBEnvVar = "TEST_DATABASE_URL"

var (
	pgOnce sync.Once
	pgPool *pgxpool.Pool
	pgErr  error
)

func pgTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv(testDBEnvVar)
	if dsn == "" {
		t.Skipf("%s nicht gesetzt — DB-Integrationstest übersprungen", testDBEnvVar)
	}
	pgOnce.Do(func() { pgPool, pgErr = baueRepoTestDB(dsn) })
	if pgErr != nil {
		t.Fatalf("Test-DB konnte nicht vorbereitet werden: %v", pgErr)
	}
	return pgPool
}

func baueRepoTestDB(dsn string) (*pgxpool.Pool, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	// Notbremse vor DROP SCHEMA: nur auf einer Wegwerf-"test"-Datenbank arbeiten.
	var name string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		return nil, err
	}
	if !strings.Contains(strings.ToLower(name), "test") {
		return nil, fmt.Errorf("Sicherheitsabbruch: Datenbank %q enthält nicht \"test\"", name)
	}

	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		return nil, err
	}
	sql, err := os.ReadFile(filepath.Join("..", "schema.sql"))
	if err != nil {
		return nil, err
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		return nil, err
	}
	return pool, nil
}

// resetInventurDaten räumt zwischen Tests die Bestands-, Ausleih- und Personendaten
// leer, damit jeder Test von einer bekannten Basis startet (schema-Load passiert nur
// einmal). CASCADE räumt abhängige Zeilen (u. a. schadensfaelle) mit.
func resetInventurDaten(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE inventur_erfassungen, inventur_sessions, schadensfaelle, ausleihen,
		         buecher_exemplare, buecher_titel, signatures, schueler, benutzer
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("Reset der Testdaten fehlgeschlagen: %v", err)
	}
}

// seedSignaturMitExemplaren legt eine Signatur mit n ausleihbaren Exemplaren an und
// liefert Signatur-ID und die Exemplar-IDs.
func seedSignaturMitExemplaren(t *testing.T, pool *pgxpool.Pool, name string, n int) (int, []string) {
	t.Helper()
	ctx := context.Background()

	var sigID int
	if err := pool.QueryRow(ctx, `INSERT INTO signatures (name) VALUES ($1) RETURNING id`, name).Scan(&sigID); err != nil {
		t.Fatalf("Signatur %q anlegen: %v", name, err)
	}

	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		var titelID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_titel (titel, signature_id) VALUES ($1, $2) RETURNING id`,
			fmt.Sprintf("%s-Buch-%d", name, i), sigID).Scan(&titelID); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
		var exID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`,
			titelID, fmt.Sprintf("BC-%s-%d", name, i)).Scan(&exID); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		ids = append(ids, exID)
	}
	return sigID, ids
}
