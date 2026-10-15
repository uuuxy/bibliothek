package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Ein Typ nennt seinen Empfänger in jeder seiner Methoden gleich. So verlangt es Googles
// Regelwerk für Go (Decisions, „Receiver names"): „Applied consistently to every receiver for
// that type". Wer eine Methode liest, erkennt den Empfänger am Namen wieder.
//
// Ein Test neben dem Linter: Die Regel receiver-naming von revive vergleicht die Methoden
// einer Datei; ein Typ, dessen Methoden in mehreren Dateien stehen, bestand sie mit zwei Namen.
//
// Blindheit: Testdateien liest die Prüfung nicht. Ob der Name ein Kürzel des Typs ist, liest
// ein Mensch; die Türen der Themen heißen s wie am Server, von dem sie stammen.
func TestEmpfaenger_EinNameJeTyp(t *testing.T) {
	fset := token.NewFileSet()
	namen := map[string]map[string]string{} // Ordner.Typ → Name → erste Stelle
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
		datei, err := parser.ParseFile(fset, pfad, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		for _, dekl := range datei.Decls {
			fd, ok := dekl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) != 1 || len(fd.Recv.List[0].Names) != 1 {
				continue
			}
			name := fd.Recv.List[0].Names[0].Name
			if name == "_" {
				continue
			}
			typ := filepath.Dir(pfad) + "." + empfaengerTyp(fd.Recv.List[0].Type)
			if namen[typ] == nil {
				namen[typ] = map[string]string{}
			}
			if _, bekannt := namen[typ][name]; !bekannt {
				namen[typ][name] = fset.Position(fd.Pos()).String()
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}

	// Gegenprobe am Detektor: Findet er die Typen mit Methoden überhaupt?
	if len(namen) < 100 {
		t.Fatalf("nur %d Typen mit Methoden gefunden: Der Detektor greift ins Leere", len(namen))
	}
	typen := make([]string, 0, len(namen))
	for typ := range namen {
		typen = append(typen, typ)
	}
	sort.Strings(typen)
	for _, typ := range typen {
		if len(namen[typ]) < 2 {
			continue
		}
		var stellen []string
		for name, stelle := range namen[typ] {
			stellen = append(stellen, name+" ("+stelle+")")
		}
		sort.Strings(stellen)
		t.Errorf("%s nennt seinen Empfänger verschieden: %s. Einen Namen wählen und in jeder Methode nehmen.",
			typ, strings.Join(stellen, ", "))
	}
}

// empfaengerTyp liefert den Namen des Typs hinter einem Empfänger, auch hinter einem Zeiger
// und mit Typparametern.
func empfaengerTyp(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return empfaengerTyp(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return empfaengerTyp(t.X)
	case *ast.IndexListExpr:
		return empfaengerTyp(t.X)
	}
	return "?"
}
