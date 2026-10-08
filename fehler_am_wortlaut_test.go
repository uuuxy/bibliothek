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

// Gate: Ein Fehler wird an seinem Namen erkannt, nicht an seinem Wortlaut.
//
// Anlass (docs/OFFEN.md 5.10, gezählt am 08.10.2026): Sieben Stellen entschieden den Status
// ihrer Antwort mit strings.Contains(err.Error(), "…"). Formuliert jemand die Meldung an der
// Quelle um, wird dort aus einer Auskunft ein Serverfehler oder umgekehrt, und kein Test
// merkt es. An einer der sieben war es schon geschehen: Den gesuchten Wortlaut erzeugte keine
// Stelle mehr, der Zweig war tot.
//
// Regel: Kein Vergleich und keine Teilsuche am Text eines Fehlers — weder direkt an
// <x>.Error() noch an einer Variablen, die in derselben Funktion diesen Text bekommt.
//
// Reparatur bei Rot: ein benannter Fehler mit errors.Is, ein eigener Fehlertyp mit errors.As
// oder bei Datenbankfehlern der SQLSTATE und der Name der Bedingung (pgconn.PgError, Helfer
// wie repository.IstExemplarBarcodeVergeben).
//
// Sieht nicht: einen Text, der als Parameter in eine andere Funktion wandert und dort
// durchsucht wird (apierrors.istDatenbankFehler), und einen Text, der über ein Feld oder über
// zwei Zuweisungen weitergereicht wird.

// wortlautPraedikate sind die Funktionen aus strings, deren Ergebnis eine Entscheidung ist.
var wortlautPraedikate = map[string]bool{
	"Contains": true, "ContainsAny": true, "ContainsRune": true, "HasPrefix": true, "HasSuffix": true,
	"EqualFold": true, "Index": true, "LastIndex": true, "Count": true, "Compare": true,
}

// wortlautSucher sind die Methoden eines regulären Ausdrucks, die einen Text durchsuchen.
var wortlautSucher = map[string]bool{
	"MatchString": true, "FindString": true, "FindStringSubmatch": true, "FindStringIndex": true,
}

// istFehlertextAufruf erkennt <x>.Error() ohne Argumente.
func istFehlertextAufruf(n ast.Node) bool {
	aufruf, ok := n.(*ast.CallExpr)
	if !ok || len(aufruf.Args) != 0 {
		return false
	}
	sel, ok := aufruf.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Error"
}

// nenntFehlertext sagt, ob der Ausdruck den Text eines Fehlers trägt: einen Aufruf von
// Error() oder eine Variable, der die Funktion einen solchen Text zugewiesen hat.
func nenntFehlertext(e ast.Node, texte map[string]bool) bool {
	gefunden := false
	ast.Inspect(e, func(n ast.Node) bool {
		if gefunden || n == nil {
			return false
		}
		if istFehlertextAufruf(n) {
			gefunden = true
		}
		if id, ok := n.(*ast.Ident); ok && texte[id.Name] {
			gefunden = true
		}
		return !gefunden
	})
	return gefunden
}

// fehlertextVariablen sammelt die Namen, denen der Rumpf den Text eines Fehlers zuweist.
func fehlertextVariablen(rumpf *ast.BlockStmt) map[string]bool {
	texte := map[string]bool{}
	merke := func(ziel ast.Expr, wert ast.Expr) {
		if id, ok := ziel.(*ast.Ident); ok && id.Name != "_" && nenntFehlertext(wert, nil) {
			texte[id.Name] = true
		}
	}
	ast.Inspect(rumpf, func(n ast.Node) bool {
		switch z := n.(type) {
		case *ast.AssignStmt:
			if len(z.Lhs) == len(z.Rhs) {
				for i := range z.Lhs {
					merke(z.Lhs[i], z.Rhs[i])
				}
			}
		case *ast.ValueSpec:
			if len(z.Names) == len(z.Values) {
				for i := range z.Names {
					merke(z.Names[i], z.Values[i])
				}
			}
		}
		return true
	})
	return texte
}

func istNichtLeeresTextliteral(e ast.Expr) bool {
	lit, ok := ohneKlammern(e).(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && len(lit.Value) > 2
}

// wortlautEntscheidung sagt, ob der Knoten am Text eines Fehlers entscheidet.
func wortlautEntscheidung(n ast.Node, texte map[string]bool) bool {
	switch k := n.(type) {
	case *ast.CallExpr:
		sel, ok := k.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		paket, istPaket := sel.X.(*ast.Ident)
		praedikat := istPaket && paket.Name == "strings" && wortlautPraedikate[sel.Sel.Name]
		if !praedikat && !wortlautSucher[sel.Sel.Name] {
			return false
		}
		for _, arg := range k.Args {
			if nenntFehlertext(arg, texte) {
				return true
			}
		}
	case *ast.BinaryExpr:
		if k.Op != token.EQL && k.Op != token.NEQ {
			return false
		}
		return (nenntFehlertext(k.X, texte) && istNichtLeeresTextliteral(k.Y)) ||
			(nenntFehlertext(k.Y, texte) && istNichtLeeresTextliteral(k.X))
	case *ast.SwitchStmt:
		return k.Tag != nil && nenntFehlertext(k.Tag, texte)
	}
	return false
}

// wortlautFunde nennt die Stellen einer Funktion, die am Text eines Fehlers entscheiden.
func wortlautFunde(fn *ast.FuncDecl) []token.Pos {
	texte := fehlertextVariablen(fn.Body)
	var funde []token.Pos
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if n != nil && wortlautEntscheidung(n, texte) {
			funde = append(funde, n.Pos())
		}
		return true
	})
	return funde
}

// fehlerAmWortlautBestand: „datei:funktion" → warum es dort richtig ist.
var fehlerAmWortlautBestand = map[string]string{
	"apierrors/apierrors.go:sanitizeInternalError": "wählt den neutralen Satz einer Antwort, deren Status feststeht; " +
		"der Fehler kann hier jede Herkunft haben, auch ohne pgconn.PgError in der Kette",
}

func TestKeinFehlerAmWortlaut(t *testing.T) {
	var treffer []string
	funktionenMitFehlertext := 0
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
			if nenntFehlertext(fn.Body, nil) {
				funktionenMitFehlertext++
			}
			for _, pos := range wortlautFunde(fn) {
				treffer = append(treffer, fmt.Sprintf("%s:%s (%s)", filepath.ToSlash(pfad), fn.Name.Name, fset.Position(pos)))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Der Bestand reicht den Text eines Fehlers an vielen Stellen an Protokoll und Antwort
	// weiter. Findet der Durchlauf kaum eine davon, liest er nicht, was er lesen soll.
	if funktionenMitFehlertext < 50 {
		t.Fatalf("nur %d Funktionen mit einem Aufruf von Error() gelesen, erwartet mindestens 50 — der Detektor läuft leer", funktionenMitFehlertext)
	}
	sort.Strings(treffer)

	gesehen := map[string]bool{}
	var neu []string
	for _, tr := range treffer {
		schluessel := tr[:strings.Index(tr, " (")]
		gesehen[schluessel] = true
		if _, ok := fehlerAmWortlautBestand[schluessel]; !ok {
			neu = append(neu, tr)
		}
	}
	if len(neu) > 0 {
		t.Errorf("Fehler am Wortlaut erkannt: Hier entscheidet der Text einer Fehlermeldung. Eine Umformulierung "+
			"an der Quelle ändert die Antwort, ohne dass ein Test es merkt:\n  %s\n"+
			"Fix: benannter Fehler mit errors.Is, eigener Fehlertyp mit errors.As, bei Datenbankfehlern der "+
			"SQLSTATE mit dem Namen der Bedingung (pgconn.PgError).",
			strings.Join(neu, "\n  "))
	}
	for schluessel := range fehlerAmWortlautBestand {
		if !gesehen[schluessel] {
			t.Errorf("%s ist inzwischen sauber — bitte aus fehlerAmWortlautBestand streichen, damit die Ratsche greift.", schluessel)
		}
	}
}

// Gegenprobe am Detektor über die Formen der Klasse: Teilsuche, Anfang, ohne Rücksicht auf
// Groß und Klein, verneint, umklammert, über eine Variable, als Vergleich, im switch, über
// einen regulären Ausdruck.
func TestFehlerAmWortlautDetektorErkenntDieFormen(t *testing.T) {
	quelle := `package p
func wortlautTeilsuche() { if strings.Contains(err.Error(), "no rows") { x() } }
func wortlautAnfang() { if strings.HasPrefix(fehler.Error(), "fehler bei") { x() } }
func wortlautKlein() { if strings.Contains(strings.ToLower(err.Error()), "duplicate key") { x() } }
func wortlautGleichgueltig() { if strings.EqualFold(err.Error(), "nicht gefunden") { x() } }
func wortlautVerneint() { if !(strings.HasSuffix(err.Error(), "abgebrochen")) { x() } }
func wortlautOder() { if strings.Contains(err.Error(), "unique constraint") || istAnderes(err) { x() } }
func wortlautVariable() { text := err.Error(); if strings.Contains(text, "23505") { x() } }
func wortlautVariableKlein() { var text = strings.ToLower(err.Error()); if strings.Index(text, "sql") >= 0 { x() } }
func wortlautSpaetereZuweisung() { msg := ""; msg = err.Error(); switch { case strings.Contains(msg, "violates"): x() } }
func wortlautVergleich() { if err.Error() == "EOF" { x() } }
func wortlautVergleichGedreht() { if "EOF" != err.Error() { x() } }
func wortlautSwitch() { switch err.Error() { case "EOF": x() } }
func wortlautMuster() { if muster.MatchString(err.Error()) { x() } }
func wortlautEinGeschachtelterFehler() { if strings.Contains(antwort.Fehler.Error(), "zugeordnet") { x() } }
func sauberBenannt() { if errors.Is(err, pgx.ErrNoRows) { x() } }
func sauberTyp() { var f *Eigener; if errors.As(err, &f) { x() } }
func sauberWeitergereicht() { writeError(w, 400, err.Error()) }
func sauberProtokoll() { log.Printf("gescheitert: %v", strings.TrimSpace(err.Error())) }
func sauberLeer() { msg := err.Error(); if msg == "" { x() } }
func sauberAndererText() { if strings.Contains(name, "no rows") { x() } }
func sauberFehlerSelbst() { if err == io.EOF { x() } }
func sauberZusammengesetzt() { return fmt.Errorf("lesen: %s", err.Error()) }
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
		gefunden := len(wortlautFunde(fn)) > 0
		if erwartet := strings.HasPrefix(fn.Name.Name, "wortlaut"); gefunden != erwartet {
			t.Errorf("%s: erkannt=%v, erwartet %v", fn.Name.Name, gefunden, erwartet)
		}
	}
}
