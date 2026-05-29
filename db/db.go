package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Database wraps the pgxpool.Pool to provide access to database operations.
type Database struct {
	Pool *pgxpool.Pool
}

// Connect establishes a connection pool to the PostgreSQL database using the provided DSN.
// It configures connection pool limits and performs a ping check to verify connectivity.
func Connect(ctx context.Context, dsn string) (*Database, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	// Configure pool size and connection lifetimes for optimal performance
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 15 * time.Minute

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
