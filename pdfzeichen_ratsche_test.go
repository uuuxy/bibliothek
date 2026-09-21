package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Ratsche: Jeder cp1252-Übersetzer eines PDF-Dokuments läuft durch pdfzeichen.Uebersetzer.
//
// Anlass (21.09.2026, OFFEN.md 5.5): Die Ersetzung für Buchstaben außerhalb von cp1252
// (ş, ł, ğ …) gab es seit dem Schüler-Etikett — aber nur dort und im Bescheid. Die
// Buchetiketten, die Mahnbriefe, die Bestellungen und alle anderen Renderer holten sich
// ihren Übersetzer direkt von gofpdf und druckten weiter Punkte. Zwei Wege zum selben
// Papier, und der zweite kannte die Regel des ersten nicht.
//
// Regel: Eine Zeile, die UnicodeTranslatorFromDescriptor( aufruft, trägt auf derselben
// Zeile pdfzeichen.Uebersetzer(. Wer einen neuen Renderer baut, bekommt die Ersetzung
// damit von allein — oder diesen Test rot.
func TestPdfUebersetzer_NurUeberPdfzeichen(t *testing.T) {
	var verstoesse []string
	gefunden := 0
	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "frontend" {
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
		for nr, zeile := range strings.Split(string(inhalt), "\n") {
			if strings.HasPrefix(strings.TrimSpace(zeile), "//") {
				continue
			}
			if !strings.Contains(zeile, "UnicodeTranslatorFromDescriptor(") {
				continue
			}
			gefunden++
			if !strings.Contains(zeile, "pdfzeichen.Uebersetzer(") {
				verstoesse = append(verstoesse, pfad+":"+strconv.Itoa(nr+1))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Nicht-leer-Garantie: Findet der Scanner keine Aufrufe mehr, prüft er nichts.
	if gefunden < 10 {
		t.Fatalf("nur %d Übersetzer-Aufrufe gefunden — der Scanner greift nicht mehr (erwartet ≥ 17)", gefunden)
	}
	if len(verstoesse) > 0 {
		t.Fatalf("cp1252-Übersetzer ohne pdfzeichen.Uebersetzer in %v — dort werden ş, ł, ğ zu Punkten. "+
			"Form: pdfzeichen.Uebersetzer(p.UnicodeTranslatorFromDescriptor(\"\"))", verstoesse)
	}
}
