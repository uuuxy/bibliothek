package service

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestMeldungZeigtNurDieMeldung: Die Theke zeigt err.Error() (api/action.go). Bis zum 28.09.2026
// las sie dort etwa „Fehler: ungültiger Transaktionszustand: Bitte scannen Sie zuerst einen
// Ausweis" oder „Fehler: conflict: dieses Exemplar wurde soeben bereits ausgeliehen" — die
// Bezeichnung der Fehlerart ist kein Satz für den Bildschirm (M3, Word choice: „UI text should be
// understandable by anyone, anywhere"). Die Art bleibt für errors.Is erhalten, auch eingewickelt,
// denn an ihr hängt der HTTP-Status.
func TestMeldungZeigtNurDieMeldung(t *testing.T) {
	for _, art := range []error{ErrInvalidState, ErrConflict, ErrNotFound} {
		err := meldung(art, "Buchexemplar %s ist ausgesondert", "4711")
		if soll := "Buchexemplar 4711 ist ausgesondert"; err.Error() != soll {
			t.Errorf("%v: Meldung %q, erwartet %q", art, err.Error(), soll)
		}
		if !errors.Is(err, art) || !errors.Is(fmt.Errorf("beim Ausleihen: %w", err), art) {
			t.Errorf("die Art %q muss erkennbar bleiben, auch eingewickelt", art)
		}
		if errors.Is(err, ErrBlocked) {
			t.Errorf("%v darf nicht als Sperre gelten", art)
		}
	}
}

// ohneBezeichnung sind die Fehlerarten, deren Text eine interne Bezeichnung ist. ErrBlocked fehlt
// mit Absicht: „die ausleihe ist gesperrt" ist der Kopf jeder Sperrmeldung (loan.go).
var ohneBezeichnung = map[string]bool{"ErrInvalidState": true, "ErrConflict": true, "ErrNotFound": true}

// TestFehlerartenNurUeberMeldung: Kein fmt.Errorf im Bestand wickelt eine dieser Arten ein —
// sonst stünde ihre Bezeichnung wieder vor der Meldung an der Theke. Am 28.09.2026 waren es 19
// Stellen in internal/service.
//
// Blindheit: Gesehen wird fmt.Errorf mit der Art als Argument, in Go-Dateien außer Tests
// (im Paket service als Name, anderswo als service.Name). errors.Join oder ein eigener Wrapper
// fallen nicht auf.
func TestFehlerartenNurUeberMeldung(t *testing.T) {
	fset := token.NewFileSet()
	gesehen := 0
	err := filepath.WalkDir("../..", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "frontend", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		datei, err := parser.ParseFile(fset, pfad, nil, 0)
		if err != nil {
			return err
		}
		imPaket := datei.Name.Name == "service"
		ast.Inspect(datei, func(n ast.Node) bool {
			aufruf, ok := n.(*ast.CallExpr)
			if !ok || !istFmtErrorf(aufruf.Fun) {
				return true
			}
			gesehen++
			for _, arg := range aufruf.Args[1:] {
				if art := fehlerart(arg, imPaket); ohneBezeichnung[art] {
					t.Errorf("%s: fmt.Errorf wickelt %s ein — die Theke zeigte dann dessen Bezeichnung vor "+
						"der Meldung; meldung(%s, …) benutzen", fset.Position(aufruf.Pos()), art, art)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if gesehen < 500 {
		t.Fatalf("nur %d fmt.Errorf-Aufrufe gesehen (am 28.09.2026: rund 690) — der Detektor läuft ins Leere", gesehen)
	}
}

func istFmtErrorf(fun ast.Expr) bool {
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Errorf" {
		return false
	}
	paket, ok := sel.X.(*ast.Ident)
	return ok && paket.Name == "fmt"
}

// fehlerart nennt den Namen eines Sentinels aus diesem Paket, sonst "".
func fehlerart(e ast.Expr, imPaket bool) string {
	switch x := e.(type) {
	case *ast.Ident:
		if imPaket {
			return x.Name
		}
	case *ast.SelectorExpr:
		if paket, ok := x.X.(*ast.Ident); ok && paket.Name == "service" {
			return x.Sel.Name
		}
	}
	return ""
}
