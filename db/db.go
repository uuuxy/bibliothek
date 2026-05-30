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

// InitPermissions initializes the role_permissions schema and seeds it with default settings.
func (db *Database) InitPermissions(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS role_permissions (
			role benutzer_rolle NOT NULL,
			permission VARCHAR(100) NOT NULL,
			allowed BOOLEAN NOT NULL DEFAULT false,
			PRIMARY KEY (role, permission)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create role_permissions table: %w", err)
	}

	defaults := []struct {
		Role       string
		Permission string
		Allowed    bool
	}{
		// Admin defaults
		{"admin", "view_students", true},
		{"admin", "create_students", true},
		{"admin", "delete_students", true},
		{"admin", "import_students", true},
		{"admin", "upload_photos", true},
		{"admin", "view_books", true},
		{"admin", "edit_books", true},
		{"admin", "delete_books", true},
		{"admin", "inventory_scan", true},
		{"admin", "view_orders", true},
		{"admin", "create_orders", true},
		{"admin", "view_graduates", true},
		{"admin", "view_stats", true},
		{"admin", "audit_logs", true},
		{"admin", "manage_users", true},

		// Mitarbeiter defaults
		{"mitarbeiter", "view_students", true},
		{"mitarbeiter", "create_students", true},
		{"mitarbeiter", "delete_students", true},
		{"mitarbeiter", "import_students", true},
		{"mitarbeiter", "upload_photos", true},
		{"mitarbeiter", "view_books", true},
		{"mitarbeiter", "edit_books", true},
		{"mitarbeiter", "delete_books", true},
		{"mitarbeiter", "inventory_scan", true},
		{"mitarbeiter", "view_orders", true},
		{"mitarbeiter", "create_orders", true},
		{"mitarbeiter", "view_graduates", true},
		{"mitarbeiter", "view_stats", true},
		{"mitarbeiter", "audit_logs", false},
		{"mitarbeiter", "manage_users", false},

		// Lehrer defaults
		{"lehrer", "view_students", true},
		{"lehrer", "create_students", false},
		{"lehrer", "delete_students", false},
		{"lehrer", "import_students", false},
		{"lehrer", "upload_photos", true},
		{"lehrer", "view_books", true},
		{"lehrer", "edit_books", false},
		{"lehrer", "delete_books", false},
		{"lehrer", "inventory_scan", false},
		{"lehrer", "view_orders", false},
		{"lehrer", "create_orders", false},
		{"lehrer", "view_graduates", false},
		{"lehrer", "view_stats", false},
		{"lehrer", "audit_logs", false},
		{"lehrer", "manage_users", false},
	}

	for _, d := range defaults {
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO role_permissions (role, permission, allowed)
			VALUES ($1, $2, $3)
			ON CONFLICT (role, permission) DO NOTHING
		`, d.Role, d.Permission, d.Allowed)
		if err != nil {
			return fmt.Errorf("failed to seed permission default (%s, %s): %w", d.Role, d.Permission, err)
		}
	}
	return nil
}

// InitLieferanten initializes the lieferanten table and seeds it with default values.
func (db *Database) InitLieferanten(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS lieferanten (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			kundennummer VARCHAR(100) NOT NULL,
			erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create lieferanten table: %w", err)
	}

	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM lieferanten").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query lieferanten count: %w", err)
	}

	if count == 0 {
		defaults := []struct {
			Name         string
			Email        string
			Kundennummer string
		}{
			{"Klett Verlag", "bestellung@klett.de", "K-99281"},
			{"Cornelsen", "service@cornelsen.de", "C-88123"},
			{"Westermann", "order@westermann.de", "W-77441"},
		}

		for _, d := range defaults {
			_, err = db.Pool.Exec(ctx, `
				INSERT INTO lieferanten (name, email, kundennummer)
				VALUES ($1, $2, $3)
			`, d.Name, d.Email, d.Kundennummer)
			if err != nil {
				return fmt.Errorf("failed to seed supplier default (%s): %w", d.Name, err)
			}
		}
	}
	return nil
}
