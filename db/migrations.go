package db

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations applies all pending SQL migration files from the given directory.
// It creates a `schema_migrations` tracking table on first run and executes each
// *.sql file exactly once, in filename order (lexicographic = numeric prefix order).
// Each migration runs in its own transaction; on failure the run is aborted and
// the error is returned — the database is left in its last-successfully-migrated state.
func (d *Database) RunMigrations(ctx context.Context, migrationsDir string) error {
	// 0. Fresh-database-Erkennung + schema.sql-Baseline
	if err := d.ensureBaselineSchema(ctx); err != nil {
		return err
	}

	// 1. Ensure the tracking table exists (idempotent)
	if err := d.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	// 2. Collect all *.sql files, sorted by name
	files, skip, err := collectMigrationFiles(migrationsDir)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}

	// 3. Apply each file that hasn't been recorded yet
	applied := 0
	for _, path := range files {
		didApply, err := d.applyMigration(ctx, path)
		if err != nil {
			return err
		}
		if didApply {
			applied++
		}
	}

	if applied == 0 {
		log.Println("Migrations: all up to date.")
	} else {
		log.Printf("Migrations: %d migration(s) applied successfully.", applied)
	}
	return nil
}

// ensureBaselineSchema erkennt eine frische Datenbank (fehlende schema_migrations-
// Tabelle) und spielt in diesem Fall schema.sql als Ausgangsbasis ein. Eine fehlende
// schema.sql ist kein Fehler (nur eine Warnung).
func (d *Database) ensureBaselineSchema(ctx context.Context) error {
	var hasMigrationsTable bool
	err := d.Pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'schema_migrations')").Scan(&hasMigrationsTable)
	if err != nil {
		return fmt.Errorf("migrations: failed to check for schema_migrations: %w", err)
	}
	if hasMigrationsTable {
		return nil
	}

	log.Println("Migrations: Fresh database detected. Applying schema.sql as baseline...")
	schemaBytes, err := os.ReadFile("schema.sql")
	if err != nil {
		log.Printf("Migrations: warning: schema.sql not found: %v", err)
		return nil
	}
	if _, err := d.Pool.Exec(ctx, string(schemaBytes)); err != nil {
		return fmt.Errorf("migrations: failed to apply schema.sql: %w", err)
	}
	log.Println("Migrations: schema.sql applied successfully.")
	return nil
}

// ensureMigrationsTable legt die Tracking-Tabelle schema_migrations idempotent an.
func (d *Database) ensureMigrationsTable(ctx context.Context) error {
	_, err := d.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     VARCHAR(255) PRIMARY KEY,
			applied_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("migrations: failed to create schema_migrations table: %w", err)
	}
	return nil
}

// collectMigrationFiles sammelt alle *.sql-Dateien unterhalb von migrationsDir in
// lexikographischer Reihenfolge. skip=true bedeutet: Verzeichnis existiert nicht
// (kein Fehler; der Migrationslauf wird dann übersprungen).
func collectMigrationFiles(migrationsDir string) (files []string, skip bool, err error) {
	err = filepath.WalkDir(migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".sql") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Migrations: directory %q not found — skipping.", migrationsDir)
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("migrations: failed to scan directory: %w", err)
	}
	sort.Strings(files)
	return files, false, nil
}

// applyMigration wendet eine einzelne Migrationsdatei an, sofern ihre Version noch
// nicht in schema_migrations vermerkt ist. didApply=false heißt: bereits angewendet.
// Jede Migration läuft in einer eigenen Transaktion (Atomarität pro Datei).
func (d *Database) applyMigration(ctx context.Context, path string) (didApply bool, err error) {
	version := filepath.Base(path)

	var exists bool
	if err := d.Pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("migrations: failed to check version %q: %w", version, err)
	}
	if exists {
		return false, nil // already applied
	}

	// Read SQL content
	// #nosec G304 - path comes from filepath.WalkDir of a trusted directory
	content, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("migrations: failed to read %q: %w", version, err)
	}

	// Execute in a transaction for atomicity
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("migrations: failed to begin transaction for %q: %w", version, err)
	}

	if _, err := tx.Exec(ctx, string(content)); err != nil {
		SafeRollback(ctx, tx)
		return false, fmt.Errorf("migrations: failed to apply %q: %w", version, err)
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO schema_migrations (version) VALUES ($1)", version,
	); err != nil {
		SafeRollback(ctx, tx)
		return false, fmt.Errorf("migrations: failed to record version %q: %w", version, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("migrations: failed to commit migration %q: %w", version, err)
	}

	log.Printf("Migrations: applied %s", version)
	return true, nil
}
