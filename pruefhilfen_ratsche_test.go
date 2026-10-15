package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// pruefhilfen sind die Pakete, die nur Tests einbinden: Sie legen Zeilen in der Testdatenbank
// an und leeren Tabellen, bauen Attrappen und lesen erzeugte Blätter.
var pruefhilfen = []string{
	"bibliothek/internal/testdaten",
	"bibliothek/internal/tuertest",
	"bibliothek/internal/routertest",
	"bibliothek/internal/pgtest",
	"bibliothek/internal/pdftest",
	"bibliothek/internal/smtptest",
	"bibliothek/internal/xlsxtest",
}

// Kein Programmcode bindet eine Prüfhilfe ein. Die Prüfhilfen sind gewöhnliche Pakete, damit
// Tests mehrerer Pakete sie teilen können (docs/ARCHITEKTUR.md 5.2.2); was sie tun, gehört
// nicht in das ausgelieferte Programm.
//
// Blindheit: Nur die Einbindung im Kopf einer Datei. Eine Prüfhilfe, die dieser Liste fehlt,
// sieht die Prüfung nicht; ein neues Paket der Art wird hier eingetragen.
func TestPruefhilfen_NurTestsBindenSieEin(t *testing.T) {
	fset := token.NewFileSet()
	gelesen := 0
	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Versteckte Ordner tragen keinen Quelltext des Programms, wohl aber Arbeitskopien.
			if name := d.Name(); name == "node_modules" || name == "frontend" || name == "testdata" ||
				(strings.HasPrefix(name, ".") && pfad != ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		// Die Prüfhilfen bauen aufeinander auf.
		for _, hilfe := range pruefhilfen {
			if filepath.ToSlash(filepath.Dir(pfad)) == strings.TrimPrefix(hilfe, "bibliothek/") {
				return nil
			}
		}
		datei, err := parser.ParseFile(fset, pfad, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		gelesen++
		for _, einbindung := range datei.Imports {
			ziel, err := strconv.Unquote(einbindung.Path.Value)
			if err != nil {
				return err
			}
			for _, hilfe := range pruefhilfen {
				if ziel == hilfe {
					t.Errorf("%s bindet die Prüfhilfe %s ein. Sie gehört nur in Tests; was das Programm "+
						"braucht, steht in einem Paket des Programms.", pfad, hilfe)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}
	// Gegenprobe am Detektor: Liest er den Programmcode überhaupt?
	if gelesen < 400 {
		t.Fatalf("nur %d Dateien des Programms gelesen: Der Detektor greift ins Leere", gelesen)
	}
}
