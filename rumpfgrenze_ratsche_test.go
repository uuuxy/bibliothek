package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Quelltext-Ratsche: Eine Tür liest ihren JSON-Rumpf über api/tuer (LeseAnfrage,
// LeseAnfrageStreng, DekodiereRumpf), die ihn auf maxRumpfJSON begrenzen. Ein eigener
// Decoder am Rumpf hätte nur die Grenze aller Anfragen von 100 MB, und der Decoder hält den Rumpf
// im Speicher. Erlaubt ist ein Decoder über Daten, die schon im Speicher liegen.
//
// Blind für einen Rumpf, der über einen anderen Leser als json.NewDecoder und io.ReadAll gelesen
// wird, etwa io.Copy in einen Puffer.
func TestRumpfgrenze_KeineTuerLiestOhneGrenze(t *testing.T) {
	var fundstellen []string
	ueberTuer := 0
	err := filepath.Walk("api", func(pfad string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") ||
			strings.HasPrefix(pfad, filepath.Join("api", "tuer")+string(filepath.Separator)) {
			return err
		}
		dateisatz := token.NewFileSet()
		datei, err := parser.ParseFile(dateisatz, pfad, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(datei, func(n ast.Node) bool {
			aufruf, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch paketFunktion(aufruf.Fun) {
			case "tuer.LeseAnfrage", "tuer.LeseAnfrageStreng", "tuer.DekodiereRumpf":
				ueberTuer++
			case "json.NewDecoder":
				if len(aufruf.Args) == 1 && !liegtImSpeicher(aufruf.Args[0]) {
					fundstellen = append(fundstellen, dateisatz.Position(aufruf.Pos()).String())
				}
			case "io.ReadAll":
				if len(aufruf.Args) == 1 && istRumpf(aufruf.Args[0]) {
					fundstellen = append(fundstellen, dateisatz.Position(aufruf.Pos()).String())
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("Quellbaum lesen: %v", err)
	}
	// Über 60 Türen lesen ihren Rumpf über api/tuer. Sieht der Lauf weniger, erreicht er sie nicht
	// und fände dort auch keinen eigenen Decoder.
	if ueberTuer < 60 {
		t.Fatalf("nur %d Aufrufe von LeseAnfrage, LeseAnfrageStreng und DekodiereRumpf unter api/: "+
			"Der Lauf erreicht die Türen nicht", ueberTuer)
	}
	if len(fundstellen) > 0 {
		t.Errorf("Rumpf ohne die Grenze aus api/tuer gelesen, bitte tuer.DekodiereRumpf nehmen:\n%s",
			strings.Join(fundstellen, "\n"))
	}
}

// paketFunktion nennt einen Aufruf der Form paket.Funktion, sonst "".
func paketFunktion(fun ast.Expr) string {
	auswahl, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	paket, ok := auswahl.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return paket.Name + "." + auswahl.Sel.Name
}

// liegtImSpeicher sagt, ob ein Leser über Daten im Speicher läuft (bytes.NewReader,
// strings.NewReader).
func liegtImSpeicher(arg ast.Expr) bool {
	aufruf, ok := arg.(*ast.CallExpr)
	if !ok {
		return false
	}
	name := paketFunktion(aufruf.Fun)
	return name == "bytes.NewReader" || name == "strings.NewReader"
}

// istRumpf sagt, ob ein Ausdruck der Rumpf einer Anfrage ist (….Body).
func istRumpf(arg ast.Expr) bool {
	auswahl, ok := arg.(*ast.SelectorExpr)
	return ok && auswahl.Sel.Name == "Body"
}
