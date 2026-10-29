package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Prüfung: Ein Serverfehler steht einmal im Log (docs/STIL.md, Abschnitt 11).
//
// Warum: Antwortet eine Tür mit 5xx, schreibt logMiddleware die Zeile zum Status mit Route und
// der Ursache, die apierrors ihr übergibt. Schreibt die Tür die Ursache vorher selbst mit
// slog.Error und antwortet ohne sie, steht jede Störung zweimal unter „Fehler im Log" und in der
// Alarm-Mail: die Ursache ohne Route und die Route ohne Ursache.
//
// Regel: In einem Block folgt auf einen Aufruf von slog.Error oder slog.ErrorContext kein Aufruf
// von apierrors.AntworteFehler oder apierrors.AntworteFehlerMitMeldung mit einem Status 5xx. Die
// Ursache gehört in die Antwort: `apierrors.AntworteFehler(w, http.StatusInternalServerError,
// fmt.Errorf("klassen laden: %w", err))`.
//
// Sieht nicht: einen Logger aus einer Variablen, eine Zeile in einer aufgerufenen Funktion und
// eine Antwort über MitFehlerantwort; einen Status aus einer Variablen.
var einmalImLogBestand = map[string]string{
	// „datei:funktion": Grund.
	"api/middleware.go:panikMiddleware": "die Panik steht mit Stack in ihrer eigenen Zeile; die Antwort trägt mit Absicht keine Ursache",
}

// fuenfhundert sind die Status, deren Antwort logMiddleware als Fehler schreibt.
var fuenfhundert = map[string]bool{
	"StatusInternalServerError": true, "StatusNotImplemented": true, "StatusBadGateway": true,
	"StatusServiceUnavailable": true, "StatusGatewayTimeout": true,
}

// istAufruf meldet, ob der Ausdruck ein Aufruf paket.name(…) mit einem der Namen ist.
func istAufruf(ausdruck ast.Expr, paket string, namen ...string) (*ast.CallExpr, bool) {
	aufruf, ok := ausdruck.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	sel, ok := aufruf.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}
	links, ok := sel.X.(*ast.Ident)
	if !ok || links.Name != paket {
		return nil, false
	}
	for _, name := range namen {
		if sel.Sel.Name == name {
			return aufruf, true
		}
	}
	return nil, false
}

// antwortetMitFuenfhundert meldet eine Antwort von apierrors mit einem Status 5xx.
func antwortetMitFuenfhundert(ausdruck ast.Expr) bool {
	aufruf, ok := istAufruf(ausdruck, "apierrors", "AntworteFehler", "AntworteFehlerMitMeldung")
	if !ok || len(aufruf.Args) < 2 {
		return false
	}
	switch status := aufruf.Args[1].(type) {
	case *ast.SelectorExpr:
		return fuenfhundert[status.Sel.Name]
	case *ast.BasicLit:
		return strings.HasPrefix(status.Value, "5")
	}
	return false
}

// doppeltImLog liefert je Block die Stelle einer Antwort 5xx, vor der dieselbe Funktion den
// Fehler schon mit slog.Error geschrieben hat.
func doppeltImLog(rumpf *ast.BlockStmt, fset *token.FileSet) []string {
	var stellen []string
	ast.Inspect(rumpf, func(n ast.Node) bool {
		block, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}
		geloggt := false
		for _, anweisung := range block.List {
			ausdruck, ok := anweisung.(*ast.ExprStmt)
			if !ok {
				continue
			}
			if _, ok := istAufruf(ausdruck.X, "slog", "Error", "ErrorContext"); ok {
				geloggt = true
				continue
			}
			if geloggt && antwortetMitFuenfhundert(ausdruck.X) {
				stellen = append(stellen, fset.Position(ausdruck.Pos()).String())
			}
		}
		return true
	})
	return stellen
}

func TestServerfehlerStehtEinmalImLog(t *testing.T) {
	fset := token.NewFileSet()
	treffer := map[string][]string{}
	err := filepath.WalkDir(".", func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if eintrag.IsDir() {
			switch eintrag.Name() {
			case "frontend", "node_modules", ".git", "e2e":
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
		for _, decl := range datei.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if stellen := doppeltImLog(fn.Body, fset); len(stellen) > 0 {
				treffer[pfad+":"+fn.Name.Name] = stellen
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var neu []string
	for schluessel, stellen := range treffer {
		if _, ok := einmalImLogBestand[schluessel]; !ok {
			neu = append(neu, stellen...)
		}
	}
	sort.Strings(neu)
	if len(neu) > 0 {
		t.Errorf("Serverfehler zweimal im Log: slog.Error, danach eine Antwort 5xx ohne die Ursache:\n  %s\n"+
			"Fix: die Zeile mit slog.Error streichen und die Ursache an die Antwort geben, "+
			"etwa fmt.Errorf(\"klassen laden: %%w\", err) (docs/STIL.md 11).",
			strings.Join(neu, "\n  "))
	}
	for schluessel := range einmalImLogBestand {
		if _, ok := treffer[schluessel]; !ok {
			t.Errorf("%s schreibt keinen Fehler mehr doppelt; aus einmalImLogBestand streichen.", schluessel)
		}
	}
}

// Gegenprobe am Detektor: Fände er die Form nicht, wäre die Prüfung grün, ohne zu prüfen.
func TestEinmalImLogDetektorErkenntDieForm(t *testing.T) {
	quelle := `package p
func doppelt(w http.ResponseWriter) { _, err := f(); if err != nil { slog.Error("laden", "fehler", err); apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("x")); return } }
func doppelt2(w http.ResponseWriter) { if err := f(); err != nil { slog.ErrorContext(ctx, "laden", "fehler", err); apierrors.AntworteFehlerMitMeldung(w, 503, "Satz", nil) } }
func einmal(w http.ResponseWriter) { if err := f(); err != nil { apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("laden: %w", err)) } }
func vierhundert(w http.ResponseWriter) { if err := f(); err != nil { slog.Error("x", "fehler", err); apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Satz", nil) } }
func warnung(w http.ResponseWriter) { if err := f(); err != nil { slog.Warn("x", "fehler", err); apierrors.AntworteFehler(w, http.StatusInternalServerError, err) } }
`
	fset := token.NewFileSet()
	datei, err := parser.ParseFile(fset, "probe.go", quelle, 0)
	if err != nil {
		t.Fatal(err)
	}
	gefunden := map[string]bool{}
	for _, decl := range datei.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			gefunden[fn.Name.Name] = len(doppeltImLog(fn.Body, fset)) > 0
		}
	}
	soll := map[string]bool{"doppelt": true, "doppelt2": true, "einmal": false, "vierhundert": false, "warnung": false}
	for name, erwartet := range soll {
		if gefunden[name] != erwartet {
			t.Errorf("%s: gefunden %v, erwartet %v", name, gefunden[name], erwartet)
		}
	}
	if t.Failed() {
		t.Log(fmt.Sprint(gefunden))
	}
}
