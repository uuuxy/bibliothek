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

// Gate gegen Fehler-Kollaps.
//
// Anlass (29.08.2026): GetBookByID nannte JEDEN Fehler „buch nicht gefunden" — ein
// Scan-Fehler (NULL in string, Schema-Drift) wäre als 404 verkleidet gewesen, und ein
// Jules-PR (#529) wollte genau das per Test festschreiben. Dieselbe Form stand in
// CreateKlassensatzReservierung: `if err != nil || !exists { 404 }`. Die Klasse:
// Eine Bedingung, die einen ECHTEN Fehler (err != nil) und einen fachlichen
// Nicht-Treffer in dieselbe harmlose Antwort (404/401/403) presst. Die Diagnose
// stirbt, der Betreiber sieht einen Bedienfehler statt eines kaputten Systems.
//
// Regel: Ein `if`, dessen Bedingung `<err> != nil` enthält und dessen Rumpf mit
// StatusNotFound/StatusUnauthorized/StatusForbidden antwortet, muss den Fehler in der
// Bedingung EINORDNEN (errors.Is / errors.As / ein Err*-Sentinel). Sonst ist es ein
// Kollaps. Konservativ: Nur die Bedingung wird gelesen, nicht der Rumpf — ein
// `errors.Is` im Rumpf gilt nicht, weil dann der Rest des Rumpfs trotzdem alles
// andere in dieselbe Antwort kippt.
//
// Die übliche saubere Form im Haus — `if err != nil { if errors.Is(err, ErrNoRows)
// { 404 }; 500 }` — ordnet im ERSTEN verschachtelten if ein; die gilt als sauber.
//
// Reparatur bei Rot: `errors.Is(err, pgx.ErrNoRows)` bzw. den Sentinel prüfen und den
// Rest als 500 (oder gewrappt) weiterreichen — wie repository_metadata.go seit 29.08.
var kollapsBestand = map[string]string{
	// „datei:funktion": Einordnung. Der Sweep vom 29.08.2026 fand 12 Stellen; 4 davon
	// waren echte DB-Kollapse und sind behoben (MeHandler, Foto, zwei Druck-Handler,
	// dazu GetBookByID und CreateKlassensatzReservierung). Die folgenden acht sind
	// semantisch vertretbar — der Fehler IST dort die negative Antwort:
	"api/csrf.go:CSRFMiddleware":                    "Validierungsfehler des Double-Submit = 403; kein DB-Zugriff",
	"api/permission_middleware.go:claimsAusRequest": "Token-Parsefehler = 401; kein DB-Zugriff",
	"auth/handlers.go:MeHandler":                    "Cookie/Token-Parsefehler = 401 (zwei Stellen); der DB-Kollaps dahinter ist behoben",
	"auth/handlers.go:RefreshTokenHandler":          "Token-Parsefehler = 401; kein DB-Zugriff",
	// Die drei externen Lookups (DNB, Google, OpenLibrary) sind seit dem 31.08.2026
	// eingeordnet: Netzausfall = 502 (inventur.ErrKatalogdiensteNichtErreichbar),
	// erreichbar ohne Treffer = 404 (Produktentscheidung; Sweep docs/sweeps.md).
}

var kollapsAntworten = map[string]bool{"StatusNotFound": true, "StatusUnauthorized": true, "StatusForbidden": true}

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
func sauber3(w http.ResponseWriter) { _, err := f(); if err == ErrBookNotFound { w.WriteHeader(http.StatusNotFound) } }
func sauber4(w http.ResponseWriter) { _, err := f(); if err != nil { if errors.Is(err, pgx.ErrNoRows) { w.WriteHeader(http.StatusNotFound); return }; w.WriteHeader(500) } }
func kollaps2(w http.ResponseWriter) { _, err := f(); if err != nil { log.Print(err); w.WriteHeader(http.StatusNotFound) } }
`
	fset := token.NewFileSet()
	datei, err := parser.ParseFile(fset, "probe.go", quelle, 0)
	if err != nil {
		t.Fatal(err)
	}
	erwartet := map[string]bool{"kollaps": true, "kollaps2": true, "sauber": false, "sauber2": false, "sauber3": false, "sauber4": false}
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
