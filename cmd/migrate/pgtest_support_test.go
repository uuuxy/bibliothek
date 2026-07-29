package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Tests in diesem Paket laufen gegen ECHTES PostgreSQL, und das ist keine Kür:
// Der Fehler, den sie absichern, existiert ausschließlich dort. Postgres versetzt eine
// Transaktion beim ersten Fehler in den Abbruchzustand (SQLSTATE 25P02) — ein Mock kennt
// diesen Zustand nicht und lässt jedes `continue` in der Schleife plausibel aussehen.
// Genau deshalb ist der Datenverlust in cmd/migrate jahrelang unbemerkt geblieben.
//
// Ohne TEST_DATABASE_URL werden sie übersprungen. In CI setzt der Workflow die Variable
// auf einen Postgres-Service-Container.

const testDBEnvVar = "TEST_DATABASE_URL"

// testDBLockKey serialisiert die Test-DB-Nutzung über db/, repository/, api/ und
// cmd/migrate/ — alle teilen sich EINE Test-DB, und `go test ./...` startet ihre Binaries
// parallel. Ohne den Lock kollidieren gleichzeitige DROP SCHEMA (Deadlock). Wert identisch
// in allen Paketen halten.
const testDBLockKey int64 = 0x42DB0001

var (
	pgTestOnce sync.Once
	pgTestDB   *pgxpool.Pool
	pgTestErr  error
	lockConn   *pgx.Conn // hält den Lock bis Prozessende
)

// pgTestPool liefert den gemeinsamen Test-Pool mit geladenem schema.sql.
func pgTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv(testDBEnvVar)
	if dsn == "" {
		t.Skipf("%s nicht gesetzt — DB-Integrationstest übersprungen", testDBEnvVar)
	}

	pgTestOnce.Do(func() { pgTestDB, pgTestErr = baueTestDB(dsn) })
	if pgTestErr != nil {
		t.Fatalf("Test-DB konnte nicht vorbereitet werden: %v", pgTestErr)
	}
	return pgTestDB
}

// schemaPfad wird über runtime.Caller aufgelöst statt relativ zum Arbeitsverzeichnis:
// die Tests wechseln per t.Chdir in ein temporäres Verzeichnis, damit das Fehlerprotokoll
// nicht im Repository landet. Ein relativer Pfad wäre danach kaputt — je nach
// Ausführungsreihenfolge, also mal grün und mal rot.
func schemaPfad() string {
	_, dieseDatei, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(dieseDatei), "..", "..", "schema.sql")
}

func baueTestDB(dsn string) (*pgxpool.Pool, error) {
	ctx := context.Background()

	lc, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if _, err := lc.Exec(ctx, "SELECT pg_advisory_lock($1)", testDBLockKey); err != nil {
		return nil, err
	}
	lockConn = lc

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	if err := pruefeTestDatenbank(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		return nil, err
	}
	sqlText, err := os.ReadFile(schemaPfad())
	if err != nil {
		return nil, err
	}
	if _, err := pool.Exec(ctx, string(sqlText)); err != nil {
		return nil, err
	}
	return pool, nil
}

// pruefeTestDatenbank ist die Notbremse vor dem DROP SCHEMA: Zeigt TEST_DATABASE_URL
// versehentlich auf eine echte Datenbank, wäre das ein Totalverlust.
func pruefeTestDatenbank(ctx context.Context, pool *pgxpool.Pool) error {
	var name string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		return err
	}
	if !strings.Contains(strings.ToLower(name), "test") {
		return fmt.Errorf(
			"Sicherheitsabbruch: Datenbank %q enthält nicht \"test\". Diese Tests verwerfen das "+
				"gesamte Schema — %s darf nur auf eine Wegwerf-Datenbank zeigen", name, testDBEnvVar)
	}
	return nil
}

// leereBestand setzt Titel und Exemplare zurück. Anders als in db/ ist kein Rollback um
// den Testfall möglich: insertBatch öffnet seine eigene Transaktion, und genau deren
// COMMIT ist hier der Prüfgegenstand.
func leereBestand(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`TRUNCATE buecher_exemplare, buecher_titel RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("Bestand konnte nicht geleert werden: %v", err)
	}
}

// testLogger legt das Fehlerprotokoll in einem temporären Verzeichnis an und liefert eine
// Funktion, die es geleert zurückliest — das Protokoll ist gepuffert, ungeflusht steht
// dort nichts.
func testLogger(t *testing.T) (*errLogger, func() string) {
	t.Helper()
	pfad := filepath.Join(t.TempDir(), "migration_errors.log")
	el, err := newErrLoggerAt(pfad)
	if err != nil {
		t.Fatalf("Fehlerprotokoll konnte nicht angelegt werden: %v", err)
	}
	t.Cleanup(el.close)
	return el, func() string {
		if err := el.w.Flush(); err != nil {
			t.Fatalf("Fehlerprotokoll konnte nicht geschrieben werden: %v", err)
		}
		b, err := os.ReadFile(pfad) // #nosec G304 - Pfad aus t.TempDir()
		if err != nil {
			t.Fatalf("Fehlerprotokoll konnte nicht gelesen werden: %v", err)
		}
		return string(b)
	}
}
