package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Bibliothekscode legt keinen Kontext an, er reicht den des Rufers weiter. So verlangt es
// Googles Regelwerk für Go (Decisions, „Contexts"): „Unless you are implementing a server
// framework, you shouldn't create contexts with context.Background() in library code."
// context.Background() und context.TODO() stehen deshalb nur in Programmen (package main), in
// Tests und in den Prüfhilfen, die nur Tests einbinden. Muss Arbeit das Ende der Anfrage
// überdauern, nimmt sie context.WithoutCancel(ctx).
//
// Blindheit: das Paket context unter anderem Namen eingebunden. Ob ein übergebener Kontext der
// richtige ist, prüft contextcheck im Linter.
func TestKontext_NurProgrammeLegenIhnAn(t *testing.T) {
	fset := token.NewFileSet()
	var funde []string
	gelesen := 0
	for pfad, quelle := range leseGoQuellen(t, ".") {
		if strings.HasSuffix(pfad, "_test.go") || istPruefhilfe(pfad) {
			continue
		}
		datei, err := parser.ParseFile(fset, pfad, quelle, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s lesen: %v", pfad, err)
		}
		if datei.Name.Name == "main" {
			continue
		}
		gelesen++
		ast.Inspect(datei, func(knoten ast.Node) bool {
			aufruf, ok := knoten.(*ast.CallExpr)
			if !ok {
				return true
			}
			auswahl, ok := aufruf.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if paket, ok := auswahl.X.(*ast.Ident); ok && paket.Name == "context" &&
				(auswahl.Sel.Name == "Background" || auswahl.Sel.Name == "TODO") {
				funde = append(funde, fset.Position(aufruf.Pos()).String()+": context."+auswahl.Sel.Name+"()")
			}
			return true
		})
	}

	// Gegenprobe am Detektor: Ohne die Bibliotheken liest er den falschen Baum.
	if gelesen < 300 {
		t.Fatalf("nur %d Dateien von Bibliotheken gelesen: Der Detektor liest den falschen Baum", gelesen)
	}
	sort.Strings(funde)
	for _, fund := range funde {
		t.Errorf("%s in einer Bibliothek. Den Kontext des Rufers weiterreichen; muss die Arbeit das "+
			"Ende der Anfrage überdauern, context.WithoutCancel(ctx).", fund)
	}
}

// istPruefhilfe sagt, ob die Datei zu einer Prüfhilfe gehört (pruefhilfen_ratsche_test.go).
func istPruefhilfe(pfad string) bool {
	ordner := filepath.ToSlash(filepath.Dir(pfad))
	for _, hilfe := range pruefhilfen {
		if ordner == strings.TrimPrefix(hilfe, "bibliothek/") {
			return true
		}
	}
	return false
}
