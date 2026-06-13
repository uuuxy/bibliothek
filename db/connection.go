package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxPoolIface defines the set of methods used from pgxpool.Pool to allow for mocking.
type PgxPoolIface interface {
	Close()
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Ping(ctx context.Context) error
}

// Database wraps the PgxPoolIface to provide access to database operations.
type Database struct {
	Pool PgxPoolIface
}

// Connect establishes a connection pool to the PostgreSQL database using the provided DSN.
// It configures connection pool limits and performs a ping check to verify connectivity.
func Connect(ctx context.Context, dsn string) (*Database, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	// Configure pool size and connection lifetimes for 8-PC high-concurrency operation.
	// 50 max connections provide enough headroom for 8 parallel scanning clients plus
	// background jobs (mahnwesen, GDPR cron, SSE) under peak load.
	config.MaxConns = 50
	config.MinConns = 10
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 15 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Ping database to verify active connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{Pool: pool}, nil
}

// Close gracefully closes all connections in the pool.
func (db *Database) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}
