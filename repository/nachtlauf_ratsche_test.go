package repository

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Kein Nachtlauf löscht oder anonymisiert mit einer eigenen Bedingung: Die Bedingung kommt aus
// loeschfristen.go, damit der Wächter des Rückstands dieselbe Frage zählt. Formulierte ein Lauf
// sie selbst, meldete der Wächter weiter „kein Rückstand", während der Lauf etwas anderes
// löscht oder gar nichts.
//
// Blindheit: Geprüft sind die Funktionen dieses Pakets, die jobs/ unmittelbar ruft; eine
// Methode eines Repositorys und eine Funktion, die erst über eine zweite gerufen wird, nicht.
// Ob Bedingung und Tabelle zusammenpassen, misst jobs/loeschrueckstand_paarung_pg_test.go an
// der Datenbank.

// ohneBedingungAusLoeschfristen nennt die Funktionen der Nachtläufe, die ändern oder löschen,
// ohne eine Bedingung aus loeschfristen.go einzusetzen, und warum.
var ohneBedingungAusLoeschfristen = map[string]string{
	"AnonymisiereBearbeiterAlterAusleihen": "feste Frist von 14 Tagen in der Anweisung; der Wächter des Rückstands zählt sie nicht",
	"LoescheFotosAnonymisierterSchueler":   "folgt der Anonymisierung und hat keine eigene Frist",
	"LoescheAlteIdempotenzSchluessel":      "Schlüssel der Theke nach 24 Stunden; keine Frist aus den Einstellungen",
}

var aenderndeAnweisung = regexp.MustCompile(`(?i)\b(DELETE\s+FROM|UPDATE)\s+[a-z_]+`)

// produktivDateien liest die Go-Dateien eines Ordners ohne Tests.
func produktivDateien(t *testing.T, ordner string) map[string]*ast.File {
	t.Helper()
	eintraege, err := os.ReadDir(ordner)
	if err != nil {
		t.Fatalf("%s nicht lesbar: %v", ordner, err)
	}
	dateien := map[string]*ast.File{}
	for _, e := range eintraege {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		datei, err := parser.ParseFile(token.NewFileSet(), filepath.Join(ordner, name), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s nicht lesbar: %v", name, err)
		}
		dateien[name] = datei
	}
	return dateien
}

// vonJobsGerufen nennt die Funktionen dieses Pakets, die eine Datei in jobs/ unmittelbar ruft.
func vonJobsGerufen(t *testing.T) map[string]bool {
	t.Helper()
	gerufen := map[string]bool{}
	for _, datei := range produktivDateien(t, filepath.Join("..", "jobs")) {
		ast.Inspect(datei, func(knoten ast.Node) bool {
			ruf, ok := knoten.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := ruf.Fun.(*ast.SelectorExpr); ok {
				if paket, ok := sel.X.(*ast.Ident); ok && paket.Name == "repository" {
					gerufen[sel.Sel.Name] = true
				}
			}
			return true
		})
	}
	return gerufen
}

// aendertUndNenntBedingung sagt für einen Funktionsrumpf, ob er eine ändernde Anweisung trägt
// und ob er eine Funktion ruft, deren Name mit Predikat beginnt.
func aendertUndNenntBedingung(rumpf ast.Node) (aendert, nenntBedingung bool) {
	ast.Inspect(rumpf, func(knoten ast.Node) bool {
		switch k := knoten.(type) {
		case *ast.BasicLit:
			if k.Kind == token.STRING {
				if text, err := strconv.Unquote(k.Value); err == nil && aenderndeAnweisung.MatchString(text) {
					aendert = true
				}
			}
		case *ast.CallExpr:
			if name, ok := k.Fun.(*ast.Ident); ok && strings.HasPrefix(name.Name, "Predikat") {
				nenntBedingung = true
			}
		}
		return true
	})
	return aendert, nenntBedingung
}

func TestNachtlaeufe_AenderndeAnweisungSetztBedingungAusLoeschfristenEin(t *testing.T) {
	gerufen := vonJobsGerufen(t)
	mitBedingung := 0
	ohne := map[string]bool{}
	for name, datei := range produktivDateien(t, ".") {
		for _, decl := range datei.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil || !gerufen[fn.Name.Name] {
				continue
			}
			aendert, nenntBedingung := aendertUndNenntBedingung(fn.Body)
			switch {
			case !aendert:
			case nenntBedingung:
				mitBedingung++
			default:
				ohne[fn.Name.Name] = true
				if _, ok := ohneBedingungAusLoeschfristen[fn.Name.Name]; !ok {
					t.Errorf("%s (repository/%s) ändert oder löscht für einen Nachtlauf mit einer eigenen "+
						"Bedingung. Die Bedingung gehört als Predikat-Funktion nach loeschfristen.go, damit "+
						"Lauf und Wächter des Rückstands denselben Satz einsetzen.", fn.Name.Name, name)
				}
			}
		}
	}
	for name, grund := range ohneBedingungAusLoeschfristen {
		if !ohne[name] {
			t.Errorf("%s steht als Ausnahme (%q), trifft aber nicht mehr zu — Eintrag entfernen.", name, grund)
		}
	}
	// Nicht-leer-Garantie: Ohne Fundstellen prüfte der Test ins Leere.
	if mitBedingung < 8 {
		t.Fatalf("nur %d ändernde Anweisungen der Nachtläufe mit Bedingung gefunden — der Detektor misst offenbar nichts mehr", mitBedingung)
	}
}

// Eine Löschbedingung entsteht nur in loeschfristen.go. Wer eine der Anweisungen rufen will,
// bekommt seine Bedingung von dort und kann keine eigene zusammensetzen.
func TestLoeschbedingung_EntstehtNurInLoeschfristen(t *testing.T) {
	const heimat = "repository/loeschfristen.go"
	gefunden := map[string]int{}
	err := filepath.WalkDir("..", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "node_modules" || name == "frontend" || name == "testdata" ||
				(strings.HasPrefix(name, ".") && pfad != "..") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		datei, err := parser.ParseFile(token.NewFileSet(), pfad, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel("..", pfad)
		if err != nil {
			return err
		}
		ast.Inspect(datei, func(knoten ast.Node) bool {
			if lit, ok := knoten.(*ast.CompositeLit); ok && istLoeschbedingung(lit.Type) {
				gefunden[filepath.ToSlash(rel)]++
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if gefunden[heimat] < 10 {
		t.Fatalf("nur %d Löschbedingungen in %s gefunden — der Detektor misst offenbar nichts mehr", gefunden[heimat], heimat)
	}
	var fremde []string
	for pfad := range gefunden {
		if pfad != heimat {
			fremde = append(fremde, pfad)
		}
	}
	sort.Strings(fremde)
	for _, pfad := range fremde {
		t.Errorf("%s setzt eine Loeschbedingung selbst zusammen. Die Bedingung gehört als Predikat-Funktion "+
			"nach %s: Dort setzt sie auch der Wächter des Rückstands ein.", pfad, heimat)
	}
}

// istLoeschbedingung erkennt den Typ im eigenen Paket und mit Paketnamen davor.
func istLoeschbedingung(typ ast.Expr) bool {
	switch t := typ.(type) {
	case *ast.Ident:
		return t.Name == "Loeschbedingung"
	case *ast.SelectorExpr:
		return t.Sel.Name == "Loeschbedingung"
	}
	return false
}
