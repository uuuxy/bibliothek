package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Kein SQL im Programm darf den Bestellstatus wieder aus dem Freitext zustand_notiz ableiten; er
// steht in der Spalte bestellstatus. Die Muster unten sind die vier Filter auf den Text, die es
// dafür gab. Erlaubt bleibt, in Go den Namen des Lieferanten vorn aus der Notiz zu lesen (reine
// Anzeige, strings.HasPrefix); verboten ist jede Entscheidung in SQL über den Text.
//
// Blind für jede andere Schreibweise der vier Muster: like klein geschrieben, ein Vergleich ohne
// Leerzeichen, ein Muster aus einer Variablen.
func TestKeinBestellstatusAusNotizText(t *testing.T) {
	verboten := []string{
		"LIKE 'Im Zulauf",
		"LIKE 'Bestellt",
		"zustand_notiz = 'bestellt'",
		`zustand_notiz != 'bestellt'`,
	}
	wurzel := "."
	mitNotiz := 0
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
		if strings.Contains(string(inhalt), "zustand_notiz") {
			mitNotiz++
		}
		for _, muster := range verboten {
			if strings.Contains(string(inhalt), muster) {
				t.Errorf("%s entscheidet wieder über den Notiz-Text (%q) — der Bestellstatus steht in der Spalte bestellstatus", pfad, muster)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Quelldurchlauf: %v", err)
	}
	// Die Abfragen an Exemplaren nennen die Notiz. Sieht der Lauf kaum eine Datei mit ihr, reicht
	// er nicht bis zu ihnen und fände dort auch keinen Filter. Eine allein genügt nicht: Die
	// Schnittstellenbeschreibung in docs/ nennt das Feld auch.
	if mitNotiz < 10 {
		t.Fatalf("Go-Dateien unter %q, die zustand_notiz nennen: %d. Der Lauf erreicht die Abfragen an "+
			"Exemplaren nicht und prüft nichts", wurzel, mitNotiz)
	}
}
