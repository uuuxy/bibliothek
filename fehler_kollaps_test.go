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

// Prüfung gegen den Kollaps von Fehlern.
//
// Warum: Eine Bedingung, die einen echten Fehler (err != nil) und einen fachlichen Nicht-Treffer
// in dieselbe harmlose Antwort (404/401/403) presst, verkleidet einen Ausfall als Bedienfehler.
// Ein Fehler beim Lesen (NULL in string, abweichendes Schema) erscheint dann als „nicht
// gefunden", die Diagnose stirbt, und der Betrieb sieht einen Bedienfehler statt eines kaputten
// Systems. Die Form: `if err != nil || !exists { 404 }`.
//
// Regel: Ein `if`, dessen Bedingung `<err> != nil` enthält und dessen Rumpf mit
// StatusNotFound/StatusUnauthorized/StatusForbidden antwortet, muss den Fehler in der Bedingung
// einordnen (errors.Is / errors.As / ein Err*-Sentinel). Sonst ist es ein Kollaps. Konservativ:
// Nur die Bedingung wird gelesen, nicht der Rumpf; ein `errors.Is` im Rumpf gilt nicht, weil dann
// der Rest des Rumpfs trotzdem alles andere in dieselbe Antwort kippt.
//
// Die übliche Form im Haus, `if err != nil { if errors.Is(err, ErrNoRows) { 404 }; 500 }`, ordnet
// im ersten verschachtelten if ein; die gilt als sauber.
//
// Reparatur bei Rot: `errors.Is(err, pgx.ErrNoRows)` bzw. den Sentinel prüfen und den Rest als
// 500 (oder eingepackt) weiterreichen, wie inventur/titel_metadaten.go.
var kollapsBestand = map[string]string{
	// „datei:funktion": Einordnung. Die Stellen hier sind vertretbar; der Fehler ist dort die
	// negative Antwort.
	"api/csrf.go:csrfMiddleware": "Validierungsfehler des Double-Submit = 403; kein DB-Zugriff",
	// Die drei Stellen mit dem Token lesen über PruefeToken die Datenbank (Sperrliste, Kontostatus).
	// Die Zeile davor ordnet ErrPruefungGestoert als 503 ein; die verbleibende 401 ist ein Fehler
	// beim Lesen oder an der Signatur, ein widerrufenes Token oder ein deaktiviertes Konto.
	"api/tuer/sitzung.go:PruefeSitzung":   "Token ungültig/widerrufen/Konto inaktiv = 401; DB-Ausfall vorher als 503 eingeordnet",
	"auth/anmelden.go:leseSitzung":        "Token ungültig/widerrufen/Konto inaktiv = 401; DB-Ausfall vorher als 503 eingeordnet",
	"auth/anmelden.go:SitzungVerlaengern": "Token ungültig/widerrufen/Konto inaktiv = 401; DB-Ausfall vorher als 503 eingeordnet",
	// Die drei Kataloge (DNB, Google, OpenLibrary) stehen nicht hier, sie ordnen ein: Netzausfall =
	// 502 (inventur.ErrKatalogdiensteNichtErreichbar), erreichbar ohne Treffer = 404.
}

// kollapsAntworten: die harmlosen Antworten als Status-Konstante und als Helfer der Hausform
// (apierrors.NotFound/Unauthorized/Forbidden hinter apierrors.MitFehlerantwort). Ohne die Helfer
// wäre die Ratsche für die häufigste Form blind.
var kollapsAntworten = map[string]bool{
	"StatusNotFound": true, "StatusUnauthorized": true, "StatusForbidden": true,
	"NotFound": true, "Unauthorized": true, "Forbidden": true,
}

func istFehlerName(name string) bool {
	n := strings.ToLower(name)
	return n == "err" || n == "fehler" || strings.HasSuffix(n, "err") || strings.HasSuffix(n, "fehler")
}

// enthaeltFehlerVergleich: irgendwo in der Bedingung steht `<err> != nil`.
func enthaeltFehlerVergleich(cond ast.Expr) bool {
	gefunden := false
	ast.Inspect(cond, func(n ast.Node) bool {
		if b, ok := n.(*ast.BinaryExpr); ok && b.Op == token.NEQ {
			if id, ok := b.X.(*ast.Ident); ok && istFehlerName(id.Name) {
				if nl, ok := b.Y.(*ast.Ident); ok && nl.Name == "nil" {
					gefunden = true
				}
			}
		}
		return !gefunden
	})
	return gefunden
}

// ordnetFehlerEin: die Bedingung selbst unterscheidet Fehlerarten.
func ordnetFehlerEin(cond ast.Expr) bool {
	gefunden := false
	ast.Inspect(cond, func(n ast.Node) bool {
		switch k := n.(type) {
		case *ast.SelectorExpr:
			if k.Sel.Name == "Is" || k.Sel.Name == "As" || strings.HasPrefix(k.Sel.Name, "Err") {
				gefunden = true
			}
		case *ast.Ident:
			if strings.HasPrefix(k.Name, "Err") || strings.HasPrefix(k.Name, "err") && len(k.Name) > 3 && k.Name != "error" {
				gefunden = true
			}
		}
		return !gefunden
	})
	return gefunden
}

// ordnetImRumpfEin: das erste Statement des Rumpfs ist ein if, das den Fehler einordnet
// (errors.Is / Sentinel / err.Error()-Vergleich) — die verschachtelte saubere Form.
func ordnetImRumpfEin(body *ast.BlockStmt) bool {
	if len(body.List) == 0 {
		return false
	}
	inner, ok := body.List[0].(*ast.IfStmt)
	if !ok {
		return false
	}
	if ordnetFehlerEin(inner.Cond) {
		return true
	}
	vergleich := false
	ast.Inspect(inner.Cond, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if sel, ok := c.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Error" {
				vergleich = true
			}
		}
		return !vergleich
	})
	return vergleich
}

func antwortetHarmlos(body *ast.BlockStmt) bool {
	gefunden := false
	ast.Inspect(body, func(n ast.Node) bool {
		if s, ok := n.(*ast.SelectorExpr); ok && kollapsAntworten[s.Sel.Name] {
			gefunden = true
		}
		return !gefunden
	})
	return gefunden
}

func TestKeinFehlerKollaps(t *testing.T) {
	var treffer []string
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if eintrag.IsDir() {
			name := eintrag.Name()
			if name == "frontend" || name == "node_modules" || name == ".git" || name == "e2e" {
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
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				ifs, ok := n.(*ast.IfStmt)
				if !ok {
					return true
				}
				if enthaeltFehlerVergleich(ifs.Cond) && !ordnetFehlerEin(ifs.Cond) && !ordnetImRumpfEin(ifs.Body) && antwortetHarmlos(ifs.Body) {
					treffer = append(treffer, fmt.Sprintf("%s:%s (%s)", pfad, fn.Name.Name, fset.Position(ifs.Pos())))
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(treffer)

	gesehen := map[string]bool{}
	var neu []string
	for _, tr := range treffer {
		schluessel := tr[:strings.Index(tr, " (")]
		gesehen[schluessel] = true
		if _, ok := kollapsBestand[schluessel]; !ok {
			neu = append(neu, tr)
		}
	}
	if len(neu) > 0 {
		t.Errorf("Fehler-Kollaps: err != nil landet in 404/401/403, ohne den Fehler einzuordnen:\n  %s\n"+
			"Fix: errors.Is(err, pgx.ErrNoRows) bzw. den Err*-Sentinel prüfen, den Rest als 500 weiterreichen.",
			strings.Join(neu, "\n  "))
	}
	for schluessel := range kollapsBestand {
		if !gesehen[schluessel] {
			t.Errorf("%s ist inzwischen sauber — bitte aus kollapsBestand streichen, damit die Ratsche greift.", schluessel)
		}
	}
}

// Gegenprobe am Detektor: ein Muster, das nichts findet, meldet ewig „alles gut".
func TestFehlerKollapsDetektorErkenntDieForm(t *testing.T) {
	quelle := `package p
import "net/http"
func kollaps(w http.ResponseWriter) { exists, err := f(); if err != nil || !exists { w.WriteHeader(http.StatusNotFound) } }
func sauber(w http.ResponseWriter) { _, err := f(); if errors.Is(err, pgx.ErrNoRows) { w.WriteHeader(http.StatusNotFound) } }
func sauber2(w http.ResponseWriter) { _, err := f(); if err != nil { w.WriteHeader(http.StatusInternalServerError) } }
func sauber3(w http.ResponseWriter) { _, err := f(); if err == ErrTitelNichtGefunden { w.WriteHeader(http.StatusNotFound) } }
func sauber4(w http.ResponseWriter) { _, err := f(); if err != nil { if errors.Is(err, pgx.ErrNoRows) { w.WriteHeader(http.StatusNotFound); return }; w.WriteHeader(500) } }
func kollaps2(w http.ResponseWriter) { _, err := f(); if err != nil { log.Print(err); w.WriteHeader(http.StatusNotFound) } }
func kollaps3() error { _, err := f(); if err != nil { return apierrors.NotFound("x", err) }; return nil }
func kollaps4() error { _, err := f(); if err != nil { return apierrors.Unauthorized("x", err) }; return nil }
func sauber5() error { _, err := f(); if errors.Is(err, pgx.ErrNoRows) { return apierrors.NotFound("x", err) }; return nil }
`
	fset := token.NewFileSet()
	datei, err := parser.ParseFile(fset, "probe.go", quelle, 0)
	if err != nil {
		t.Fatal(err)
	}
	// kollaps3/kollaps4/sauber5: die Hausform der Türen hinter apierrors.MitFehlerantwort. Ein
	// Detektor, der nur http.StatusNotFound & Co. kennt, ließe einen neuen `return
	// apierrors.NotFound("…", err)` im Zweig des Fehlers durch.
	erwartet := map[string]bool{"kollaps": true, "kollaps2": true, "kollaps3": true, "kollaps4": true,
		"sauber": false, "sauber2": false, "sauber3": false, "sauber4": false, "sauber5": false}
	for _, decl := range datei.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		gefunden := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if ifs, ok := n.(*ast.IfStmt); ok && enthaeltFehlerVergleich(ifs.Cond) && !ordnetFehlerEin(ifs.Cond) && !ordnetImRumpfEin(ifs.Body) && antwortetHarmlos(ifs.Body) {
				gefunden = true
			}
			return true
		})
		if gefunden != erwartet[fn.Name.Name] {
			t.Errorf("%s: erkannt=%v, erwartet %v", fn.Name.Name, gefunden, erwartet[fn.Name.Name])
		}
	}
}
