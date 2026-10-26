package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// Eine Sperre entsteht über service.Sperre. Ihr Grund ist dort ein Feld, und service.MeldungAus
// baut daraus die Meldung der Theke: „Die Ausleihe ist gesperrt: …", groß begonnen, während der
// Fehlertext klein bleibt (docs/STIL.md 11). Hängt ein Grund mit fmt.Errorf an ErrGesperrt,
// steht er nur im Fehlertext, und die Theke zeigt den Kopf allein.
//
// Blindheit: ErrGesperrt unter anderem Namen (eine Variable, die ihn hält) und ein Grund, der
// über errors.Join oder einen eigenen Fehlertyp an ErrGesperrt hängt. Testdateien liest die
// Prüfung nicht.
func TestSperre_GrundNurUeberSperre(t *testing.T) {
	quellen := leseGoQuellen(t, ".")
	fset := token.NewFileSet()
	var funde []string
	errorfAufrufe := 0
	for pfad, quelle := range quellen {
		if strings.HasSuffix(pfad, "_test.go") {
			continue
		}
		datei, err := parser.ParseFile(fset, pfad, quelle, 0)
		if err != nil {
			t.Fatalf("%s lesen: %v", pfad, err)
		}
		for _, pos := range grundAmSperrfehler(datei) {
			funde = append(funde, fset.Position(pos).String())
		}
		errorfAufrufe += zaehleErrorf(datei)
	}

	// Nicht-leer-Garantie: Der Server ruft fmt.Errorf an Hunderten Stellen. Findet der Durchlauf
	// kaum welche, liest er den falschen Baum.
	if errorfAufrufe < 100 {
		t.Fatalf("nur %d Aufrufe von fmt.Errorf gelesen: Der Detektor liest den falschen Baum", errorfAufrufe)
	}
	sort.Strings(funde)
	for _, fund := range funde {
		t.Errorf("%s: Grund mit fmt.Errorf an ErrGesperrt. Die Theke zeigt ihn nicht; service.Sperre(\"…\") nehmen.", fund)
	}
}

// istErrorf erkennt fmt.Errorf(…).
func istErrorf(n ast.Node) (*ast.CallExpr, bool) {
	aufruf, ok := n.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	sel, ok := aufruf.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Errorf" {
		return nil, false
	}
	paket, ok := sel.X.(*ast.Ident)
	return aufruf, ok && paket.Name == "fmt"
}

// grundAmSperrfehler liefert die Stellen, an denen fmt.Errorf ErrGesperrt einwickelt.
func grundAmSperrfehler(datei ast.Node) []token.Pos {
	var funde []token.Pos
	ast.Inspect(datei, func(n ast.Node) bool {
		aufruf, ok := istErrorf(n)
		if !ok {
			return true
		}
		for _, arg := range aufruf.Args {
			switch a := arg.(type) {
			case *ast.Ident:
				ok = a.Name == "ErrGesperrt"
			case *ast.SelectorExpr:
				ok = a.Sel.Name == "ErrGesperrt"
			default:
				ok = false
			}
			if ok {
				funde = append(funde, aufruf.Pos())
			}
		}
		return true
	})
	return funde
}

func zaehleErrorf(datei ast.Node) int {
	anzahl := 0
	ast.Inspect(datei, func(n ast.Node) bool {
		if _, ok := istErrorf(n); ok {
			anzahl++
		}
		return true
	})
	return anzahl
}

func TestSperre_DetektorFasstDieFormen(t *testing.T) {
	for quelle, erwartet := range map[string]int{
		`return fmt.Errorf("%w: Gerät ist aktuell gesperrt", ErrGesperrt)`:                   1,
		`return fmt.Errorf("%w: Manuelle Sperre", service.ErrGesperrt)`:                      1,
		`return UebergehbareSperre(fmt.Errorf("%w: %d offen", ErrGesperrt, offen))`:          1,
		`return Sperre("Gerät ist aktuell gesperrt")`:                                        0,
		`return fmt.Errorf("ausleihen: %w", err)`:                                            0,
		`if errors.Is(err, ErrGesperrt) { return fmt.Errorf("theke: %w", err) }; return nil`: 0,
	} {
		datei, err := parser.ParseFile(token.NewFileSet(), "probe.go", "package p\nfunc f() error {\n"+quelle+"\n}", 0)
		if err != nil {
			t.Fatalf("%s: %v", quelle, err)
		}
		if got := len(grundAmSperrfehler(datei)); got != erwartet {
			t.Errorf("%s: %d Funde, erwartet %d", quelle, got, erwartet)
		}
	}
}
