package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Ein Test setzt keine Variable der Paketebene um, weder des eigenen Pakets noch eines fremden
// (paket.Name = …). Googles Regelwerk nennt solchen Zustand unsicher, wenn „independent test
// cases interact with each other through global state" (Best Practices, „Global state"): Der
// Test, der den Wert umsetzt, verändert jeden Test nach ihm, und ein vergessenes Zurücksetzen
// macht eine Prüfung grün oder rot je nach Reihenfolge der Dateien. Was ein Test austauschen
// will, gehört einem Typ, den der Rufer anlegt (tuer.Basis mit Mail und Rechte,
// katalog.Tueren mit cover, auth.Anmeldung mit imapTLS).
//
// Blindheit: Gesehen werden Zuweisungen, auch zusammengesetzte (+=) und ++/--, an einen Namen,
// den eine Programmdatei desselben Ordners auf Paketebene erklärt, und an paket.Name eines
// eingebundenen Pakets. Nicht gesehen werden Funktionen, die einen Wert des Pakets setzen
// (clientip.SetzeProxys), Schreiben über einen Zeiger oder eine Map der Paketebene, die ein Test
// befüllt, und Variablen, die eine Testdatei selbst erklärt.
func TestPaketzustand_TestsSetzenKeineVariableDesPakets(t *testing.T) {
	funde, gelesen := paketzustandFunde(t, ".")
	if gelesen < 500 {
		t.Fatalf("nur %d Testdateien gelesen: Der Sammler greift ins Leere", gelesen)
	}
	for _, f := range funde {
		t.Errorf("%s setzt eine Variable der Paketebene um. Der Wert gehört einem Typ, den der "+
			"Rufer anlegt und der Test selbst baut (docs/STIL.md, Abschnitt 10)", f)
	}
}

func TestPaketzustand_Selbstprobe(t *testing.T) {
	quellen := map[string]string{
		"a/a.go": "package a\nvar zaehler int\nvar Lader func()\n",
		"a/a_test.go": `package a
import (
	"os"
	q "bibliothek/quelle"
)
var eigene int
func TestX(t *testing.T) {
	zaehler = 1
	zaehler += 2
	zaehler++
	os.Stdout = nil
	q.Wert, eigene = 3, 4
	eigene = 5
	zaehler := 7
	zaehler = 8
	var lokal struct{ Lader func() }
	lokal.Lader = nil
}
func hilfe(Lader func()) { Lader = nil }
`,
	}
	funde := paketzustandIn(t, quellen)
	erwartet := []string{
		"a/a_test.go:8: zaehler", "a/a_test.go:9: zaehler", "a/a_test.go:10: zaehler",
		"a/a_test.go:11: os.Stdout", "a/a_test.go:12: q.Wert",
	}
	if strings.Join(funde, "\n") != strings.Join(erwartet, "\n") {
		t.Errorf("Funde:\n%s\nerwartet:\n%s", strings.Join(funde, "\n"), strings.Join(erwartet, "\n"))
	}
}

// paketzustandFunde liest alle Go-Dateien unter wurzel und liefert die Funde mit der Zahl der
// gelesenen Testdateien.
func paketzustandFunde(t *testing.T, wurzel string) ([]string, int) {
	t.Helper()
	quellen := map[string]string{}
	tests := 0
	err := filepath.WalkDir(wurzel, func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "node_modules" || name == "frontend" || name == "testdata" ||
				(strings.HasPrefix(name, ".") && pfad != wurzel) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		quellen[filepath.ToSlash(pfad)] = string(inhalt)
		if strings.HasSuffix(pfad, "_test.go") {
			tests++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}
	return paketzustandIn(t, quellen), tests
}

// paketzustandIn prüft die Testdateien gegen die Variablen, die die Programmdateien desselben
// Ordners auf Paketebene erklären.
func paketzustandIn(t *testing.T, quellen map[string]string) []string {
	t.Helper()
	pfade := make([]string, 0, len(quellen))
	for p := range quellen {
		pfade = append(pfade, p)
	}
	sort.Strings(pfade)
	fset := token.NewFileSet()
	paketVariablen := map[string]map[string]bool{}
	for _, p := range pfade {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		datei, err := parser.ParseFile(fset, p, quellen[p], parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		ordner := filepath.Dir(p)
		if paketVariablen[ordner] == nil {
			paketVariablen[ordner] = map[string]bool{}
		}
		for _, dekl := range datei.Decls {
			if g, ok := dekl.(*ast.GenDecl); ok && g.Tok == token.VAR {
				for _, spez := range g.Specs {
					werte, ok := spez.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, n := range werte.Names {
						paketVariablen[ordner][n.Name] = true
					}
				}
			}
		}
	}
	var funde []string
	for _, p := range pfade {
		if !strings.HasSuffix(p, "_test.go") {
			continue
		}
		// Mit Auflösung der Namen: Ein Name, den die Datei selbst erklärt (lokal oder auf ihrer
		// Paketebene), trägt sein Objekt; ein Name aus einer Programmdatei bleibt offen.
		datei, err := parser.ParseFile(fset, p, quellen[p], 0)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		importe := map[string]bool{}
		for _, imp := range datei.Imports {
			pfad, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("%s: Pfad des Imports %s: %v", p, imp.Path.Value, err)
			}
			name := filepath.Base(pfad)
			if imp.Name != nil {
				name = imp.Name.Name
			}
			importe[name] = true
		}
		variablen := paketVariablen[filepath.Dir(p)]
		pruefe := func(ziel ast.Expr) {
			switch x := ziel.(type) {
			case *ast.SelectorExpr:
				if id, ok := x.X.(*ast.Ident); ok && id.Obj == nil && importe[id.Name] {
					funde = append(funde, p+":"+strconv.Itoa(fset.Position(x.Pos()).Line)+": "+id.Name+"."+x.Sel.Name)
				}
			case *ast.Ident:
				if x.Obj == nil && variablen[x.Name] {
					funde = append(funde, p+":"+strconv.Itoa(fset.Position(x.Pos()).Line)+": "+x.Name)
				}
			}
		}
		ast.Inspect(datei, func(n ast.Node) bool {
			switch s := n.(type) {
			case *ast.AssignStmt:
				if s.Tok != token.DEFINE {
					for _, ziel := range s.Lhs {
						pruefe(ziel)
					}
				}
			case *ast.IncDecStmt:
				pruefe(s.X)
			}
			return true
		})
	}
	return funde
}
