package inventur

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewDB erstellt einen Pool für PostgreSQL und prüft die Verbindung per Ping.
func NewDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL darf nicht leer sein")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("ungültige database URL: %w", err)
	}

	// Connection Pool Tuning für eine kleine Schulanwendung
	config.MaxConns = 10                        // Max 10 gleichzeitige Verbindungen (Standard: 4 * CPU-Kerne)
	config.MinConns = 2                         // Mindestens 2 Verbindungen warmhalten
	config.MaxConnLifetime = 30 * time.Minute   // Verbindungen nach 30 Min. erneuern (verhindert stale connections)
	config.MaxConnIdleTime = 5 * time.Minute    // Ungenutzte Verbindungen nach 5 Min. freigeben
	config.HealthCheckPeriod = 30 * time.Second // Alle 30 Sek. prüfen, ob Verbindungen noch leben

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("pool konnte nicht erstellt werden: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres nicht erreichbar: %w", err)
	}

	return pool, nil
}
