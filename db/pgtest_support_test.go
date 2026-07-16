package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Diese Tests prüfen die DB-Invarianten (🟢 im Invarianten-Katalog) gegen eine ECHTE
// PostgreSQL-Instanz: Sie provozieren jede Verletzung und erwarten den passenden
// Constraint-Fehler. Unit-Tests mit pgxmock können das prinzipiell nicht — dort gibt
// es keine Constraints, nur nachgespielte Antworten.
//
// Lokal ohne TEST_DATABASE_URL werden sie übersprungen. In CI setzt der Workflow die
// Variable auf einen Postgres-Service-Container.
//
// Warum das nötig war: Der CHECK aus Migration 043 sah zunächst korrekt aus, liess
// aber ist_ausgesondert=true mit grund=NULL durch ("TRUE AND (NULL IN (...))" = NULL,
// und ein CHECK schlägt nur bei FALSE an). Gefunden hat das erst ein Lauf gegen echtes
// Postgres — festgehalten in constraints_aussonderung_pg_test.go.
//
// Aufteilung: dieses Modul enthält die Mechanik (Pool, Transaktion, Erwartungen),
// die Fixtures liegen in pgtest_fixtures_test.go, die Fälle in constraints_*_pg_test.go.

const testDBEnvVar = "TEST_DATABASE_URL"

// testDBLockKey serialisiert die Test-DB-Nutzung über db/, repository/ und api/ —
// alle teilen sich EINE Test-DB, und `go test ./...` startet ihre Binaries parallel.
// Ohne den Lock kollidieren gleichzeitige DROP SCHEMA (Deadlock). Wert identisch in
// allen drei Paketen halten.
const testDBLockKey int64 = 0x42DB0001

// Pool und Schema werden prozessweit genau einmal aufgebaut: schema.sql ist nicht
// idempotent (CREATE TYPE bricht beim zweiten Lauf ab), und jeder Test räumt ohnehin
// per Transaktions-Rollback hinter sich auf.
var (
	pgTestOnce sync.Once
	pgTestDB   *pgxpool.Pool
	pgTestErr  error
	// lockConn hält den paket-übergreifenden Lock bis Prozessende.
	lockConn *pgx.Conn
)

// pgTestPool liefert den gemeinsamen Test-Pool mit geladenem schema.sql. Ohne
// TEST_DATABASE_URL wird der Test übersprungen (lokal ist kein Postgres Pflicht).
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

// baueTestDB verbindet zur Test-DB und spielt schema.sql ein — den Zustand, den auch
// eine Neuinstallation erhält (inkl. aller Constraints aus den Migrationen 039–044).
//
// Das Schema wird vorher verworfen, denn schema.sql ist nicht idempotent (CREATE TYPE
// bricht beim zweiten Lauf ab). Ohne diesen Reset wäre der Test nur gegen eine
// jungfräuliche DB grün — beim zweiten `go test` gegen dieselbe DB schlüge er fehl.
func baueTestDB(dsn string) (*pgxpool.Pool, error) {
	ctx := context.Background()

	lc, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if _, err := lc.Exec(ctx, "SELECT pg_advisory_lock($1)", testDBLockKey); err != nil {
		return nil, err
	}
	lockConn = lc // offen halten bis Prozessende

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

	// Reset auf leeres Schema — sicher, weil pruefeTestDatenbank oben abgesichert hat,
	// dass wir NICHT auf einer produktiven Datenbank arbeiten.
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

// pruefeTestDatenbank ist die Notbremse vor dem DROP SCHEMA: Diese Tests löschen das
// gesamte Schema. Zeigt TEST_DATABASE_URL versehentlich auf eine echte Datenbank,
// wäre das ein Totalverlust. Deshalb muss der Datenbankname „test" enthalten.
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

// inTx führt fn in einer Transaktion aus, die anschliessend IMMER zurückgerollt wird.
// So bleibt jeder Testfall unabhängig, ohne die DB zwischen den Fällen neu aufzubauen.
func inTx(t *testing.T, pool *pgxpool.Pool, fn func(tx pgx.Tx)) {
	t.Helper()

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Transaktion konnte nicht gestartet werden: %v", err)
	}
	// Rollback ist hier der Normalfall, nicht der Fehlerfall.
	defer SafeRollback(ctx, tx)

	fn(tx)
}

// erwarteConstraintVerletzung führt sql aus und verlangt, dass Postgres es mit genau
// dem erwarteten Constraint ablehnt. Ein erfolgreiches Statement ist ein Testfehler:
// dann fehlt der Schutz, den der Katalog behauptet.
//
// Das Statement läuft in einem Savepoint (tx.Begin auf einer laufenden Tx), denn ein
// Fehler versetzt die Transaktion sonst in den Abbruchzustand (SQLSTATE 25P02) und
// jede folgende Gegenprobe im selben Testfall würde nur noch daran scheitern.
func erwarteConstraintVerletzung(t *testing.T, tx pgx.Tx, constraint, sql string, args ...any) {
	t.Helper()

	ctx := context.Background()
	sp, err := tx.Begin(ctx)
	if err != nil {
		t.Fatalf("Savepoint konnte nicht gesetzt werden: %v", err)
	}
	defer SafeRollback(ctx, sp)

	_, err = sp.Exec(ctx, sql, args...)
	if err == nil {
		t.Fatalf("Constraint %q hat NICHT gegriffen — die Verletzung wurde akzeptiert", constraint)
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("Constraint %q: unerwarteter Fehlertyp: %v", constraint, err)
	}
	if pgErr.ConstraintName != constraint {
		t.Fatalf("Constraint %q erwartet, aber %q hat gegriffen (SQLSTATE %s): %s",
			constraint, pgErr.ConstraintName, pgErr.Code, pgErr.Message)
	}
}

// erwarteErfolg stellt sicher, dass ein GÜLTIGER Wert durchgeht. Ohne diese Gegenprobe
// könnte ein zu strenger Constraint unbemerkt den Normalbetrieb blockieren.
func erwarteErfolg(t *testing.T, tx pgx.Tx, was, sql string, args ...any) {
	t.Helper()

	if _, err := tx.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: gültiger Wert wurde abgelehnt: %v", was, err)
	}
}
