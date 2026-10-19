package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// Eine Hilfsfunktion, die einen Test führt (*testing.T, *testing.B, *testing.F, testing.TB),
// beginnt mit t.Helper(): Dann nennt eine Meldung aus ihr die Zeile, an der der Test sie ruft
// (Decisions, „Test helpers": „call t.Helper to attribute failures in the test helper to the
// line where the helper is called"). Den Parameter nimmt sie vorn, vor ihm steht nur ein Kontext
// (ebenda).
//
// Ausgenommen sind Rückrufe wie der Rumpf von t.Run, als Literal oder als Funktion, deren Name
// einem Aufruf übergeben wird, und die Funktionen in einer Tabelle von Fällen. Sie gehören zu
// ihrem Fall; mit t.Helper() nennte eine Meldung die Schleife, die den Fall ruft, statt der
// Zeile im Fall („Do not use this pattern when it obscures the connection between a test
// failure and the conditions that led to it", ebenda).
//
// Blindheit: Erkannt wird der Parameter an seinem Typ im Quelltext, nicht ein eigener Typ, der
// testing.T einbettet, und keine Methode, die t aus einem Feld nimmt. Eine Hilfsfunktion, die
// nichts melden kann, braucht nach Google kein t.Helper(), sondern keinen Parameter t; das
// prüft sie nicht.
func TestHilfsfunktionen_BeginnenMitHelper(t *testing.T) {
	funde, gezaehlt := hilfsfunktionenIn(t, leseGoQuellen(t, "."))
	if gezaehlt < 500 {
		t.Fatalf("nur %d Hilfsfunktionen gefunden: Der Sammler greift ins Leere", gezaehlt)
	}
	for _, f := range funde {
		t.Errorf("%s (docs/STIL.md, Abschnitt 10)", f)
	}
}

func TestHilfsfunktionen_Selbstprobe(t *testing.T) {
	quellen := map[string]string{
		"a/a_test.go": `package a

import (
	"context"
	pruef "testing"
)

func TestX(t *pruef.T) {
	t.Run("fall", func(t *pruef.T) { t.Fatal("x") })
	t.Run("benannt", fall)
	t.Run("auch gerufen", beides)
	beides(t)
	faelle := []struct{ pruefe func(t *pruef.T) }{{func(t *pruef.T) { t.Fatal("x") }}}
	ohne := func(t *pruef.T) { t.Fatal("x") }
	mit := func(t *pruef.T) { t.Helper() }
	_, _, _ = faelle, ohne, mit
}

func ohneHelper(t *pruef.T) { t.Fatal("x") }

func spaeterHelper(t *pruef.T) {
	t.Log("x")
	t.Helper()
}

func mitKontext(ctx context.Context, t *pruef.T) { t.Helper() }

func hinten(name string, t *pruef.T) { t.Helper() }

func vergleich(b *pruef.B) { b.Helper() }

func beide(tb pruef.TB) {}

func BenchmarkX(b *pruef.B) {}

func Testdaten(t *pruef.T) {}

func fall(t *pruef.T) { t.Fatal("x") }

func beides(t *pruef.T) { t.Fatal("x") }
`,
		"a/b.go": "package a\n\nimport \"testing\"\n\nvar lege = func(t *testing.T) {}\n",
	}
	funde, gezaehlt := hilfsfunktionenIn(t, quellen)
	erwartet := []string{
		"a/a_test.go:14: ohne beginnt nicht mit t.Helper()",
		"a/a_test.go:19: ohneHelper beginnt nicht mit t.Helper()",
		"a/a_test.go:21: spaeterHelper beginnt nicht mit t.Helper()",
		"a/a_test.go:28: hinten nimmt t nicht als ersten Parameter",
		"a/a_test.go:32: beide beginnt nicht mit tb.Helper()",
		"a/a_test.go:36: Testdaten beginnt nicht mit t.Helper()",
		"a/a_test.go:40: beides beginnt nicht mit t.Helper()",
		"a/b.go:5: lege beginnt nicht mit t.Helper()",
	}
	if strings.Join(funde, "\n") != strings.Join(erwartet, "\n") {
		t.Errorf("Funde:\n%s\nerwartet:\n%s", strings.Join(funde, "\n"), strings.Join(erwartet, "\n"))
	}
	if gezaehlt != 11 {
		t.Errorf("hilfsfunktionenIn zählt %d Hilfsfunktionen, erwartet 11", gezaehlt)
	}
}

// hilfsfunktionenIn liefert die Hilfsfunktionen, die nicht mit t.Helper() beginnen oder t nicht
// vorn nehmen, und die Zahl aller Hilfsfunktionen.
func hilfsfunktionenIn(t *testing.T, quellen map[string]string) ([]string, int) {
	t.Helper()
	pfade := make([]string, 0, len(quellen))
	for p := range quellen {
		pfade = append(pfade, p)
	}
	sort.Strings(pfade)
	fset := token.NewFileSet()
	dateien := map[string]*ast.File{}
	// Je Ordner die Namen, die einem Aufruf übergeben, und die, die gerufen werden: Eine Funktion,
	// die nur übergeben wird, ist ein Rückruf wie der Rumpf von t.Run.
	uebergeben := map[string]map[string]bool{}
	gerufen := map[string]map[string]bool{}
	for _, pfad := range pfade {
		datei, err := parser.ParseFile(fset, pfad, quellen[pfad], parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s lesen: %v", pfad, err)
		}
		dateien[pfad] = datei
		ordner := path.Dir(pfad)
		if uebergeben[ordner] == nil {
			uebergeben[ordner], gerufen[ordner] = map[string]bool{}, map[string]bool{}
		}
		ast.Inspect(datei, func(n ast.Node) bool {
			aufruf, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if name, ok := aufruf.Fun.(*ast.Ident); ok {
				gerufen[ordner][name.Name] = true
			}
			for _, argument := range aufruf.Args {
				if name, ok := argument.(*ast.Ident); ok {
					uebergeben[ordner][name.Name] = true
				}
			}
			return true
		})
	}
	rueckruf := func(ordner, name string) bool { return uebergeben[ordner][name] && !gerufen[ordner][name] }
	var funde []string
	gezaehlt := 0
	for _, pfad := range pfade {
		datei := dateien[pfad]
		paket := testingName(datei)
		if paket == "" {
			continue
		}
		pruefe := func(name string, typ *ast.FuncType, rumpf *ast.BlockStmt) {
			stelle, param := testParameter(typ, paket)
			if stelle < 0 {
				return
			}
			gezaehlt++
			zeile := fset.Position(typ.Pos()).Line
			if stelle > 1 || stelle == 1 && !istKontext(typ.Params.List[0].Type) {
				funde = append(funde, fmt.Sprintf("%s:%d: %s nimmt %s nicht als ersten Parameter", pfad, zeile, name, param))
			}
			if !beginntMitHelper(rumpf, param) {
				funde = append(funde, fmt.Sprintf("%s:%d: %s beginnt nicht mit %s.Helper()", pfad, zeile, name, param))
			}
		}
		test := strings.HasSuffix(pfad, "_test.go")
		ast.Inspect(datei, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if x.Body == nil || x.Recv == nil &&
					(test && istTestfunktion(x.Name.Name) || rueckruf(path.Dir(pfad), x.Name.Name)) {
					return true
				}
				pruefe(x.Name.Name, x.Type, x.Body)
			case *ast.AssignStmt:
				for i, wert := range x.Rhs {
					literal, ok := wert.(*ast.FuncLit)
					if !ok || i >= len(x.Lhs) {
						continue
					}
					if name, ok := x.Lhs[i].(*ast.Ident); ok {
						pruefe(name.Name, literal.Type, literal.Body)
					}
				}
			case *ast.ValueSpec:
				for i, wert := range x.Values {
					if literal, ok := wert.(*ast.FuncLit); ok && i < len(x.Names) {
						pruefe(x.Names[i].Name, literal.Type, literal.Body)
					}
				}
			}
			return true
		})
	}
	return funde, gezaehlt
}

// testingName liefert den Namen, unter dem eine Datei das Paket testing einbindet, sonst "".
func testingName(datei *ast.File) string {
	for _, einbindung := range datei.Imports {
		if pfad, err := strconv.Unquote(einbindung.Path.Value); err != nil || pfad != "testing" {
			continue
		}
		if einbindung.Name != nil {
			return einbindung.Name.Name
		}
		return "testing"
	}
	return ""
}

// testParameter liefert Stelle und Namen des Parameters, der den Test führt, sonst -1.
func testParameter(typ *ast.FuncType, paket string) (int, string) {
	stelle := 0
	for _, feld := range typ.Params.List {
		if fuehrtTest(feld.Type, paket) {
			if len(feld.Names) == 0 {
				return stelle, "_"
			}
			return stelle, feld.Names[0].Name
		}
		stelle += max(len(feld.Names), 1)
	}
	return -1, ""
}

// fuehrtTest erkennt *testing.T, *testing.B, *testing.F und testing.TB.
func fuehrtTest(typ ast.Expr, paket string) bool {
	if stern, ok := typ.(*ast.StarExpr); ok {
		auswahl, ok := stern.X.(*ast.SelectorExpr)
		return ok && istBezeichner(auswahl.X, paket) &&
			(auswahl.Sel.Name == "T" || auswahl.Sel.Name == "B" || auswahl.Sel.Name == "F")
	}
	auswahl, ok := typ.(*ast.SelectorExpr)
	return ok && istBezeichner(auswahl.X, paket) && auswahl.Sel.Name == "TB"
}

func istKontext(typ ast.Expr) bool {
	auswahl, ok := typ.(*ast.SelectorExpr)
	return ok && istBezeichner(auswahl.X, "context") && auswahl.Sel.Name == "Context"
}

func istBezeichner(ausdruck ast.Expr, name string) bool {
	bezeichner, ok := ausdruck.(*ast.Ident)
	return ok && bezeichner.Name == name
}

func beginntMitHelper(rumpf *ast.BlockStmt, param string) bool {
	if len(rumpf.List) == 0 {
		return false
	}
	anweisung, ok := rumpf.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	aufruf, ok := anweisung.X.(*ast.CallExpr)
	if !ok || len(aufruf.Args) != 0 {
		return false
	}
	auswahl, ok := aufruf.Fun.(*ast.SelectorExpr)
	return ok && istBezeichner(auswahl.X, param) && auswahl.Sel.Name == "Helper"
}

// istTestfunktion folgt der Regel von go test: Test, Benchmark, Fuzz oder Example, danach nichts
// oder kein Kleinbuchstabe.
func istTestfunktion(name string) bool {
	for _, vorsilbe := range []string{"Test", "Benchmark", "Fuzz", "Example"} {
		rest, ok := strings.CutPrefix(name, vorsilbe)
		if !ok {
			continue
		}
		erstes, _ := utf8.DecodeRuneInString(rest)
		if rest == "" || !unicode.IsLower(erstes) {
			return true
		}
	}
	return false
}
