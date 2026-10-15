// Package db öffnet den Verbindungspool, fährt die Migrationen und legt beim Start an, was das
// Programm braucht: die Rechte je Rolle, das erste Admin-Konto und die Mail-Konfiguration aus
// der Umgebung. Beispieldaten legt der Start nie an, weil sie auf einer echten Anlage von echten
// nicht zu unterscheiden wären; sie kommen nur aus scripts/seed_demo.sql.
package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool sind die Methoden von pgxpool.Pool, die das Programm braucht. Als Schnittstelle lässt
// sich der Pool in Tests durch eine Attrappe ersetzen (pgxmock) oder einwickeln.
type Pool interface {
	Close()
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Ping(ctx context.Context) error
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

// Verbindung ist der Zugang des Programms zur Datenbank; an ihr hängen die Schritte des
// Starts.
type Verbindung struct {
	Pool Pool
}

// Verbinde öffnet den Verbindungspool zur Datenbank unter dsn und prüft mit einem Ping, dass sie
// antwortet.
func Verbinde(ctx context.Context, dsn string) (*Verbindung, error) {
	konfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	// 50 Verbindungen reichen für acht Arbeitsplätze, die gleichzeitig scannen, und die Läufe im
	// Hintergrund (Mahnwesen, Löschjobs, SSE) unter Spitzenlast.
	konfig.MaxConns = 50
	konfig.MinConns = 10
	konfig.MaxConnLifetime = 30 * time.Minute
	konfig.MaxConnIdleTime = 15 * time.Minute
	konfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, konfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Verbindung{Pool: pool}, nil
}

// Close schließt alle Verbindungen des Pools.
func (v *Verbindung) Close() {
	if v.Pool != nil {
		v.Pool.Close()
	}
}

// Verwerfe rollt eine Transaktion zurück, gedacht für `defer db.Verwerfe(ctx, tx)`. Ist die
// Transaktion schon abgeschlossen (pgx.ErrTxClosed), ist das kein Fehler; jeder andere steht
// im Log.
func Verwerfe(ctx context.Context, tx pgx.Tx) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		log.Printf("db: transaction rollback failed: %v", err)
	}
}
