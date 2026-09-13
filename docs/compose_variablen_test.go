package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Gate: Jede Umgebungsvariable, die Compose dem Backend gibt, wird auch gelesen.
//
// Anlass (Register, Bestands-Durchgang 10.09.2026): `DB_HOST=postgres-db` und
// `SMTP_SENDER=${SMTP_FROM}` standen in beiden Compose-Dateien, und kein Go-Code fragte
// je danach. Solche Zeilen sind nicht bloß Ballast — sie sehen aus wie eine Einstellung.
// Wer den Datenbank-Host ändern will, ändert DB_HOST, startet neu und wundert sich; die
// Verbindung kommt aus DATABASE_URL.
//
// Die Gegenrichtung (Go liest etwas, das Compose NICHT durchreicht) ist der teurere Fall
// — er hat den nächtlichen Backup-Job zweimal still ausfallen lassen. Sie braucht eine
// Ausnahmeliste für die Werkzeug-Variablen (TEST_DATABASE_URL, PG_DSN, …) und liegt als
// Betriebsposten in #599; dieses Gate deckt die Richtung, die ohne Liste auskommt.
var composeVariable = regexp.MustCompile(`^\s+-\s+([A-Z0-9_]+)=`)

// backendBlock liefert den Abschnitt des backend-Dienstes einer Compose-Datei.
func backendBlock(t *testing.T, inhalt string) string {
	t.Helper()
	start := strings.Index(inhalt, "\n  backend:")
	if start < 0 {
		t.Fatal("kein backend-Dienst in der Compose-Datei — dieses Gate wäre still grün")
	}
	rest := inhalt[start+1:]
	// Der nächste Dienst auf derselben Ebene beendet den Block.
	if m := regexp.MustCompile(`\n  [a-z0-9_-]+:\n`).FindStringIndex(rest); m != nil {
		return rest[:m[0]]
	}
	return rest
}

func TestComposeVariablenWerdenGelesen(t *testing.T) {
	// Der Go-Code des Servers als eine Zeichenkette: gesucht wird der Name in
	// Anführungszeichen, so wie os.Getenv ihn nimmt.
	var quellen strings.Builder
	err := filepath.Walk("..", func(pfad string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if name := info.Name(); name == "node_modules" || name == ".git" || name == "frontend" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad) //nolint:gosec // Repo-eigene Quellen
		if err != nil {
			return err
		}
		quellen.Write(inhalt)
		return nil
	})
	if err != nil {
		t.Fatalf("Go-Quellen sammeln: %v", err)
	}
	code := quellen.String()
	if len(code) < 100_000 {
		t.Fatalf("nur %d Bytes Go-Code gesammelt — der Sammler greift ins Leere", len(code))
	}

	for _, datei := range []string{"../docker-compose.yml", "../docker-compose.local.yml"} {
		inhalt := lies(t, datei)
		gesehen := map[string]bool{}
		var namen []string
		for _, zeile := range strings.Split(backendBlock(t, inhalt), "\n") {
			m := composeVariable.FindStringSubmatch(zeile)
			if m == nil || gesehen[m[1]] {
				continue
			}
			gesehen[m[1]] = true
			namen = append(namen, m[1])
		}
		if len(namen) < 10 {
			t.Fatalf("%s: nur %d Variablen gefunden — der Sammler greift ins Leere", datei, len(namen))
		}
		sort.Strings(namen)
		for _, name := range namen {
			if !strings.Contains(code, `"`+name+`"`) {
				t.Errorf("%s reicht %s in den Container, aber kein Go-Code liest es — "+
					"entweder die Zeile streichen oder den Leser bauen (eine Zeile, die nach "+
					"Einstellung aussieht und keine ist, kostet jemanden eine Stunde)", datei, name)
			}
		}
	}
}
