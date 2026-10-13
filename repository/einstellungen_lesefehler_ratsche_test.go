package repository_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// Wer die Einstellungen liest, behandelt den Lesefehler.
//
// GetSettings und EinstellungenUeber liefern bei einem Lesefehler die Vorgaben des Programms
// zusammen mit dem Fehler. Wer den Fehler mit `_` verwirft, arbeitet mit den Vorgaben weiter,
// als wären sie eingestellt: An sechs Türen entstanden so Blätter und eine Bestellmail ohne
// die Angaben der Schule (Bestellbericht, Bestellmail, Auskunft, Mahnbrief, Elternbrief,
// Ersatzforderung). Der Linter meldet die Stelle, eine Zeile `//nolint:errcheck` stellt ihn ab;
// diese Prüfung lässt sich so nicht abstellen.
//
// Blind für: einen Fehler, der einer Variable zugewiesen und danach nicht geprüft wird, und
// für eine eigene Funktion um das Lesen, die den Fehler selbst schluckt.
func TestEinstellungenLesefehlerWirdNichtVerworfen(t *testing.T) {
	leser := map[string]bool{"GetSettings": true, "EinstellungenUeber": true}
	fset := token.NewFileSet()
	gelesen, verworfen := 0, []string{}

	err := filepath.WalkDir("..", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "node_modules" || name == ".git" || name == "frontend" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		datei, err := parser.ParseFile(fset, pfad, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		ast.Inspect(datei, func(n ast.Node) bool {
			zuweisung, ok := n.(*ast.AssignStmt)
			if !ok || len(zuweisung.Rhs) != 1 {
				return true
			}
			aufruf, ok := zuweisung.Rhs[0].(*ast.CallExpr)
			if !ok {
				return true
			}
			var name string
			switch f := aufruf.Fun.(type) {
			case *ast.SelectorExpr:
				name = f.Sel.Name
			case *ast.Ident:
				name = f.Name
			}
			if !leser[name] || len(zuweisung.Lhs) != 2 {
				return true
			}
			gelesen++
			if fehler, ok := zuweisung.Lhs[1].(*ast.Ident); ok && fehler.Name == "_" {
				verworfen = append(verworfen, fset.Position(zuweisung.Pos()).String())
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}

	// Gegenprobe am Detektor: Findet er die Leser überhaupt?
	if gelesen < 20 {
		t.Fatalf("nur %d Stellen gefunden, die die Einstellungen lesen: Der Detektor greift ins Leere", gelesen)
	}
	t.Logf("%d Stellen lesen die Einstellungen", gelesen)
	for _, stelle := range verworfen {
		t.Errorf("%s verwirft den Fehler beim Lesen der Einstellungen und arbeitet mit den Vorgaben weiter. "+
			"Den Fehler behandeln: die Antwort verweigern oder die Vorgabe ausdrücklich wählen.", stelle)
	}
}
