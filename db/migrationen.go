package db

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FuehreMigrationenAus spielt jede SQL-Datei aus verzeichnis genau einmal ein, in der
// Reihenfolge ihrer Namen (die Nummer vorn). Welche schon gelaufen sind, steht in
// schema_migrations. Jede Datei läuft in einer eigenen Transaktion; scheitert eine, bricht der
// Lauf ab, und die Datenbank bleibt auf dem Stand der letzten, die gelang.
func (v *Verbindung) FuehreMigrationenAus(ctx context.Context, verzeichnis string) error {
	if err := v.spieleSchemaEin(ctx); err != nil {
		return err
	}
	if err := v.legeMigrationstabelleAn(ctx); err != nil {
		return err
	}

	dateien, fehlt, err := findeMigrationsdateien(verzeichnis)
	if err != nil {
		return err
	}
	if fehlt {
		return nil
	}

	eingespielt := 0
	for _, pfad := range dateien {
		neu, err := v.spieleMigrationEin(ctx, pfad)
		if err != nil {
			return err
		}
		if neu {
			eingespielt++
		}
	}

	if eingespielt == 0 {
		slog.Info("Migrations: all up to date.")
	} else {
		slog.Info("Migrations applied successfully", "anzahl", eingespielt)
	}
	return nil
}

// spieleSchemaEin spielt in eine frische Datenbank (ohne schema_migrations) schema.sql als
// Ausgangsstand ein. Fehlt schema.sql, steht nur eine Warnung im Log.
func (v *Verbindung) spieleSchemaEin(ctx context.Context) error {
	var hatTabelle bool
	err := v.Pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'schema_migrations')").Scan(&hatTabelle)
	if err != nil {
		return fmt.Errorf("migrations: failed to check for schema_migrations: %w", err)
	}
	if hatTabelle {
		return nil
	}

	slog.Info("Migrations: Fresh database detected. Applying schema.sql as baseline...")
	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		slog.Warn("Migrations: schema.sql not found", "fehler", err)
		return nil
	}
	if _, err := v.Pool.Exec(ctx, string(schema)); err != nil {
		return fmt.Errorf("migrations: failed to apply schema.sql: %w", err)
	}
	slog.Info("Migrations: schema.sql applied successfully.")
	return nil
}

// legeMigrationstabelleAn legt schema_migrations an, wenn es sie noch nicht gibt.
func (v *Verbindung) legeMigrationstabelleAn(ctx context.Context) error {
	_, err := v.Pool.Exec(ctx, `
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

// findeMigrationsdateien liefert alle SQL-Dateien unter verzeichnis, nach Namen sortiert. Gibt
// es das Verzeichnis nicht, ist fehlt wahr und der Lauf entfällt; das ist kein Fehler.
func findeMigrationsdateien(verzeichnis string) (dateien []string, fehlt bool, err error) {
	err = filepath.WalkDir(verzeichnis, func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".sql") {
			dateien = append(dateien, pfad)
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("Migrations: directory not found — skipping", "verzeichnis", verzeichnis)
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("migrations: failed to scan directory: %w", err)
	}
	sort.Strings(dateien)
	return dateien, false, nil
}

// spieleMigrationEin spielt eine Datei in einer eigenen Transaktion ein, wenn ihre Version noch
// nicht in schema_migrations steht. neu ist falsch, wenn sie schon gelaufen war.
func (v *Verbindung) spieleMigrationEin(ctx context.Context, pfad string) (neu bool, err error) {
	version := filepath.Base(pfad)

	var vorhanden bool
	if err := v.Pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version,
	).Scan(&vorhanden); err != nil {
		return false, fmt.Errorf("migrations: failed to check version %q: %w", version, err)
	}
	if vorhanden {
		return false, nil
	}

	// #nosec G304 - path comes from filepath.WalkDir of a trusted directory
	inhalt, err := os.ReadFile(pfad)
	if err != nil {
		return false, fmt.Errorf("migrations: failed to read %q: %w", version, err)
	}

	tx, err := v.Pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("migrations: failed to begin transaction for %q: %w", version, err)
	}

	if _, err := tx.Exec(ctx, string(inhalt)); err != nil {
		Verwerfe(ctx, tx)
		return false, fmt.Errorf("migrations: failed to apply %q: %w", version, err)
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO schema_migrations (version) VALUES ($1)", version,
	); err != nil {
		Verwerfe(ctx, tx)
		return false, fmt.Errorf("migrations: failed to record version %q: %w", version, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("migrations: failed to commit migration %q: %w", version, err)
	}

	slog.Info("Migrations: applied", "version", version)
	return true, nil
}
