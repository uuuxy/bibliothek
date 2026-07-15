package db

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

// seedEntryPattern erfasst die in schema.sql als angewendet markierten Versionen,
// z. B. ('035_lusd_id_partial_unique.sql').
var seedEntryPattern = regexp.MustCompile(`'(\d{3}_[a-z0-9_]+\.sql)'`)

// TestMigrationSeedListMatchesFiles stellt sicher, dass die in schema.sql als
// „bereits angewendet" markierten Migrationen EXAKT den Dateien in migrations/
// entsprechen. Fehlt ein Seed-Eintrag, läuft die Migration bei einer Neuinstallation
// gegen das fertige schema.sql und kann den Serverstart abbrechen; ein Seed-Eintrag
// ohne Datei ist ein Tippfehler. Dieser Test fängt beide Fälle in CI ab.
func TestMigrationSeedListMatchesFiles(t *testing.T) {
	seed := seedVersionsFromSchema(t)
	files := migrationFilenames(t)

	if missing := nurIn(files, seed); len(missing) > 0 {
		t.Errorf("Migrationsdateien fehlen in der schema.sql-Seed-Liste: %v", missing)
	}
	if extra := nurIn(seed, files); len(extra) > 0 {
		t.Errorf("Seed-Einträge in schema.sql ohne zugehörige Migrationsdatei: %v", extra)
	}
}

// seedVersionsFromSchema liest die geseedeten Migrationsversionen aus schema.sql.
func seedVersionsFromSchema(t *testing.T) map[string]bool {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "schema.sql"))
	if err != nil {
		t.Fatalf("schema.sql nicht lesbar: %v", err)
	}
	versions := make(map[string]bool)
	for _, m := range seedEntryPattern.FindAllStringSubmatch(string(content), -1) {
		versions[m[1]] = true
	}
	if len(versions) == 0 {
		t.Fatal("keine Seed-Einträge in schema.sql gefunden — Muster veraltet?")
	}
	return versions
}

// migrationFilenames listet alle *.sql-Dateien in migrations/.
func migrationFilenames(t *testing.T) map[string]bool {
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
