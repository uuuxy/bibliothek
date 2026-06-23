package db

import (
	"context"
	"fmt"
	"log"
	"os"
)

const (
	createPgTrgmExtensionSQL = "CREATE EXTENSION IF NOT EXISTS pg_trgm;"

	createRolePermissionsTableSQL = `
		CREATE TABLE IF NOT EXISTS role_permissions (
			role VARCHAR(50) NOT NULL,
			permission VARCHAR(100) NOT NULL,
			allowed BOOLEAN NOT NULL DEFAULT false,
			PRIMARY KEY (role, permission)
		)
	`

	createBenutzerRollenTableSQL = `
		CREATE TABLE IF NOT EXISTS benutzer_rollen (
			benutzer_id UUID PRIMARY KEY REFERENCES benutzer(id) ON DELETE CASCADE,
			rolle VARCHAR(50) NOT NULL CHECK (rolle IN ('ADMIN', 'MITARBEITER', 'LEHRER', 'HELFER'))
		)
	`

	migrateBenutzerRollenSQL = `
		INSERT INTO benutzer_rollen (benutzer_id, rolle)
		SELECT id, UPPER(rolle::text)
		FROM benutzer
		ON CONFLICT DO NOTHING
	`

	seedRolePermissionSQL = `
		INSERT INTO role_permissions (role, permission, allowed)
		VALUES ($1, $2, $3)
		ON CONFLICT (role, permission) DO NOTHING
	`

	createLieferantenTableSQL = `
		CREATE TABLE IF NOT EXISTS lieferanten (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			kundennummer VARCHAR(100) NOT NULL,
			erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`

	seedLieferantenSQL = `
		INSERT INTO lieferanten (name, email, kundennummer)
		VALUES ($1, $2, $3)
	`

	insertInitialAdminSQL = `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, passwort_hash, rolle, aktiv)
		VALUES ('admin', 'System', 'Administrator', $1, '$2a$10$W65tT30cE2t3e0l25QJ0rO0wE64q.O3v6fQ20K7Y0R2h25n8aUvF6', 'admin', true)
		RETURNING id
	`

	insertInitialAdminRoleSQL = `
		INSERT INTO benutzer_rollen (benutzer_id, rolle)
		VALUES ($1, 'ADMIN')
	`
)

// InitPermissions initializes the role_permissions schema, runs db migrations, and seeds defaults.
func (db *Database) InitPermissions(ctx context.Context) error {
	// 1. Enable pg_trgm extension
	_, err := db.Pool.Exec(ctx, createPgTrgmExtensionSQL)
	if err != nil {
		return fmt.Errorf("failed to create pg_trgm extension: %w", err)
	}

	// 2. Create pg_trgm GIN indexes
	queries := []string{
		"CREATE INDEX IF NOT EXISTS idx_buecher_titel_trgm ON buecher_titel USING gin (titel gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_buecher_autor_trgm ON buecher_titel USING gin (autor gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_buecher_isbn_trgm ON buecher_titel USING gin (isbn gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_schueler_vorname_trgm ON schueler USING gin (vorname gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_schueler_nachname_trgm ON schueler USING gin (nachname gin_trgm_ops);",
	}
	for _, q := range queries {
		if _, err := db.Pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("failed to create GIN index: %w", err)
		}
	}

	// 3. Migrate role_permissions table role column to VARCHAR(50) if it's enum
	var dataType string
	err = db.Pool.QueryRow(ctx, `
		SELECT data_type 
		FROM information_schema.columns 
		WHERE table_name = 'role_permissions' AND column_name = 'role'
	`).Scan(&dataType)
	if err == nil && dataType == "USER-DEFINED" {
		_, err = db.Pool.Exec(ctx, "ALTER TABLE role_permissions ALTER COLUMN role TYPE VARCHAR(50);")
		if err != nil {
			return fmt.Errorf("failed to alter role_permissions.role column type: %w", err)
		}
	}

	// 4. Create role_permissions table
	_, err = db.Pool.Exec(ctx, createRolePermissionsTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create role_permissions table: %w", err)
	}

	// 5. Create benutzer_rollen table
	_, err = db.Pool.Exec(ctx, createBenutzerRollenTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create benutzer_rollen table: %w", err)
	}

	// 6. Migrate existing roles from benutzer table to benutzer_rollen if empty
	var exists int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM benutzer_rollen").Scan(&exists)
	if err == nil && exists == 0 {
		_, err = db.Pool.Exec(ctx, migrateBenutzerRollenSQL)
		if err != nil {
			return fmt.Errorf("failed to migrate existing benutzer roles: %w", err)
		}
	}

	// 7. Seed default role permissions with uppercase role names
	defaults := []struct {
		Role       string
		Permission string
		Allowed    bool
	}{
		// Admin defaults
		{"ADMIN", "view_students", true},
		{"ADMIN", "edit_students", true},
		{"ADMIN", "create_students", true},
		{"ADMIN", "delete_students", true},
		{"ADMIN", "import_students", true},
		{"ADMIN", "upload_photos", true},
		{"ADMIN", "view_books", true},
		{"ADMIN", "edit_books", true},
		{"ADMIN", "delete_books", true},
		{"ADMIN", "inventory_scan", true},
		{"ADMIN", "manage_inventory", true},
		{"ADMIN", "view_orders", true},
		{"ADMIN", "create_orders", true},
		{"ADMIN", "view_graduates", true},
		{"ADMIN", "view_stats", true},
		{"ADMIN", "audit_logs", true},
		{"ADMIN", "manage_users", true},

		// Mitarbeiter defaults
		{"MITARBEITER", "view_students", true},
		{"MITARBEITER", "edit_students", true},
		{"MITARBEITER", "create_students", true},
		{"MITARBEITER", "delete_students", true},
		{"MITARBEITER", "import_students", true},
		{"MITARBEITER", "upload_photos", true},
		{"MITARBEITER", "view_books", true},
		{"MITARBEITER", "edit_books", true},
		{"MITARBEITER", "delete_books", true},
		{"MITARBEITER", "inventory_scan", true},
		{"MITARBEITER", "manage_inventory", true},
		{"MITARBEITER", "view_orders", true},
		{"MITARBEITER", "create_orders", true},
		{"MITARBEITER", "view_graduates", true},
		{"MITARBEITER", "view_stats", true},
		{"MITARBEITER", "audit_logs", false},
		{"MITARBEITER", "manage_users", false},

		// Lehrer defaults
		{"LEHRER", "view_students", true},
		{"LEHRER", "edit_students", false},
		{"LEHRER", "create_students", false},
		{"LEHRER", "delete_students", false},
		{"LEHRER", "import_students", false},
		{"LEHRER", "upload_photos", true},
		{"LEHRER", "view_books", true},
		{"LEHRER", "edit_books", false},
		{"LEHRER", "delete_books", false},
		{"LEHRER", "inventory_scan", false},
		{"LEHRER", "view_orders", false},
		{"LEHRER", "create_orders", false},
		{"LEHRER", "view_graduates", false},
		{"LEHRER", "view_stats", false},
		{"LEHRER", "audit_logs", false},
		{"LEHRER", "manage_users", false},

		// Helfer defaults
		{"HELFER", "view_students", false},
		{"HELFER", "edit_students", false},
		{"HELFER", "create_students", false},
		{"HELFER", "delete_students", false},
		{"HELFER", "import_students", false},
		{"HELFER", "upload_photos", false},
		{"HELFER", "view_books", false},
		{"HELFER", "edit_books", false},
		{"HELFER", "delete_books", false},
		{"HELFER", "inventory_scan", false},
		{"HELFER", "view_orders", false},
		{"HELFER", "create_orders", false},
		{"HELFER", "view_graduates", false},
		{"HELFER", "view_stats", false},
		{"HELFER", "audit_logs", false},
		{"HELFER", "manage_users", false},
	}

	for _, d := range defaults {
		_, err = db.Pool.Exec(ctx, seedRolePermissionSQL, d.Role, d.Permission, d.Allowed)
		if err != nil {
			return fmt.Errorf("failed to seed permission default (%s, %s): %w", d.Role, d.Permission, err)
		}
	}
	return nil
}

// InitLieferanten initializes the lieferanten table and seeds it with default values.
func (db *Database) InitLieferanten(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, createLieferantenTableSQL)
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
			_, err = db.Pool.Exec(ctx, seedLieferantenSQL, d.Name, d.Email, d.Kundennummer)
			if err != nil {
				return fmt.Errorf("failed to seed supplier default (%s): %w", d.Name, err)
			}
		}
	}
	return nil
}

// InitAdmin checks if the users table is empty and bootstraps the first admin
// using INITIAL_ADMIN_EMAIL and INITIAL_ADMIN_PASSWORD environment variables.
func (db *Database) InitAdmin(ctx context.Context) error {
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM benutzer").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query benutzer count: %w", err)
	}

	if count > 0 {
		return nil // Users exist, no need to bootstrap
	}

	email := os.Getenv("INITIAL_ADMIN_EMAIL")

	if email == "" {
		log.Println("Warnung: Keine Benutzer in der Datenbank und INITIAL_ADMIN_EMAIL nicht gesetzt. System startet ohne Admin-Zugang.")
		return nil
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for initial admin: %w", err)
	}
	defer SafeRollback(ctx, tx)

	var adminID string
	err = tx.QueryRow(ctx, insertInitialAdminSQL, email).Scan(&adminID)
	if err != nil {
		return fmt.Errorf("failed to insert initial admin: %w", err)
	}

	_, err = tx.Exec(ctx, insertInitialAdminRoleSQL, adminID)
	if err != nil {
		return fmt.Errorf("failed to insert initial admin role: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit initial admin transaction: %w", err)
	}

	log.Printf("Erster Admin-Benutzer (%s) wurde erfolgreich initialisiert.", email)
	return nil
}
