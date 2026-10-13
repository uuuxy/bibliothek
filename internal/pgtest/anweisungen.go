package pgtest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InTx führt f in einer eigenen Transaktion aus und schließt sie ab: für Schreiber, die eine
// Transaktion des Aufrufers verlangen.
func InTx(t *testing.T, pool *pgxpool.Pool, f func(tx pgx.Tx) error) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := f(tx); err != nil {
		_ = tx.Rollback(ctx) //nolint:errcheck
		t.Fatalf("in Transaktion: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

// ZaehleZeilen liest eine Zählung.
func ZaehleZeilen(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("Zählung: %v", err)
	}
	return n
}

// RaeumeAuf führt eine Anweisung zum Aufräumen aus und schreibt einen Fehler ins Protokoll des
// Tests, statt ihn zu verschlucken.
func RaeumeAuf(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Logf("Aufraeumen fehlgeschlagen (%s): %v", sql, err)
	}
}
