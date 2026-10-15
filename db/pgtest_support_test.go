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

// Diese Tests prüfen die Zusagen der Datenbank (🟢 im Katalog der Invarianten) an einem echten
// PostgreSQL: Sie verletzen jede und erwarten den passenden Fehler der Bedingung. Mit pgxmock geht
// das nicht; dort gibt es keine Bedingungen, nur nachgespielte Antworten.
//
// Ein Beispiel ist die Bedingung an der Aussonderung: Sie ließ ist_ausgesondert=true mit
// grund=NULL durch ("TRUE AND (NULL IN (...))" ist NULL, und ein CHECK schlägt nur bei FALSE an),
// obwohl sie im Text richtig aussah (bedingung_aussonderung_pg_test.go).
//
// Pool, Advisory-Lock und Notbremse stehen in internal/pgtest; hier stehen Transaktion und
// Erwartungen, die Testdaten in pgtest_fixtures_test.go, die Fälle in bedingung_*_pg_test.go.
func pgTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return pgtest.Pool(t)
}

// inTx führt fn in einer Transaktion aus, die danach immer zurückgerollt wird. So bleibt jeder
// Fall unabhängig, ohne die Datenbank zwischen den Fällen neu aufzubauen.
func inTx(t *testing.T, pool *pgxpool.Pool, fn func(tx pgx.Tx)) {
	t.Helper()

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Transaktion konnte nicht gestartet werden: %v", err)
	}
	// Zurückrollen ist hier der gewöhnliche Fall, nicht der Fehler.
	defer Verwerfe(ctx, tx)

	fn(tx)
}

// erwarteVerletzteBedingung führt sql aus und verlangt, dass Postgres es mit genau der erwarteten
// Bedingung ablehnt. Gelingt die Anweisung, ist das ein Fehler des Tests: Dann fehlt der Schutz,
// den der Katalog behauptet.
//
// Die Anweisung läuft in einem Savepoint (tx.Begin auf einer laufenden Transaktion); ein Fehler
// versetzte die Transaktion sonst in den Abbruchzustand (SQLSTATE 25P02), und jede folgende
// Gegenprobe im selben Fall scheiterte nur noch daran.
func erwarteVerletzteBedingung(t *testing.T, tx pgx.Tx, constraint, sql string, args ...any) {
	t.Helper()

	ctx := context.Background()
	sp, err := tx.Begin(ctx)
	if err != nil {
		t.Fatalf("Savepoint konnte nicht gesetzt werden: %v", err)
	}
	defer Verwerfe(ctx, sp)

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

// erwarteErfolg stellt sicher, dass ein gültiger Wert durchgeht. Ohne diese Gegenprobe könnte eine
// zu strenge Bedingung unbemerkt den Betrieb aufhalten.
func erwarteErfolg(t *testing.T, tx pgx.Tx, was, sql string, args ...any) {
	t.Helper()

	if _, err := tx.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: gültiger Wert wurde abgelehnt: %v", was, err)
	}
}
