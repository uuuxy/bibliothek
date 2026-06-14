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

const (
	createSchemaMigrationsTableSQL = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     VARCHAR(255) PRIMARY KEY,
			applied_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`
	checkMigrationExistsSQL = "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)"
	insertMigrationSQL      = "INSERT INTO schema_migrations (version) VALUES ($1)"
)

// RunMigrations applies all pending SQL migration files from the given directory.
// It creates a `schema_migrations` tracking table on first run and executes each
// *.sql file exactly once, in filename order (lexicographic = numeric prefix order).
// Each migration runs in its own transaction; on failure the run is aborted and
// the error is returned — the database is left in its last-successfully-migrated state.
func (d *Database) RunMigrations(ctx context.Context, migrationsDir string) error {
	// 1. Ensure the tracking table exists (idempotent)
	_, err := d.Pool.Exec(ctx, createSchemaMigrationsTableSQL)
	if err != nil {
		return fmt.Errorf("migrations: failed to create schema_migrations table: %w", err)
	}

	// 2. Collect all *.sql files, sorted by name
	var files []string
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
			return nil
		}
		return fmt.Errorf("migrations: failed to scan directory: %w", err)
	}
	sort.Strings(files)

	// 3. Apply each file that hasn't been recorded yet
	applied := 0
	for _, path := range files {
		version := filepath.Base(path)

		var exists bool
		err = d.Pool.QueryRow(ctx, checkMigrationExistsSQL, version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("migrations: failed to check version %q: %w", version, err)
		}
		if exists {
			continue // already applied
		}

		// Read SQL content
		// #nosec G304 - path comes from filepath.WalkDir of a trusted directory
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("migrations: failed to read %q: %w", version, err)
		}

		// Execute in a transaction for atomicity
		tx, err := d.Pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("migrations: failed to begin transaction for %q: %w", version, err)
		}

		if _, err = tx.Exec(ctx, string(content)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrations: failed to apply %q: %w", version, err)
		}

		if _, err = tx.Exec(ctx, insertMigrationSQL, version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrations: failed to record version %q: %w", version, err)
		}

		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("migrations: failed to commit migration %q: %w", version, err)
		}

		log.Printf("Migrations: applied %s", version)
		applied++
	}

	if applied == 0 {
		log.Println("Migrations: all up to date.")
	} else {
		log.Printf("Migrations: %d migration(s) applied successfully.", applied)
	}
	return nil
}
