package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Ratsche: Jeder cp1252-Übersetzer eines PDF-Dokuments läuft durch pdfzeichen.Uebersetzer.
//
// Anlass (21.09.2026, OFFEN.md 5.5): Die Ersetzung für Buchstaben außerhalb von cp1252
// (ş, ł, ğ …) gab es seit dem Schüler-Etikett — aber nur dort und im Bescheid. Die
// Buchetiketten, die Mahnbriefe, die Bestellungen und alle anderen Renderer holten sich
// ihren Übersetzer direkt von gofpdf und druckten weiter Punkte. Zwei Wege zum selben
// Papier, und der zweite kannte die Regel des ersten nicht.
//
// Regel: Eine Zeile, die UnicodeTranslatorFromDescriptor( aufruft, trägt auf derselben
// Zeile pdfzeichen.Uebersetzer(. Wer einen neuen Renderer baut, bekommt die Ersetzung
// damit von allein — oder diesen Test rot.
func TestPdfUebersetzer_NurUeberPdfzeichen(t *testing.T) {
	var verstoesse []string
	gefunden := 0
	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "frontend" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad) // #nosec G304 -- Repo-Dateien
		if err != nil {
			return err
		}
		for nr, zeile := range strings.Split(string(inhalt), "\n") {
			if strings.HasPrefix(strings.TrimSpace(zeile), "//") {
				continue
			}
			if !strings.Contains(zeile, "UnicodeTranslatorFromDescriptor(") {
				continue
			}
			gefunden++
			if !strings.Contains(zeile, "pdfzeichen.Uebersetzer(") {
				verstoesse = append(verstoesse, pfad+":"+strconv.Itoa(nr+1))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Nicht-leer-Garantie: Findet der Scanner keine Aufrufe mehr, prüft er nichts.
	if gefunden < 10 {
		t.Fatalf("nur %d Übersetzer-Aufrufe gefunden — der Scanner greift nicht mehr (erwartet ≥ 17)", gefunden)
	}
	if len(verstoesse) > 0 {
		t.Fatalf("cp1252-Übersetzer ohne pdfzeichen.Uebersetzer in %v — dort werden ş, ł, ğ zu Punkten. "+
			"Form: pdfzeichen.Uebersetzer(p.UnicodeTranslatorFromDescriptor(\"\"))", verstoesse)
	}
}

// Ratsche: Jeder Text, den ein Erzeuger mit gofpdf druckt, geht durch den Übersetzer.
//
// gofpdf druckt in cp1252. Ein Text mit Umlaut, der roh in eine Zelle geht, steht verstümmelt
// auf dem Blatt („KÃ¶ln"); an einer Schule in einem Ort ohne Umlaut fällt das nie auf.
//
// Regel: Das Text-Argument eines Druckaufrufs ist ein Aufruf des Übersetzers (tr, druck) oder
// sichtbar ohne Buchstaben außerhalb von ASCII: ein Literal, eine Zahl (fmt.Sprintf mit
// Zahlenplatzhaltern, strconv.Itoa), ein formatierter Zeitpunkt (Format) oder eine Verkettung
// daraus. Ein Wert aus einer Variable gilt nicht als übersetzt: Der Aufruf nennt den Übersetzer
// selbst.
//
// Blindheit: ein Übersetzer unter anderem Namen; eine Methode Format, die keinen Zeitpunkt
// formatiert; Erzeuger mit maroto.

// druckMethoden nennt je Druckmethode von gofpdf die Zahl ihrer Argumente und die Stelle des
// Texts. Die Zahl hält andere Methoden gleichen Namens heraus (ein Write mit einem Argument).
var druckMethoden = map[string][2]int{
	"Cell":       {3, 2},
	"CellFormat": {9, 2},
	"MultiCell":  {6, 2},
	"Text":       {3, 2},
	"Write":      {2, 1},
}

var zahlenPlatzhalter = regexp.MustCompile(`%%|%[-+ 0-9.]*[dfFeEgGxXbo]`)

func nurASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

// nurZahlenFormat sagt, ob ein Formattext aus ASCII besteht und nur Zahlen einsetzt.
func nurZahlenFormat(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING || !nurASCII(lit.Value) {
		return false
	}
	return !strings.Contains(zahlenPlatzhalter.ReplaceAllString(lit.Value, ""), "%")
}

// druckTextGeprueft sagt, ob ein Ausdruck durch den Übersetzer geht oder sichtbar keinen
// Buchstaben außerhalb von ASCII trägt.
func druckTextGeprueft(e ast.Expr) bool {
	switch a := e.(type) {
	case *ast.BasicLit:
		return a.Kind == token.STRING && nurASCII(a.Value)
	case *ast.ParenExpr:
		return druckTextGeprueft(a.X)
	case *ast.BinaryExpr:
		return a.Op == token.ADD && druckTextGeprueft(a.X) && druckTextGeprueft(a.Y)
	case *ast.CallExpr:
		switch f := a.Fun.(type) {
		case *ast.Ident:
			return f.Name == "tr" || f.Name == "druck"
		case *ast.SelectorExpr:
			paket := ""
			if name, ok := f.X.(*ast.Ident); ok {
				paket = name.Name
			}
			switch {
			case paket == "strconv" && f.Sel.Name == "Itoa":
				return true
			case paket == "fmt" && f.Sel.Name == "Sprintf":
				return len(a.Args) > 0 && nurZahlenFormat(a.Args[0])
			case f.Sel.Name == "Format":
				return true
			}
		}
	}
	return false
}

// druckaufrufeOhneUebersetzer liest eine Go-Datei und nennt die Druckaufrufe, deren Text nicht
// geprüft ist, dazu die Zahl aller gelesenen Druckaufrufe.
func druckaufrufeOhneUebersetzer(t *testing.T, pfad string, quelle any) (verstoesse []string, gelesen int) {
	t.Helper()
	fset := token.NewFileSet()
	datei, err := parser.ParseFile(fset, pfad, quelle, 0)
	if err != nil {
		t.Fatalf("%s nicht lesbar: %v", pfad, err)
	}
	ast.Inspect(datei, func(n ast.Node) bool {
		ruf, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := ruf.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		form, bekannt := druckMethoden[sel.Sel.Name]
		if !bekannt || len(ruf.Args) != form[0] {
			return true
		}
		gelesen++
		if text := ruf.Args[form[1]]; !druckTextGeprueft(text) {
			verstoesse = append(verstoesse, pfad+":"+strconv.Itoa(fset.Position(ruf.Pos()).Line)+
				": "+sel.Sel.Name+"(…, "+types.ExprString(text)+")")
		}
		return true
	})
	return verstoesse, gelesen
}

func TestPdfDruck_JederTextGehtDurchDenUebersetzer(t *testing.T) {
	var verstoesse []string
	gelesen := 0
	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "frontend" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad) // #nosec G304 -- Repo-Dateien
		if err != nil {
			return err
		}
		if !strings.Contains(string(inhalt), `"github.com/jung-kurt/gofpdf"`) {
			return nil
		}
		gefunden, anzahl := druckaufrufeOhneUebersetzer(t, pfad, inhalt)
		verstoesse = append(verstoesse, gefunden...)
		gelesen += anzahl
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Nicht-leer-Garantie: Liest der Scanner keine Druckaufrufe mehr, prüft er nichts.
	if gelesen < 200 {
		t.Fatalf("nur %d Druckaufrufe gelesen — der Scanner greift nicht mehr (erwartet über 250)", gelesen)
	}
	if len(verstoesse) > 0 {
		t.Fatalf("Text ohne Übersetzer gedruckt: Umlaute stünden verstümmelt auf dem Blatt.\n  %s\n"+
			"Form: p.Cell(w, h, tr(text)); eine Zahl als strconv.Itoa oder fmt.Sprintf mit %%d.",
			strings.Join(verstoesse, "\n  "))
	}
}

// Der Detektor kennt jede Form: was er durchlässt und was er meldet.
func TestPdfDruck_DetektorKenntJedeForm(t *testing.T) {
	quelle := func(text string) string {
		return "package x\nfunc f() { p.CellFormat(10, 5, " + text + ", \"1\", 0, \"L\", false, 0, \"\") }\n"
	}
	geprueft := []string{
		`tr(name)`, `druck(name)`, `tr(fmt.Sprintf("%s, %s", a, b))`,
		`"ISBN"`, `""`,
		`fmt.Sprintf("%d Tage", n)`, `fmt.Sprintf("- %d -", p.PageNo())`, `fmt.Sprintf("%.2f %%", x)`,
		`strconv.Itoa(n)`,
		`tag.Format(dateFormatDE)`, `tag.In(zone).Format("02.01.2006")`,
		`"Datum: " + schulzeit.Jetzt().Format(dateFormatDE)`, `("Nr. " + strconv.Itoa(n))`,
	}
	for _, text := range geprueft {
		if v, n := druckaufrufeOhneUebersetzer(t, "probe.go", quelle(text)); n != 1 || len(v) != 0 {
			t.Errorf("%s: gelesen %d, gemeldet %v — erwartet einen Aufruf ohne Meldung", text, n, v)
		}
	}
	ungeprueft := []string{
		`name`, `schule.OrtDatum(tag)`, `strings.ToUpper(name)`, `fmt.Sprint(name)`,
		`"Köln"`, `"Fällig"`,
		`fmt.Sprintf("%s", name)`, `fmt.Sprintf("%d Bücher", n)`, `fmt.Sprintf(format, n)`,
		`"Name: " + name`, `name + tr(klasse)`,
	}
	for _, text := range ungeprueft {
		if v, n := druckaufrufeOhneUebersetzer(t, "probe.go", quelle(text)); n != 1 || len(v) != 1 {
			t.Errorf("%s: gelesen %d, gemeldet %v — erwartet eine Meldung", text, n, v)
		}
	}
	// Jede Druckmethode mit ihrer Stelle; gleichnamige Methoden mit anderer Zahl von Argumenten
	// sind keine Druckaufrufe.
	andere := "package x\nfunc f() {\n" +
		"p.Cell(10, 5, name)\np.MultiCell(10, 5, name, \"\", \"L\", false)\np.Text(1, 2, name)\np.Write(5, name)\n" +
		"w.Write(bytes)\nbetrag.Text(wert)\np.Cell(10, 5, tr(name))\n}\n"
	if v, n := druckaufrufeOhneUebersetzer(t, "probe.go", andere); n != 5 || len(v) != 4 {
		t.Errorf("Druckmethoden: gelesen %d, gemeldet %d (%v) — erwartet 5 gelesen und 4 gemeldet", n, len(v), v)
	}
}
