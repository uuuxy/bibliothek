package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"slices"
	"strings"
	"testing"

	"bibliothek/api/tuer"
)

// Jedes Thema bekommt die Basis des Servers als Zeiger. Mit einer Kopie arbeiteten seine Türen
// mit Datenbank, Anmeldung und Uhr vom Bau des Routers weiter, auch wenn am Server danach
// etwas gestellt wird.
func TestJederZugangReichtDieBasisDesServersWeiter(t *testing.T) {
	s := &Server{}
	zugaenge := map[string]*tuer.Basis{
		"benutzer":     s.benutzer().Basis,
		"druck":        s.druck().Basis,
		"inventurlauf": s.inventurlauf().Basis,
		"katalog":      s.katalog().Basis,
		"lernmittel":   s.lernmittel().Basis,
		"mahnwesen":    s.mahnwesen().Basis,
		"portal":       s.portal().Basis,
		"schaden":      s.schaden().Basis,
		"system":       s.system().Basis,
	}
	for name, basis := range zugaenge {
		if basis != &s.Basis {
			t.Errorf("s.%s() reicht nicht die Basis des Servers weiter: Die Türen des Themas sähen "+
				"nicht, was am Server nach dem Bau des Routers gestellt wird", name)
		}
	}

	erklaert := erklaerteZugaenge(t)
	geprueft := slices.Sorted(maps.Keys(zugaenge))
	if !slices.Equal(erklaert, geprueft) {
		t.Errorf("api/ erklärt die Zugänge %v, geprüft sind %v: die Liste in diesem Test nachziehen",
			erklaert, geprueft)
	}
}

// erklaerteZugaenge nennt die Methoden des Servers, die die Türen eines Themas liefern: mit
// einem Zeiger auf den Typ Tueren eines Pakets als einzigem Ergebnis.
func erklaerteZugaenge(t *testing.T) []string {
	t.Helper()
	var namen []string
	fset := token.NewFileSet()
	for _, pfad := range apiDateien(t) {
		if strings.Contains(pfad, "/") {
			continue
		}
		datei, err := parser.ParseFile(fset, pfad, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s nicht lesbar: %v", pfad, err)
		}
		for _, erklaerung := range datei.Decls {
			fn, ok := erklaerung.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
				continue
			}
			stern, ok := fn.Type.Results.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			if typ, ok := stern.X.(*ast.SelectorExpr); ok && typ.Sel.Name == "Tueren" {
				namen = append(namen, fn.Name.Name)
			}
		}
	}
	slices.Sort(namen)
	return namen
}
