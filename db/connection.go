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

// PgxPoolIface definiert die Methoden aus pgxpool.Pool, um Mocking zu ermöglichen.
type PgxPoolIface interface {
	Close()
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Ping(ctx context.Context) error
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

// Database kapselt das PgxPoolIface, um Zugriff auf Datenbankoperationen bereitzustellen.
type Database struct {
	Pool PgxPoolIface
}

// Connect stellt einen Verbindungspool zur PostgreSQL-Datenbank über den bereitgestellten DSN her.
// Es konfiguriert die Limits des Verbindungspools und führt einen Ping-Check durch, um die Konnektivität zu überprüfen.
func Connect(ctx context.Context, dsn string) (*Database, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	// Pool-Größe und Verbindungslebensdauer für hohen Nebenläufigkeitsbetrieb (8-PC) konfigurieren.
	// 50 maximale Verbindungen bieten genug Spielraum für 8 parallele Scanner-Clients plus
	// Hintergrundjobs (Mahnwesen, DSGVO-Cron, SSE) unter Spitzenlast.
	config.MaxConns = 50
	config.MinConns = 10
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 15 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Datenbank anpingen, um aktive Verbindung zu verifizieren
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{Pool: pool}, nil
}

// Close schließt ordnungsgemäß alle Verbindungen im Pool.
func (db *Database) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// SafeRollback rollt eine Transaktion zurück und ignoriert das harmlose pgx.ErrTxClosed,
// das auftritt, wenn die Transaktion bereits committet (oder zurückgerollt) wurde.
// Unerwartete Fehler werden geloggt. Gedacht für den Einsatz als `defer db.SafeRollback(ctx, tx)`.
func SafeRollback(ctx context.Context, tx pgx.Tx) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		log.Printf("db: transaction rollback failed: %v", err)
	}
}
