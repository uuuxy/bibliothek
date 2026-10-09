package main

// Ratsche: Die Eingabe der Buchetiketten (pdf.BuchEtikett) füllt im Produktivcode nur die
// Funktion buchEtiketten in api/labels.go.
//
// Die Erzeuger in pdf/ kennen den Topf eines Exemplars nicht und drucken den
// Eigentumsvermerk, den sie bekommen. Welcher gilt, entscheidet buchEtiketten für alle
// Druckwege. Ein zweiter Füller wählte den Vermerk nach eigener Regel oder gar nicht, und
// dasselbe Buch trüge je nach Druckweg einen anderen Aufdruck.
//
// Regel: Außerhalb von pdf/ nennt keine Produktivdatei den Typ, außer in buchEtiketten. Wer
// nur das Ergebnis der Funktion weiterreicht, braucht den Namen nicht.
//
// Blindheit: ein Füller, der den Typ nie nennt (Reflection, eine Funktion in pdf/ selbst).
// Testdateien prüft die Ratsche mit Absicht nicht.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const (
	etikettEingabePaket = "bibliothek/pdf"
	etikettEingabeTyp   = "BuchEtikett"
)

// etikettEingabeNennungen liefert je Nennung des Typs „datei:funktion"; außerhalb einer
// Funktion steht „(Paketebene)". Ist quelle gesetzt, wird nur dieser Text gelesen.
func etikettEingabeNennungen(t *testing.T, pfad string, quelle any) []string {
	t.Helper()
	datei, err := parser.ParseFile(token.NewFileSet(), pfad, quelle, 0)
	if err != nil {
		t.Fatalf("%s parsen: %v", pfad, err)
	}
	// Unter welchem Namen die Datei das Paket einbindet: der letzte Teil des Pfads, ein
	// eigener Name oder der Punkt, mit dem der Typ ohne Vorsatz dasteht.
	name := ""
	for _, imp := range datei.Imports {
		if p, err := strconv.Unquote(imp.Path.Value); err != nil || p != etikettEingabePaket {
			continue
		}
		name = "pdf"
		if imp.Name != nil {
			name = imp.Name.Name
		}
	}
	if name == "" || name == "_" {
		return nil
	}

	var nennungen []string
	suche := func(knoten ast.Node, ort string) {
		ast.Inspect(knoten, func(n ast.Node) bool {
			switch k := n.(type) {
			case *ast.SelectorExpr:
				if x, ok := k.X.(*ast.Ident); ok && x.Name == name && k.Sel.Name == etikettEingabeTyp {
					nennungen = append(nennungen, filepath.ToSlash(pfad)+":"+ort)
				}
			case *ast.Ident:
				if name == "." && k.Name == etikettEingabeTyp {
					nennungen = append(nennungen, filepath.ToSlash(pfad)+":"+ort)
				}
			}
			return true
		})
	}
	for _, decl := range datei.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			suche(fn, fn.Name.Name)
			continue
		}
		suche(decl, "(Paketebene)")
	}
	return nennungen
}

func TestEtikettEingabe_NurBuchEtikettenFuelltSie(t *testing.T) {
	const erlaubt = "api/labels.go:buchEtiketten"
	var fremd []string
	inErlaubter := 0
	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "frontend", "pdf":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		for _, n := range etikettEingabeNennungen(t, pfad, nil) {
			if n == erlaubt {
				inErlaubter++
				continue
			}
			fremd = append(fremd, n)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Nicht-leer-Garantie: Ohne Nennung an der erlaubten Stelle sucht die Ratsche am
	// falschen Namen.
	if inErlaubter == 0 {
		t.Fatalf("%s nennt %s.%s nicht mehr: Die Ratsche misst nichts. Heißt die Funktion oder "+
			"der Typ jetzt anders, den Namen hier nachziehen.", erlaubt, etikettEingabePaket, etikettEingabeTyp)
	}
	if len(fremd) > 0 {
		slices.Sort(fremd)
		t.Errorf("pdf.%s wird außerhalb von buchEtiketten genannt: %v. Die Eingabe der Etiketten-"+
			"Erzeuger füllt nur buchEtiketten (api/labels.go); dort wird der Eigentumsvermerk je "+
			"Exemplar gewählt. Einen weiteren Druckweg über diese Funktion führen.",
			etikettEingabeTyp, slices.Compact(fremd))
	}
}

// Der Detektor erkennt jede Form, in der eine Datei den Typ nennen kann, und hält einen
// gleichnamigen Typ aus einem anderen Paket nicht für ihn.
func TestEtikettEingabe_DetektorErkenntJedeForm(t *testing.T) {
	const kopf = "package x\nimport \"bibliothek/pdf\"\n"
	mit := map[string]string{
		"Literal":                kopf + "func f() any { return pdf.BuchEtikett{Titel: \"a\"} }\n",
		"Literal in einer Liste": kopf + "func f() any { return []pdf.BuchEtikett{{Titel: \"a\"}} }\n",
		"Variable":               kopf + "func f() any { var e pdf.BuchEtikett; e.Titel = \"a\"; return e }\n",
		"make":                   kopf + "func f() any { return make([]pdf.BuchEtikett, 1) }\n",
		"Parameter":              kopf + "func f(e pdf.BuchEtikett) {}\n",
		"Paketebene":             kopf + "var muster = pdf.BuchEtikett{}\n",
		"eigener Name":           "package x\nimport blatt \"bibliothek/pdf\"\nfunc f() any { return blatt.BuchEtikett{} }\n",
		"mit Punkt":              "package x\nimport . \"bibliothek/pdf\"\nfunc f() any { return BuchEtikett{} }\n",
	}
	for form, quelle := range mit {
		if len(etikettEingabeNennungen(t, "probe.go", quelle)) == 0 {
			t.Errorf("%s: die Nennung wird nicht erkannt", form)
		}
	}
	ohne := map[string]string{
		"nur das Ergebnis":       kopf + "func f() any { return pdf.GenerateLabelsPDF }\n",
		"anderer Typ des Pakets": kopf + "func f() any { return pdf.SchuelerEtikett{} }\n",
		"anderes Paket":          "package x\nimport \"bibliothek/andere\"\nfunc f() any { return andere.BuchEtikett{} }\n",
		"anderes Paket neben pdf": "package x\nimport (\n\t\"bibliothek/andere\"\n\t\"bibliothek/pdf\"\n)\n" +
			"func f() any { return andere.BuchEtikett{} }\nvar _ = pdf.GenerateLabelsPDF\n",
		"eigener Typ":      "package x\ntype BuchEtikett struct{}\nfunc f() any { return BuchEtikett{} }\n",
		"nur im Kommentar": kopf + "// pdf.BuchEtikett füllt ein anderer\nvar _ = pdf.GenerateLabelsPDF\n",
		"nur als Text":     kopf + "var _ = pdf.GenerateLabelsPDF\nvar s = \"pdf.BuchEtikett\"\n",
	}
	for form, quelle := range ohne {
		if n := etikettEingabeNennungen(t, "probe.go", quelle); len(n) != 0 {
			t.Errorf("%s: gilt als Nennung, obwohl der Typ nicht gemeint ist: %v", form, n)
		}
	}
	if n := etikettEingabeNennungen(t, "probe.go", kopf+"func buchEtiketten() any { return pdf.BuchEtikett{} }\n"); len(n) != 1 || n[0] != "probe.go:buchEtiketten" {
		t.Errorf("Ort der Nennung = %v, erwartet [probe.go:buchEtiketten]", n)
	}
}
