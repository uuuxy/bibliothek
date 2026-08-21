package db

// Schema-Parität: gewachsener Pfad ≡ frischer Pfad.
//
// Es gibt zwei Wege zu einer Datenbank: Eine FRISCHE Installation bekommt das heutige
// schema.sql als Baseline (ensureBaselineSchema), eine GEWACHSENE (wie Prod) bekommt
// nacheinander die Migrationen. Dass beide Wege dieselbe Struktur ergeben, war bisher
// reine Konvention — „Migration schreiben UND schema.sql nachziehen" stand in keinem
// Gate. Genau diese Klasse war der anonymized_at-Drift: Migration 022 legte die Spalte
// an, schema.sql kannte sie nicht, und jeder PG-Test testete ein Schema, das es auf
// Prod so nicht gibt.
//
// Der Test baut beide Wege real nach: die eingefrorene Baseline aus Commit e5740b95
// (db/testdata/, 05.06.2026, Seed-Liste bis 006) plus den ECHTEN Migrations-Runner
// gegen das heutige schema.sql — und vergleicht Spalten, Constraints, Indexe, Trigger,
// Funktionen, Enums und Sequenzen. Jede Meldung heißt: einer der beiden Wege hat etwas,
// das der andere nicht hat.

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// paritaetsAspekte: je eine Abfrage pro Strukturaspekt, jede liefert genau eine
// Textspalte. Verglichen werden MENGEN von Zeilen — die Spaltenreihenfolge in der
// Tabelle darf abweichen (ALTER ADD COLUMN hängt hinten an, schema.sql sortiert
// logisch), Namen von Constraints dagegen nicht: Beide Wege werden bewusst mit
// denselben Namen geschrieben, und ein abweichender Name IST eine Abweichung.
var paritaetsAspekte = []struct{ name, query string }{
	{"Spalte", `SELECT table_name || '.' || column_name || ' typ=' || data_type ||
		coalesce('(' || character_maximum_length || ')', '') || ' null=' || is_nullable ||
		' default=' || coalesce(column_default, '-')
		FROM information_schema.columns WHERE table_schema = 'public'`},
	{"Constraint", `SELECT conrelid::regclass::text || ' ' || contype::text || ' ' || pg_get_constraintdef(oid)
		FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND contype <> 't'`},
	{"Index", `SELECT tablename || ' ' || indexdef FROM pg_indexes WHERE schemaname = 'public'`},
	{"Trigger", `SELECT DISTINCT event_object_table || ' ' || trigger_name || ' ' || action_timing
		FROM information_schema.triggers WHERE trigger_schema = 'public'`},
	{"Funktion", `SELECT proname FROM pg_proc WHERE pronamespace = 'public'::regnamespace`},
	{"Enum-Wert", `SELECT t.typname || '=' || e.enumlabel FROM pg_type t
		JOIN pg_enum e ON e.enumtypid = t.oid WHERE t.typnamespace = 'public'::regnamespace`},
	{"Sequenz", `SELECT sequencename FROM pg_sequences WHERE schemaname = 'public'`},
}

func TestSchemaParitaet_GewachsenGleichFrisch(t *testing.T) {
	dsn := os.Getenv(testDBEnvVar)
	if dsn == "" {
		t.Skipf("%s nicht gesetzt — DB-Integrationstest übersprungen", testDBEnvVar)
	}
	ctx := context.Background()

	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Admin-Verbindung: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close(ctx) }) //nolint:errcheck

	gewachsen := paritaetsDB(t, ctx, admin, dsn, "bibliothek_test_paritaet_gewachsen")
	frisch := paritaetsDB(t, ctx, admin, dsn, "bibliothek_test_paritaet_frisch")

	// Gewachsener Pfad: eingefrorene Baseline, dann der ECHTE Runner über migrations/.
	// (ensureBaselineSchema greift nicht: die Baseline bringt schema_migrations mit.)
	//
	// role_permissions kommt aus dem Go-Startup (db/seed.go), nicht aus einer Migration —
	// auf einer gewachsenen Anlage existierte die Tabelle aus früheren Boots, BEVOR
	// Migration 055 sie voraussetzte. Der Replay komprimiert die Zeit und muss diese
	// Startup-Tabelle deshalb vor dem Migrationslauf anlegen (dasselbe SQL wie der Boot).
	fuehreDateiAus(t, ctx, gewachsen, "testdata/schema_baseline_e5740b95.sql")
	if _, err := gewachsen.Exec(ctx, createRolePermissionsTableSQL); err != nil {
		t.Fatalf("role_permissions (Startup-Tabelle) anlegen: %v", err)
	}
	booteAnwendung(t, ctx, gewachsen)

	// Frischer Pfad: das heutige schema.sql wie bei einer Neuinstallation, dann derselbe
	// Boot (Migrationen sind per Seed-Liste als angewendet markiert — muss ein No-op sein).
	fuehreDateiAus(t, ctx, frisch, "../schema.sql")
	booteAnwendung(t, ctx, frisch)

	for _, aspekt := range paritaetsAspekte {
		zeilenGewachsen := ladeZeilen(t, ctx, gewachsen, aspekt.query)
		zeilenFrisch := ladeZeilen(t, ctx, frisch, aspekt.query)
		meldeFehlende(t, aspekt.name, "nur im GEWACHSENEN Pfad (Migration ohne schema.sql-Nachzug?)", zeilenGewachsen, zeilenFrisch)
		meldeFehlende(t, aspekt.name, "nur im FRISCHEN Pfad (schema.sql-Änderung ohne Migration?)", zeilenFrisch, zeilenGewachsen)
	}
}

// booteAnwendung führt den Server-Startpfad aus (main.go: RunMigrations, dann
// InitPermissions). Beide Seiten booten IDENTISCH — so verrechnet der Vergleich nur
// den Unterschied zwischen Migrationskette und schema.sql, nicht den Boot-Code.
func booteAnwendung(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	d := &Database{Pool: pool}
	if err := d.RunMigrations(ctx, "../migrations"); err != nil {
		t.Fatalf("Migrationslauf gescheitert — dieser Pfad ist nicht mehr abspielbar: %v", err)
	}
	if err := d.InitPermissions(ctx); err != nil {
		t.Fatalf("InitPermissions: %v", err)
	}
}

// paritaetsDB legt eine leere Wegwerf-Datenbank auf demselben Server an (DROP zuerst,
// falls ein abgebrochener Lauf sie hinterließ) und liefert einen Pool darauf.
func paritaetsDB(t *testing.T, ctx context.Context, admin *pgx.Conn, dsn, name string) *pgxpool.Pool {
	t.Helper()
	if _, err := admin.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, name)); err != nil {
		t.Fatalf("DROP %s: %v", name, err)
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s`, name)); err != nil {
		t.Fatalf("CREATE %s: %v", name, err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("DSN unlesbar: %v", err)
	}
	cfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("Pool auf %s: %v", name, err)
	}
	t.Cleanup(func() {
		pool.Close()
		// Aufräumen ist best effort — der nächste Lauf räumt per DROP IF EXISTS nach.
		_, _ = admin.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, name)) //nolint:errcheck
	})
	return pool
}

func fuehreDateiAus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, pfad string) {
	t.Helper()
	inhalt, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("%s nicht lesbar: %v", pfad, err)
	}
	if _, err := pool.Exec(ctx, string(inhalt)); err != nil {
		t.Fatalf("%s anwenden: %v", pfad, err)
	}
}

func ladeZeilen(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string) map[string]bool {
	t.Helper()
	rows, err := pool.Query(ctx, query)
	if err != nil {
		t.Fatalf("Strukturabfrage: %v", err)
	}
	defer rows.Close()
	zeilen := map[string]bool{}
	for rows.Next() {
		var z string
		if err := rows.Scan(&z); err != nil {
			t.Fatalf("Scan: %v", err)
		}
		zeilen[z] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Strukturabfrage: %v", err)
	}
	return zeilen
}

// meldeFehlende meldet jede Zeile aus `menge`, die in `referenz` fehlt.
func meldeFehlende(t *testing.T, aspekt, richtung string, menge, referenz map[string]bool) {
	t.Helper()
	var fehlend []string
	for z := range menge {
		if !referenz[z] {
			fehlend = append(fehlend, z)
		}
	}
	sort.Strings(fehlend)
	for _, z := range fehlend {
		t.Errorf("%s %s: %s", aspekt, richtung, z)
	}
}
