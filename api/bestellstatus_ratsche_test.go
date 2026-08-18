package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Die Ratsche zu Befund F1: Kein Produktions-SQL darf den Bestellstatus je
// wieder aus dem Freitext zustand_notiz ableiten. Die Muster unten sind genau
// die vier LIKE-Filter, die bis Migration 071 in OPAC, Inventur und
// Wareneingang standen. Erlaubt bleibt das Go-Präfix-Parsen des
// LIEFERANTENNAMENS aus der Notiz (reine Anzeige, strings.HasPrefix) —
// verboten ist jede SQL-Entscheidung über den Text.
func TestKeinBestellstatusAusNotizText(t *testing.T) {
	verboten := []string{
		"LIKE 'Im Zulauf",
		"LIKE 'Bestellt",
		"zustand_notiz = 'bestellt'",
		`zustand_notiz != 'bestellt'`,
	}
	wurzel := ".."
	err := filepath.Walk(wurzel, func(pfad string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == ".git" || name == "migrations" || name == "bewertung" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		for _, muster := range verboten {
			if strings.Contains(string(inhalt), muster) {
				t.Errorf("%s entscheidet wieder über den Notiz-Text (%q) — Bestellstatus gehört in die Spalte bestellstatus (Migration 071, Befund F1)", pfad, muster)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Quelldurchlauf: %v", err)
	}
}
