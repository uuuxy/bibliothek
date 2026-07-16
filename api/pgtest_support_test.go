package api

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

// PG-Integrationstests fürs api-Paket (gated auf TEST_DATABASE_URL, wie db/ und
// repository/). Nötig für die order-/graduates-Bugs, deren Kern in SQL-Filtern liegt
// (bereits bestellte Exemplare, numerische Barcode-Sortierung, Abgänger-Filter) —
// pgxmock würde nur nachgespielte Antworten prüfen, nicht die SQL-Korrektheit.

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
	pgOnce.Do(func() { pgPool, pgErr = baueAPITestDB(dsn) })
	if pgErr != nil {
		t.Fatalf("Test-DB konnte nicht vorbereitet werden: %v", pgErr)
	}
	return pgPool
}

func baueAPITestDB(dsn string) (*pgxpool.Pool, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

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

// resetBestandsdaten leert Bestands-, Bestell- und Personendaten zwischen Tests.
func resetBestandsdaten(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE buecher_exemplare, buecher_titel, ausleihen, schueler, benutzer
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("Reset fehlgeschlagen: %v", err)
	}
}

// titelMitMeldebestand legt einen Titel mit gegebenem Meldebestand an.
func titelMitMeldebestand(t *testing.T, pool *pgxpool.Pool, titel string, meldebestand int) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel, meldebestand) VALUES ($1, $2) RETURNING id`,
		titel, meldebestand).Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	return id
}

// exemplar legt ein Exemplar mit Verleih-/Aussonderungsstatus und Notiz an.
func exemplar(t *testing.T, pool *pgxpool.Pool, titelID, barcode string, ausleihbar bool, notiz string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		titelID, barcode, ausleihbar, notiz).Scan(&id); err != nil {
		t.Fatalf("Exemplar %q anlegen: %v", barcode, err)
	}
	return id
}
