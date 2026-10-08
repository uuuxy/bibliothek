package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Ein Geldbetrag steht überall in derselben Schreibweise: zwei Nachkommastellen mit Komma, aus
// pkg/betrag. Neben zwei Helfern, die das Komma schrieben, standen sieben Stellen mit
// Dezimalpunkt: zwei Meldungen der Theke („Forderung über 12.50 € storniert") und fünf in
// Briefen und Rechnungen („12.50 EUR"). Dieselbe Bugklasse wie im Frontend (docs/sweeps.md,
// „Schreibweise neben ihrem Helfer").
//
// Blind für: einen Betrag über %v oder strconv, ein Format aus zwei Zeichenketten, und einen
// Betrag ohne Währung dahinter (Tabellenspalten, deren Kopf die Währung nennt).
func TestBetraege_KommenAusDemHelfer(t *testing.T) {
	vonHand := regexp.MustCompile(`%\.\d?[fF]\s?(€|EUR|Euro)`)
	gelesen := 0
	err := filepath.WalkDir(".", func(pfad string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "frontend", "docs":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad) // #nosec G304 -- Repo-Dateien
		if err != nil {
			return err
		}
		gelesen++
		for i, zeile := range strings.Split(string(inhalt), "\n") {
			if strings.HasPrefix(strings.TrimSpace(zeile), "//") {
				continue
			}
			if fund := vonHand.FindString(zeile); fund != "" {
				t.Errorf("%s:%d schreibt einen Betrag von Hand (%s) — betrag.Euro oder betrag.Text aus pkg/betrag nehmen: %s",
					filepath.ToSlash(pfad), i+1, fund, strings.TrimSpace(zeile))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if gelesen < 300 {
		t.Fatalf("nur %d Go-Dateien gelesen — der Detektor sieht den Baum nicht", gelesen)
	}
}
