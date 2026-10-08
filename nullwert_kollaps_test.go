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

// Gate: Ein gescheitertes Lesen gilt nicht als „nichts gespeichert".
//
// Anlass (08.10.2026, api/ausweis_layout.go): `case errors.Is(err, pgx.ErrNoRows) ||
// strings.TrimSpace(wert) == ""` stand vor `case err != nil`. Nach einem gescheiterten Scan
// ist der Wert leer; der erste Zweig traf deshalb jeden Fehler, und die Tür beantwortete
// einen Lesefehler mit dem leeren Design. Der Ausweis-Designer speichert nach dieser
// Antwort seine Vorgabewerte über das Design der Schule.
//
// Regel: Steht „keine Zeile" (`errors.Is(<err>, …ErrNoRows)` oder `<err> == …ErrNoRows`) in
// einem Oder, nennt jeder andere Teil des Oders denselben Fehler. Die Hausform bindet den
// Wert an das gelungene Lesen: `errors.Is(err, pgx.ErrNoRows) || (err == nil && wert == "")`.
//
// Reparatur bei Rot: den Fehler in einem eigenen Zweig vor dem Wert prüfen oder den Wert
// mit `err == nil &&` binden.
//
// Sieht nicht: einen Nullwert, der erst in einer späteren Anweisung geprüft wird, und
// „keine Zeile" hinter einem eigenen Sentinel (ErrBookNotFound).

func ohneKlammern(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

// oderTeile zerlegt `a || b || c` in seine Teile.
func oderTeile(e ast.Expr) []ast.Expr {
	e = ohneKlammern(e)
	if b, ok := e.(*ast.BinaryExpr); ok && b.Op == token.LOR {
		return append(oderTeile(b.X), oderTeile(b.Y)...)
	}
	return []ast.Expr{e}
}

func istErrNoRows(e ast.Expr) bool {
	sel, ok := ohneKlammern(e).(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "ErrNoRows"
}

// keineZeileFehler liefert den Namen der Fehlervariablen, wenn der Ausdruck „keine Zeile"
// prüft: errors.Is(<name>, x.ErrNoRows) oder <name> == x.ErrNoRows.
func keineZeileFehler(e ast.Expr) (string, bool) {
	switch k := ohneKlammern(e).(type) {
	case *ast.CallExpr:
		sel, ok := k.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Is" || len(k.Args) != 2 || !istErrNoRows(k.Args[1]) {
			return "", false
		}
		if name, ok := k.Args[0].(*ast.Ident); ok {
			return name.Name, true
		}
	case *ast.BinaryExpr:
		if k.Op != token.EQL {
			return "", false
		}
		if name, ok := k.X.(*ast.Ident); ok && istErrNoRows(k.Y) {
			return name.Name, true
		}
		if name, ok := k.Y.(*ast.Ident); ok && istErrNoRows(k.X) {
			return name.Name, true
		}
	}
	return "", false
}

func nenntBezeichner(e ast.Expr, name string) bool {
	gefunden := false
	ast.Inspect(e, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == name {
			gefunden = true
		}
		return !gefunden
	})
	return gefunden
}

// nullwertKollaps prüft ein Oder. hatKeineZeile: Es enthält eine Prüfung auf „keine Zeile".
// kollaps: Daneben steht ein Teil, der den Fehler nicht nennt.
func nullwertKollaps(e ast.Expr) (hatKeineZeile, kollaps bool) {
	b, ok := e.(*ast.BinaryExpr)
	if !ok || b.Op != token.LOR {
		return false, false
	}
	teile := oderTeile(b)
	for i, teil := range teile {
		name, ok := keineZeileFehler(teil)
		if !ok {
			continue
		}
		hatKeineZeile = true
		for j, anderer := range teile {
			if j != i && !nenntBezeichner(anderer, name) {
				kollaps = true
			}
		}
	}
	return hatKeineZeile, kollaps
}

// nullwertKollapsBestand: „datei:funktion" → warum es dort richtig ist. Leer: Die eine
// Fundstelle ist behoben.
var nullwertKollapsBestand = map[string]string{}

func TestKeinLesefehlerAlsNichtGespeichert(t *testing.T) {
	var treffer []string
	geprueft := 0
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
				ausdruck, ok := n.(ast.Expr)
				if !ok {
					return true
				}
				hat, kollaps := nullwertKollaps(ausdruck)
				if !hat {
					return true
				}
				geprueft++
				if kollaps {
					treffer = append(treffer, fmt.Sprintf("%s:%s (%s)", pfad, fn.Name.Name, fset.Position(ausdruck.Pos())))
				}
				// Die Teile dieses Oders sind geprüft; ein inneres Oder zählte sonst doppelt.
				return false
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Der Bestand kennt die gebundene Hausform an mehreren Stellen (repository/leser_konto.go,
	// repository/auflagen_klasse.go, auth/handlers.go). Findet der Durchlauf keine davon,
	// liest er nicht, was er lesen soll.
	if geprueft < 3 {
		t.Fatalf("nur %d Oder mit „keine Zeile“ gelesen, erwartet mindestens 3 — der Detektor läuft leer", geprueft)
	}
	sort.Strings(treffer)

	gesehen := map[string]bool{}
	var neu []string
	for _, tr := range treffer {
		schluessel := tr[:strings.Index(tr, " (")]
		gesehen[schluessel] = true
		if _, ok := nullwertKollapsBestand[schluessel]; !ok {
			neu = append(neu, tr)
		}
	}
	if len(neu) > 0 {
		t.Errorf("Lesefehler als „nichts gespeichert“: Neben „keine Zeile“ steht im selben Oder ein Wert, "+
			"der nach einem gescheiterten Lesen leer ist:\n  %s\n"+
			"Fix: den Fehler in einem eigenen Zweig vorher prüfen oder den Wert mit `err == nil &&` binden.",
			strings.Join(neu, "\n  "))
	}
	for schluessel := range nullwertKollapsBestand {
		if !gesehen[schluessel] {
			t.Errorf("%s ist inzwischen sauber — bitte aus nullwertKollapsBestand streichen, damit die Ratsche greift.", schluessel)
		}
	}
}

// Gegenprobe am Detektor über die Formen der Klasse: gedreht, umklammert, im switch, als
// direkter Vergleich, mit anderem Variablennamen, in einer Kette.
func TestNullwertKollapsDetektorErkenntDieFormen(t *testing.T) {
	quelle := `package p
func kollapsFund() { switch { case errors.Is(err, pgx.ErrNoRows) || strings.TrimSpace(wert) == "": x() } }
func kollapsGedreht() { if wert == "" || errors.Is(err, pgx.ErrNoRows) { x() } }
func kollapsVergleich() { if err == sql.ErrNoRows || !gefunden { x() } }
func kollapsVergleichGedreht() { if pgx.ErrNoRows == err || id == "" { x() } }
func kollapsName() { leer := errors.Is(fehler, pgx.ErrNoRows) || anzahl == 0; _ = leer }
func kollapsKette() { if a == nil || errors.Is(err, pgx.ErrNoRows) || (err == nil && b == "") { x() } }
func kollapsKlammer() { if (errors.Is(err, pgx.ErrNoRows)) || (wert == "") { x() } }
func kollapsFremderFehler() { if errors.Is(err, pgx.ErrNoRows) || (err2 == nil && wert == "") { x() } }
func sauberGebunden() { if errors.Is(err, pgx.ErrNoRows) || (err == nil && wert == "") { x() } }
func sauberZweiTreiber() { if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) { x() } }
func sauberAllein() { if errors.Is(err, pgx.ErrNoRows) { x() } }
func sauberUnd() { if errors.Is(err, pgx.ErrNoRows) && wert == "" { x() } }
func sauberOhneKeineZeile() { if err != nil || wert == "" { x() } }
`
	fset := token.NewFileSet()
	datei, err := parser.ParseFile(fset, "probe.go", quelle, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range datei.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		gefunden := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if ausdruck, ok := n.(ast.Expr); ok {
				if _, kollaps := nullwertKollaps(ausdruck); kollaps {
					gefunden = true
				}
			}
			return true
		})
		if erwartet := strings.HasPrefix(fn.Name.Name, "kollaps"); gefunden != erwartet {
			t.Errorf("%s: erkannt=%v, erwartet %v", fn.Name.Name, gefunden, erwartet)
		}
	}
}
