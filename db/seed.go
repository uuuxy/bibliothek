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
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ('admin', 'System', 'Administrator', $1, 'admin', true)
		RETURNING id
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
	if err := db.createTrgmIndexes(ctx); err != nil {
		return err
	}

	// 3. Migrate role_permissions table role column to VARCHAR(50) if it's enum
	if err := db.migrateRolePermissionsColumn(ctx); err != nil {
		return err
	}

	// 4. Create role_permissions table
	_, err = db.Pool.Exec(ctx, createRolePermissionsTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create role_permissions table: %w", err)
	}

	// 5. Seed default role permissions with uppercase role names.
	return db.seedRolePermissions(ctx)
}

// seedRolePermissions seeds the role_permissions table with default values.
func (db *Database) seedRolePermissions(ctx context.Context) error {
	// Die Rolle eines Benutzers steht in benutzer.rolle (ENUM, kleingeschrieben); die
	// Middleware verbindet beide Vokabulare per UPPER() (permission_middleware.go).
	// role_permissions bildet also nur ab, was eine ROLLE darf — nicht, wer sie hat.
	defaults := []struct {
		Role       string
		Permission string
		Allowed    bool
	}{
		// Admin defaults
		{"ADMIN", "perform_actions", true},
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
		{"MITARBEITER", "perform_actions", true},
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
		{"LEHRER", "perform_actions", true},
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

		// Helfer defaults — Kiosk-Rolle: darf NUR am Terminal ausleihen/zurücknehmen
		// (perform_actions: /api/action, /scan, /search, /events). Bewusst KEIN
		// view_students: das gäbe Zugriff auf Schülerlisten, Profile, Mahnwesen und den
		// Bulk-Mahndruck (= Mahnstufen-Eskalation) — alles jenseits des Kiosk-Zwecks.
		{"HELFER", "perform_actions", true},
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
		_, err := db.Pool.Exec(ctx, seedRolePermissionSQL, d.Role, d.Permission, d.Allowed)
		if err != nil {
			return fmt.Errorf("failed to seed permission default (%s, %s): %w", d.Role, d.Permission, err)
		}
	}
	return nil
}

// createTrgmIndexes legt die pg_trgm-GIN-Indizes für die Fuzzy-Suche an (idempotent).
func (db *Database) createTrgmIndexes(ctx context.Context) error {
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
	return nil
}

// migrateRolePermissionsColumn hebt eine evtl. noch als ENUM angelegte role-Spalte auf
// VARCHAR(50) an (idempotent, nur bei USER-DEFINED-Typ).
func (db *Database) migrateRolePermissionsColumn(ctx context.Context) error {
	var dataType string
	err := db.Pool.QueryRow(ctx, `
		SELECT data_type
		FROM information_schema.columns
		WHERE table_name = 'role_permissions' AND column_name = 'role'
	`).Scan(&dataType)
	if err == nil && dataType == "USER-DEFINED" {
		if _, err := db.Pool.Exec(ctx, "ALTER TABLE role_permissions ALTER COLUMN role TYPE VARCHAR(50);"); err != nil {
			return fmt.Errorf("failed to alter role_permissions.role column type: %w", err)
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

	// Ein einzelnes INSERT ist für sich atomar — die frühere Transaktion klammerte den
	// zusätzlichen Insert in die inzwischen entfernte Tabelle benutzer_rollen.
	var adminID string
	if err := db.Pool.QueryRow(ctx, insertInitialAdminSQL, email).Scan(&adminID); err != nil {
		return fmt.Errorf("failed to insert initial admin: %w", err)
	}

	log.Printf("Erster Admin-Benutzer (%s) wurde erfolgreich initialisiert.", email)
	return nil
}
