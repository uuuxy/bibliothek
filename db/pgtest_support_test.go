package db

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Diese Tests prüfen die DB-Invarianten (🟢 im Invarianten-Katalog) gegen eine ECHTE
// PostgreSQL-Instanz: Sie provozieren jede Verletzung und erwarten den passenden
// Constraint-Fehler. Unit-Tests mit pgxmock können das prinzipiell nicht — dort gibt
// es keine Constraints, nur nachgespielte Antworten.
//
// Warum das nötig war: Der CHECK aus Migration 043 sah zunächst korrekt aus, liess
// aber ist_ausgesondert=true mit grund=NULL durch ("TRUE AND (NULL IN (...))" = NULL,
// und ein CHECK schlägt nur bei FALSE an). Gefunden hat das erst ein Lauf gegen echtes
// Postgres — festgehalten in constraints_aussonderung_pg_test.go.
//
// Aufteilung: Pool-Aufbau, Advisory-Lock und Notbremse liegen seit dem 31.08.2026 in
// internal/pgtest (vorher fünffach kopiert); dieses Modul enthält die Mechanik
// (Transaktion, Erwartungen), die Fixtures liegen in pgtest_fixtures_test.go, die
// Fälle in constraints_*_pg_test.go.
func pgTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return pgtest.Pool(t)
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
