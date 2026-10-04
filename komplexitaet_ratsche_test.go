package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/uudashr/gocognit"
)

// komplexitaetsGrenze ist die Schwelle von SonarQube go:S3776: Gemeldet wird ab 16.
const komplexitaetsGrenze = 15

// Keine Produktionsfunktion liegt über der Grenze (Cognitive Complexity). Gemessen wird mit
// der Bibliothek hinter dem Werkzeug gocognit; dessen Zahlen deckten sich mit dem Scan von
// SonarQube. Eine Bestandsliste gibt es nicht: Die Grenze gilt für jede Funktion.
//
// Ein Test und kein Eintrag in .golangci.yml: Der Linter liest keine Datei mit
// `//go:build ignore`, und `//nolint` oder `//gocognit:ignore` schalten ihn für eine Funktion
// stumm. Dieser Test liest jede Datei und kennt keine Ausnahme per Kommentar.
//
// Reparatur bei Rot: zusammenhängende Prüfungen in einen Helfer ziehen, der die Antwort selbst
// schreibt. Bei einem Handler den Rumpf der Closure als Methode führen (`return s.handleX`):
// In der Closure kostet jede Prüfung 2, in der Methode 1.
//
// BLINDHEIT: Misst mit gocognit, nicht mit SonarQube; eine neue Version einer der beiden Seiten
// kann anders zählen. Testdateien sind mit Absicht ausgenommen, wie in
// sonar-project.properties; läuft beides auseinander, merkt es dieser Test nicht. Ein
// Funktionsliteral auf Paketebene zählt als eigene Funktion; ob SonarQube es ebenso wertet,
// ist nicht gemessen.
func TestKomplexitaet_KeineProduktionsfunktionUeberDerGrenze(t *testing.T) {
	fset := token.NewFileSet()
	dateien := 0
	var gemessen []gemesseneFunktion
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
		inDatei, err := messeKomplexitaet(fset, pfad, nil)
		if err != nil {
			return err
		}
		dateien++
		gemessen = append(gemessen, inDatei...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Nicht-leer-Garantie: Liest der Sammler kaum Dateien, misst er nichts.
	if dateien < 300 || len(gemessen) < 1500 {
		t.Fatalf("nur %d Dateien und %d Funktionen gemessen — der Sammler greift nicht mehr", dateien, len(gemessen))
	}
	var zeilen []string
	for _, f := range gemessen {
		if f.ueberDerGrenze() {
			zeilen = append(zeilen, fmt.Sprintf("%s %s: %d", f.ort, f.name, f.wert))
		}
	}
	if len(zeilen) > 0 {
		sort.Strings(zeilen)
		t.Errorf("%d Funktion(en) über %d (Cognitive Complexity, SonarQube go:S3776):\n  %s\n"+
			"Fix: zusammenhängende Prüfungen in einen Helfer ziehen; bei einem Handler den Rumpf der "+
			"Closure als Methode führen. Eine Ausnahmeliste gibt es nicht.",
			len(zeilen), komplexitaetsGrenze, strings.Join(zeilen, "\n  "))
	}
}

type gemesseneFunktion struct {
	ort  string
	name string
	wert int
}

func (f gemesseneFunktion) ueberDerGrenze() bool { return f.wert > komplexitaetsGrenze }

// messeKomplexitaet misst jede Funktionsdeklaration und jedes Funktionsliteral auf Paketebene
// einer Datei; quelle wie bei parser.ParseFile. Gemessen wird die Deklaration selbst und nicht
// über gocognit.ComplexityStats: Das überginge eine Funktion mit `//gocognit:ignore`.
func messeKomplexitaet(fset *token.FileSet, pfad string, quelle any) ([]gemesseneFunktion, error) {
	// Mit Namensauflösung, also ohne parser.SkipObjectResolution: Sonst hält gocognit den
	// Aufruf einer gleichnamigen Funktion für Rekursion und zählt einen Punkt zu viel.
	datei, err := parser.ParseFile(fset, pfad, quelle, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var gemessen []gemesseneFunktion
	nimm := func(name string, pos token.Pos, fn *ast.FuncDecl) {
		p := fset.Position(pos)
		gemessen = append(gemessen, gemesseneFunktion{
			ort:  fmt.Sprintf("%s:%d", filepath.ToSlash(p.Filename), p.Line),
			name: name,
			wert: gocognit.Complexity(fn),
		})
	}
	for _, decl := range datei.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			name := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) > 0 {
				name = "(" + types.ExprString(d.Recv.List[0].Type) + ")." + name
			}
			nimm(name, d.Pos(), d)
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, wert := range vs.Values {
					name := "Literal in " + vs.Names[min(i, len(vs.Names)-1)].Name
					ast.Inspect(wert, func(n ast.Node) bool {
						lit, ok := n.(*ast.FuncLit)
						if !ok {
							return true
						}
						nimm(name, lit.Pos(), &ast.FuncDecl{Name: ast.NewIdent(name), Type: lit.Type, Body: lit.Body})
						// Ein Literal im Literal zählt in dessen Wert mit.
						return false
					})
				}
			}
		}
	}
	return gemessen, nil
}

// Gegenprobe am Detektor: die Grenze selbst, die Formen, in denen eine Funktion vorkommt, die
// beiden Kommentare, die andere Werkzeuge stumm schalten, und der Aufruf einer gleichnamigen
// Funktion, der keine Rekursion ist.
func TestKomplexitaet_DetektorMisstDieFormen(t *testing.T) {
	bedingungen := func(n int) string { return strings.Repeat("\tif a {\n\t}\n", n) }
	quelle := "package p\n" +
		"func anDerGrenze(a bool) {\n" + bedingungen(15) + "}\n" +
		"func darueber(a bool) {\n" + bedingungen(16) + "}\n" +
		"func (s *Server) methode(a bool) {\n" + bedingungen(16) + "}\n" +
		"func huelle(a bool) func() {\n\treturn func() {\n" + bedingungen(8) + "\t}\n}\n" +
		"func huelleDarunter(a bool) func() {\n\treturn func() {\n" + bedingungen(7) + "\t}\n}\n" +
		"//gocognit:ignore\nfunc stumm(a bool) { //nolint:gocognit\n" + bedingungen(16) + "}\n" +
		"var paketebene = func(a bool) {\n" + bedingungen(16) + "}\n" +
		"var sofort = func(a bool) bool {\n" + bedingungen(16) + "\treturn a\n}(true)\n" +
		"var tabelle = map[string]func(bool){\"x\": func(a bool) {\n" + bedingungen(16) + "}}\n" +
		"func gleicherName(a bool) bool { return a }\n" +
		"func (s *Server) gleicherName(a bool) bool { return gleicherName(a) }\n"
	gemessen, err := messeKomplexitaet(token.NewFileSet(), "probe.go", quelle)
	if err != nil {
		t.Fatal(err)
	}
	var gemeldet []string
	for _, f := range gemessen {
		zeile := fmt.Sprintf("%s=%d", f.name, f.wert)
		if f.ueberDerGrenze() {
			zeile += " über der Grenze"
		}
		gemeldet = append(gemeldet, zeile)
	}
	erwartet := []string{
		"anDerGrenze=15",
		"darueber=16 über der Grenze",
		"(*Server).methode=16 über der Grenze",
		"huelle=16 über der Grenze",
		"huelleDarunter=14",
		"stumm=16 über der Grenze",
		"Literal in paketebene=16 über der Grenze",
		"Literal in sofort=16 über der Grenze",
		"Literal in tabelle=16 über der Grenze",
		"gleicherName=0",
		"(*Server).gleicherName=0",
	}
	if strings.Join(gemeldet, "|") != strings.Join(erwartet, "|") {
		t.Errorf("Detektor misst\n  %s\nerwartet\n  %s", strings.Join(gemeldet, "\n  "), strings.Join(erwartet, "\n  "))
	}
}
