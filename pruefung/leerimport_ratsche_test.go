package pruefung

import (
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// Ein Paket, das nur für seine Nebenwirkung eingebunden wird (import _), gehört in ein Programm
// oder in einen Test. So verlangt es Googles Regelwerk für Go (Decisions, „Import blank"):
// „Avoid blank imports in library packages, even if the library indirectly depends on them."
// Das Programm entscheidet dann, welche Bildformate es liest, und ein Test kann andere wählen.
//
// Ausgenommen ist embed, wie bei Google: Eine Datei mit //go:embed braucht den Import.
// Blindheit: Eine Nebenwirkung hinter einem Import mit Namen sieht die Prüfung nicht
// (chai2010/webp meldet in pkg/imageutil auch seinen Decoder an), Testdateien liest sie nicht.
func TestLeerimporte_NurInMainUndTests(t *testing.T) {
	quellen := leseGoQuellen(t, ".")
	fset := token.NewFileSet()
	var funde []string
	programme := 0
	for pfad, quelle := range quellen {
		if strings.HasSuffix(pfad, "_test.go") {
			continue
		}
		datei, err := parser.ParseFile(fset, pfad, quelle, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("%s lesen: %v", pfad, err)
		}
		if datei.Name.Name == "main" {
			programme++
			continue
		}
		for _, imp := range datei.Imports {
			ziel := strings.Trim(imp.Path.Value, `"`)
			if imp.Name != nil && imp.Name.Name == "_" && ziel != "embed" {
				funde = append(funde, fset.Position(imp.Pos()).String()+": "+ziel)
			}
		}
	}

	// Gegenprobe am Detektor: Ohne die Programme unter cmd/ liest er den falschen Baum.
	if programme < 5 {
		t.Fatalf("nur %d Dateien in package main gefunden: Der Detektor liest den falschen Baum", programme)
	}
	sort.Strings(funde)
	for _, fund := range funde {
		t.Errorf("%s ist ein Leerimport in einer Bibliothek. Ihn in main.go oder in den Test setzen, der ihn braucht.", fund)
	}
}
