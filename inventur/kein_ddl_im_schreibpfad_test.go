package inventur

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Kein Schema-Befehl im Programm: Eine Tabelle, eine Sequenz oder ein Index entsteht in einer
// Migration und steht in schema.sql. Legt das Programm die Struktur zur Laufzeit an, hält die
// Anweisung ihre Sperre bis zum Ende der Transaktion, und der Vergleich der Migrationen mit
// schema.sql sieht das Objekt nicht, weil es auf beiden Wegen fehlt.
//
// Gelesen wird der Go-Code des ganzen Baums ohne Tests, auch in Unterordnern. Blindheit: eine
// Anweisung, die aus Teilen zusammengesetzt oder aus einer .sql-Datei gelesen wird, und die
// Werkzeuge unter cmd/, die Struktur anlegen dürfen.
//
// Bei Rot: die Anweisung aus dem Go-Code nehmen und das Objekt in migrations/ und schema.sql
// anlegen. Braucht eine Datei die Anweisung zur Laufzeit, steht sie mit dem Grund in
// ddlAusnahmen.
func TestKeinDDLImSchreibpfad(t *testing.T) {
	funde, gelesen, err := ddlImBaum("..")
	if err != nil {
		t.Fatalf("Baum nicht lesbar: %v", err)
	}
	if gelesen < 300 {
		t.Fatalf("nur %d Go-Dateien gelesen — der Sammler sieht den Baum nicht", gelesen)
	}

	gedeckt := map[string]bool{}
	var offen []string
	for _, fund := range funde {
		if _, erlaubt := ddlAusnahmen[fund.datei]; erlaubt {
			gedeckt[fund.datei] = true
			continue
		}
		offen = append(offen, fund.datei+":"+strconv.Itoa(fund.zeile)+": "+fund.text)
	}
	if len(offen) > 0 {
		t.Errorf("DDL im Anwendungscode — gehört in eine Migration und in schema.sql:\n%s",
			strings.Join(offen, "\n"))
	}
	// Ein Eintrag ohne Anweisung deckte die nächste in derselben Datei, ohne dass jemand fragt.
	for datei := range ddlAusnahmen {
		if !gedeckt[datei] {
			t.Errorf("ddlAusnahmen nennt %s, dort steht keine solche Anweisung mehr — Eintrag austragen", datei)
		}
	}
}

// ddlAnweisung trifft Anweisungen, die Struktur ändern. ALTER und DROP stehen dabei, obwohl
// der erste Fund ein CREATE war: Die Fehlerart ist größer als ihr Anlass.
var ddlAnweisung = regexp.MustCompile(`(?i)\b(CREATE|ALTER|DROP)\s+(SEQUENCE|TABLE|INDEX|VIEW|TYPE|SCHEMA|FUNCTION|TRIGGER)\b`)

// ddlAusnahmen nennt die Dateien, die Struktur zur Laufzeit anlegen dürfen, mit dem Grund.
var ddlAusnahmen = map[string]string{
	"db/migrationen.go":         "der Runner der Migrationen: Die Tabelle schema_migrations muss es vor der ersten Migration geben",
	"internal/pgtest/pgtest.go": "die Harness der Datenbank-Tests leert das Schema der Testdatenbank, bevor sie schema.sql lädt",
}

// ddlOhneQuelltext nennt Ordner, die keinen Go-Code des Programms tragen.
var ddlOhneQuelltext = []string{"node_modules", "frontend", "testdata", "uploads"}

type ddlFund struct {
	datei string
	zeile int
	text  string
}

// ddlImBaum liest die Go-Dateien unter der Wurzel ohne Tests und nennt jede Zeile mit einer
// Anweisung, die Struktur ändert, dazu die Zahl der gelesenen Dateien. Kommentarzeilen zählen
// nicht: Sie erklären die Regel und sind der häufigste Fehlalarm dieser Bauart.
func ddlImBaum(wurzel string) (funde []ddlFund, gelesen int, err error) {
	err = filepath.WalkDir(wurzel, func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(wurzel, pfad)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		name := eintrag.Name()
		if eintrag.IsDir() {
			// Versteckte Ordner tragen keinen Quelltext des Programms, wohl aber Arbeitskopien.
			if pfad != wurzel && (strings.HasPrefix(name, ".") || slices.Contains(ddlOhneQuelltext, name) || rel == "cmd") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad) //nolint:gosec // Dateien des eigenen Baums
		if err != nil {
			return err
		}
		gelesen++
		for nr, zeile := range strings.Split(string(inhalt), "\n") {
			if strings.HasPrefix(strings.TrimSpace(zeile), "//") {
				continue
			}
			if ddlAnweisung.MatchString(zeile) {
				funde = append(funde, ddlFund{rel, nr + 1, strings.TrimSpace(zeile)})
			}
		}
		return nil
	})
	return funde, gelesen, err
}

// Der Sammler sieht eine Datei in jedem Ordner des Baums, auch in einem Unterordner von api/,
// und lässt aus, was kein Quelltext des Programms ist.
func TestDDLImBaum_Selbstprobe(t *testing.T) {
	wurzel := t.TempDir()
	anweisung := "package x\n\nconst a = `CREATE SEQUENCE IF NOT EXISTS probe_seq`\n"
	for pfad, inhalt := range map[string]string{
		"api/tuer.go":                 anweisung,
		"api/thema/regel.go":          "package thema\n\nconst a = `alter table probe add column b int`\n",
		"mailservice/versand.go":      "package mailservice\n\nvar a = \"DROP INDEX probe_idx\"\n",
		"repository/sauber.go":        "package repository\n\n// CREATE TABLE steht hier nur im Kommentar.\nconst a = `SELECT 1`\n",
		"repository/zeile_test.go":    anweisung,
		"cmd/werkzeug/main.go":        anweisung,
		"internal/cmd/regel.go":       anweisung,
		"frontend/node_modules/x.go":  anweisung,
		"api/thema/testdata/datei.go": anweisung,
		".arbeitskopie/api/tuer.go":   anweisung,
		"uploads/abgelegt.go":         anweisung,
		"api/uploads/abgelegt.go":     anweisung,
	} {
		ziel := filepath.Join(wurzel, filepath.FromSlash(pfad))
		if err := os.MkdirAll(filepath.Dir(ziel), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ziel, []byte(inhalt), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	funde, gelesen, err := ddlImBaum(wurzel)
	if err != nil {
		t.Fatal(err)
	}
	var stellen []string
	for _, fund := range funde {
		stellen = append(stellen, fund.datei+":"+strconv.Itoa(fund.zeile))
	}
	slices.Sort(stellen)
	erwartet := []string{"api/thema/regel.go:3", "api/tuer.go:3", "internal/cmd/regel.go:3", "mailservice/versand.go:3"}
	if !slices.Equal(stellen, erwartet) {
		t.Errorf("gefunden %v, erwartet %v", stellen, erwartet)
	}
	if gelesen != 5 {
		t.Errorf("%d Dateien gelesen, erwartet 5 (die vier mit Fund und repository/sauber.go)", gelesen)
	}
}
