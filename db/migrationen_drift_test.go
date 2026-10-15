package db

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

// migrationsEintrag trifft die Versionen, die schema.sql als angewendet einträgt, etwa
// ('035_lusd_id_partial_unique.sql').
var migrationsEintrag = regexp.MustCompile(`'(\d{3}_[a-z0-9_]+\.sql)'`)

// Die Migrationen, die schema.sql als angewendet einträgt, sind genau die Dateien in
// migrations/. Fehlt ein Eintrag, läuft die Migration bei einer neuen Installation gegen das
// fertige schema.sql und kann den Start abbrechen; ein Eintrag ohne Datei ist ein Tippfehler.
func TestMigrationslisteImSchemaGleichDenDateien(t *testing.T) {
	seed := versionenAusDemSchema(t)
	files := migrationsdateinamen(t)

	if missing := nurIn(files, seed); len(missing) > 0 {
		t.Errorf("Migrationsdateien fehlen in der schema.sql-Seed-Liste: %v", missing)
	}
	if extra := nurIn(seed, files); len(extra) > 0 {
		t.Errorf("Seed-Einträge in schema.sql ohne zugehörige Migrationsdatei: %v", extra)
	}
}

// versionenAusDemSchema liest die Versionen, die schema.sql als angewendet einträgt.
func versionenAusDemSchema(t *testing.T) map[string]bool {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "schema.sql"))
	if err != nil {
		t.Fatalf("schema.sql nicht lesbar: %v", err)
	}
	versions := make(map[string]bool)
	for _, m := range migrationsEintrag.FindAllStringSubmatch(string(content), -1) {
		versions[m[1]] = true
	}
	if len(versions) == 0 {
		t.Fatal("keine Seed-Einträge in schema.sql gefunden — Muster veraltet?")
	}
	return versions
}

// migrationsdateinamen liefert die Namen aller SQL-Dateien in migrations/.
func migrationsdateinamen(t *testing.T) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join("..", "migrations"))
	if err != nil {
		t.Fatalf("migrations/ nicht lesbar: %v", err)
	}
	files := make(map[string]bool)
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files[e.Name()] = true
		}
	}
	return files
}

// nurIn liefert die Schlüssel, die in a, aber nicht in b vorkommen (sortiert).
func nurIn(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
