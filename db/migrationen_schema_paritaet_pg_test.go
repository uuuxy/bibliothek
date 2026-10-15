package db

// Zwei Wege führen zu einer Datenbank: Eine frische Installation bekommt das heutige schema.sql
// als Ausgangsstand (spieleSchemaEin), eine gewachsene wie die des Betriebs bekommt nacheinander
// die Migrationen. Beide müssen dieselbe Struktur ergeben; sonst prüft jeder Test an der
// Datenbank ein Schema, das es im Betrieb so nicht gibt, etwa eine Spalte, die eine Migration
// anlegt und schema.sql nicht kennt.
//
// Der Test baut beide Wege nach: den eingefrorenen Ausgangsstand in db/testdata/ mit dem echten
// Lauf der Migrationen gegen das heutige schema.sql. Er vergleicht Spalten, Bedingungen, Indexe,
// Trigger, Funktionen, Enums und Sequenzen; jede Meldung heißt, dass einer der beiden Wege etwas
// hat, das der andere nicht hat.

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"

	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// paritaetsAspekte hat je Teil der Struktur eine Abfrage mit genau einer Spalte Text.
// Verglichen werden Mengen von Zeilen: Die Reihenfolge der Spalten einer Tabelle darf abweichen
// (ALTER ADD COLUMN hängt hinten an, schema.sql ordnet nach Sinn), die Namen der Bedingungen
// nicht. Beide Wege schreiben dieselben Namen, und ein anderer Name ist eine Abweichung.
var paritaetsAspekte = []struct{ name, query string }{
	{"Spalte", `SELECT table_name || '.' || column_name || ' typ=' || data_type ||
		coalesce('(' || character_maximum_length || ')', '') || ' null=' || is_nullable ||
		' default=' || coalesce(column_default, '-')
		FROM information_schema.columns WHERE table_schema = 'public'`},
	// conname steht im Vergleich: Ein Fremdschlüssel kann auf beiden Wegen gleich wirken und
	// verschieden heißen.
	{"Constraint", `SELECT conrelid::regclass::text || ' ' || conname || ' ' || contype::text || ' ' || pg_get_constraintdef(oid)
		FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND contype <> 't'`},
	{"Index", `SELECT tablename || ' ' || indexdef FROM pg_indexes WHERE schemaname = 'public'`},
	{"Trigger", `SELECT DISTINCT event_object_table || ' ' || trigger_name || ' ' || action_timing
		FROM information_schema.triggers WHERE trigger_schema = 'public'`},
	// Mit dem Rumpf (md5 von prosrc), nicht nur dem Namen: Ersetzt eine Migration eine
	// Funktion eines Triggers anders als schema.sql, liefen die Tests gegen den frischen Rumpf,
	// und der Server bekäme den gewachsenen. Die Rümpfe sind auf beiden Wegen Byte für Byte
	// gleich; ein Kommentar im Rumpf zählt also mit.
	{"Funktion", `SELECT proname || ' md5=' || md5(prosrc) FROM pg_proc WHERE pronamespace = 'public'::regnamespace`},
	{"Enum-Wert", `SELECT t.typname || '=' || e.enumlabel FROM pg_type t
		JOIN pg_enum e ON e.enumtypid = t.oid WHERE t.typnamespace = 'public'::regnamespace`},
	{"Sequenz", `SELECT sequencename FROM pg_sequences WHERE schemaname = 'public'`},
}

// sqlRechtetabelleAusDemStart ist das SQL, mit dem früher der Start des Programms role_permissions
// anlegte, wörtlich eingefroren: Auf dem gewachsenen Weg bekam die Tabelle genau so ihre Form,
// bevor Migration 055 sie voraussetzte. Migration 106 legt sie heute gleich an; weicht eine der
// beiden Fassungen ab, meldet es der Vergleich unten.
const sqlRechtetabelleAusDemStart = `
		CREATE TABLE IF NOT EXISTS role_permissions (
			role VARCHAR(50) NOT NULL,
			permission VARCHAR(100) NOT NULL,
			allowed BOOLEAN NOT NULL DEFAULT false,
			PRIMARY KEY (role, permission)
		)
	`

func TestSchemaParitaet_GewachsenGleichFrisch(t *testing.T) {
	dsn := os.Getenv(pgtest.Variable)
	if dsn == "" {
		t.Skipf("%s nicht gesetzt — DB-Integrationstest übersprungen", pgtest.Variable)
	}
	ctx := context.Background()

	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Admin-Verbindung: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close(ctx) }) //nolint:errcheck

	gewachsen := paritaetsDB(t, ctx, admin, dsn, "bibliothek_test_paritaet_gewachsen")
	frisch := paritaetsDB(t, ctx, admin, dsn, "bibliothek_test_paritaet_frisch")

	// Gewachsener Weg: der eingefrorene Ausgangsstand, dann der echte Lauf über migrations/
	// (spieleSchemaEin greift nicht, der Ausgangsstand bringt schema_migrations mit).
	//
	// Auf einer gewachsenen Anlage gab es role_permissions aus dem Start des Programms, bevor
	// Migration 055 die Tabelle voraussetzte. Das Nachspielen legt sie deshalb vor dem Lauf
	// an (sqlRechtetabelleAusDemStart); Migration 106 läuft nach 055 und kann nicht vor sie
	// treten.
	fuehreDateiAus(t, ctx, gewachsen, "testdata/schema_baseline_e5740b95.sql")
	if _, err := gewachsen.Exec(ctx, sqlRechtetabelleAusDemStart); err != nil {
		t.Fatalf("role_permissions (Startup-Tabelle) anlegen: %v", err)
	}
	booteAnwendung(t, ctx, gewachsen)

	// Frischer Weg: das heutige schema.sql wie bei einer neuen Installation, dann derselbe Start
	// (die Migrationen stehen in schema.sql als angewendet, der Lauf tut also nichts).
	fuehreDateiAus(t, ctx, frisch, "../schema.sql")
	booteAnwendung(t, ctx, frisch)

	for _, aspekt := range paritaetsAspekte {
		zeilenGewachsen := ladeZeilen(t, ctx, gewachsen, aspekt.query)
		zeilenFrisch := ladeZeilen(t, ctx, frisch, aspekt.query)
		meldeFehlende(t, aspekt.name, "nur im GEWACHSENEN Pfad (Migration ohne schema.sql-Nachzug?)", zeilenGewachsen, zeilenFrisch)
		meldeFehlende(t, aspekt.name, "nur im FRISCHEN Pfad (schema.sql-Änderung ohne Migration?)", zeilenFrisch, zeilenGewachsen)
	}
}

// booteAnwendung geht den Weg des Starts (main.go: FuehreMigrationenAus, dann
// SchreibeRechteVorgabe). Beide Seiten starten gleich; so misst der Vergleich nur den Unterschied
// zwischen der Kette der Migrationen und schema.sql, nicht den Code des Starts.
func booteAnwendung(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	d := &Verbindung{Pool: pool}
	if err := d.FuehreMigrationenAus(ctx, "../migrations"); err != nil {
		t.Fatalf("Migrationslauf gescheitert — dieser Pfad ist nicht mehr abspielbar: %v", err)
	}
	if err := d.SchreibeRechteVorgabe(ctx); err != nil {
		t.Fatalf("SchreibeRechteVorgabe: %v", err)
	}
}

// paritaetsDB legt eine leere Datenbank zum Wegwerfen auf demselben Server an (vorher DROP,
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
		// Scheitert das Aufräumen, räumt der nächste Lauf mit DROP IF EXISTS nach.
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

// meldeFehlende meldet jede Zeile aus menge, die in referenz fehlt.
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
