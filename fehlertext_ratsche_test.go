package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"
)

// Prüfung: Ein Fehlertext beginnt klein und endet ohne Satzzeichen.
//
// Warum: Ein Fehlertext steht meist mitten in einem anderen („konto löschen: konto hat 2 offene
// ausleihen"); groß geschrieben und mit Punkt zerfällt die Kette (Decisions, „Error strings").
// Den Satz für die Oberfläche schreibt die Tür (docs/STIL.md, Abschnitt 11). staticcheck (ST1005)
// sieht nur errors.New und fmt.Errorf mit einem Literal und lässt ein großes erstes Wort durch,
// sobald ein Bezeichner des Pakets so heißt („Sommerferien", „Leser"); diese Prüfung liest auch
// Konstanten, zusammengesetzte Texte und die Methoden Error.
//
// Regel: Der Text von errors.New, fmt.Errorf und der Rückgabe einer Methode Error() beginnt nicht
// mit einem Großbuchstaben, außer das erste Wort ist ein Kürzel (LUSD, CSV-Kopfzeile), ein Go-Name
// mit innerem Großbuchstaben (LadeKonto) oder ein Eigenname aus fehlertextEigennamen. Er endet
// nicht auf . : ! ? … oder einen Zeilenumbruch. Ein nolint für ST1005 gibt es nicht.
//
// Sieht nicht: Texte, die erst zur Laufzeit entstehen (Variablen, Rückgaben anderer Funktionen),
// und eigene Funktionen, die einen Text an errors.New weiterreichen.

// fehlertextEigennamen sind Eigennamen, mit denen ein Fehlertext beginnen darf.
var fehlertextEigennamen = map[string]bool{"Excel": true}

// fehlertextBefund zählt, was die Prüfung gelesen hat, und nennt jeden Verstoß.
type fehlertextBefund struct {
	gelesen    map[string]int // je Form: errors.New, fmt.Errorf, Error()
	verstoesse []string
}

func TestFehlertexte_KleinOhneSatzzeichen(t *testing.T) {
	befund := pruefeFehlertexte(t, ".")
	// Je Form eine Untergrenze: Liest der Durchlauf eine Form nicht mehr, prüft er sie nicht.
	for form, mindestens := range map[string]int{"errors.New": 300, "fmt.Errorf": 600, "Error()": 15} {
		if befund.gelesen[form] < mindestens {
			t.Errorf("nur %d Texte aus %s gelesen, erwartet mindestens %d — der Detektor läuft leer",
				befund.gelesen[form], form, mindestens)
		}
	}
	if len(befund.verstoesse) > 0 {
		t.Errorf("Fehlertext groß begonnen oder mit Satzzeichen beendet (Decisions, „Error strings\"):\n  %s\n"+
			"Fix: klein beginnen, ohne Satzzeichen enden. Ein Satz für die Oberfläche geht als Meldung hinaus "+
			"(apierrors.Conflict(\"Satz.\", err), AntworteFehlerMitMeldung) oder als Feld eines Fehlertyps, "+
			"siehe docs/STIL.md, Abschnitt 11.", strings.Join(befund.verstoesse, "\n  "))
	}
}

// pruefeFehlertexte liest alle Go-Dateien unter wurzel, Tests eingeschlossen.
func pruefeFehlertexte(t *testing.T, wurzel string) fehlertextBefund {
	t.Helper()
	fset := token.NewFileSet()
	dateien := map[string][]*ast.File{} // je Ordner, für die Konstanten des Pakets
	err := filepath.WalkDir(wurzel, func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if eintrag.IsDir() {
			switch eintrag.Name() {
			case "frontend", "node_modules", ".git", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") {
			return nil
		}
		datei, err := parser.ParseFile(fset, pfad, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		dateien[filepath.Dir(pfad)] = append(dateien[filepath.Dir(pfad)], datei)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	befund := fehlertextBefund{gelesen: map[string]int{}}
	for _, paket := range dateien {
		konstanten := textKonstanten(paket)
		for _, datei := range paket {
			pruefeDatei(fset, datei, konstanten, &befund)
		}
	}
	sort.Strings(befund.verstoesse)
	return befund
}

// textKonstanten sammelt die Konstanten eines Pakets, deren Wert ein Text ist.
func textKonstanten(paket []*ast.File) map[string]string {
	konstanten := map[string]string{}
	for _, datei := range paket {
		ast.Inspect(datei, func(n ast.Node) bool {
			decl, ok := n.(*ast.GenDecl)
			if !ok || decl.Tok != token.CONST {
				return true
			}
			for _, spec := range decl.Specs {
				werte, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range werte.Names {
					if i < len(werte.Values) {
						if lit, ok := werte.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							if text, err := strconv.Unquote(lit.Value); err == nil {
								konstanten[name.Name] = text
							}
						}
					}
				}
			}
			return true
		})
	}
	return konstanten
}

func pruefeDatei(fset *token.FileSet, datei *ast.File, konstanten map[string]string, befund *fehlertextBefund) {
	melde := func(pos token.Pos, form, grund string, text fehlertextTeile) {
		wortlaut := text.anfang + "…" + text.ende
		if text.anfang == text.ende {
			wortlaut = text.anfang
		}
		befund.verstoesse = append(befund.verstoesse, fset.Position(pos).String()+": "+form+" "+grund+": "+
			strconv.Quote(wortlaut))
	}
	pruefe := func(pos token.Pos, form string, text fehlertextTeile) {
		befund.gelesen[form]++
		if grund := fehlertextVerstoss(text); grund != "" {
			melde(pos, form, grund, text)
		}
	}
	for _, gruppe := range datei.Comments {
		for _, kommentar := range gruppe.List {
			if strings.HasPrefix(kommentar.Text, "//nolint") && strings.Contains(kommentar.Text, "ST1005") {
				befund.verstoesse = append(befund.verstoesse, fset.Position(kommentar.Pos()).String()+
					": nolint für ST1005 — ein Fehlertext folgt der Regel, eine Meldung ist kein Fehlertext")
			}
		}
	}
	ast.Inspect(datei, func(n ast.Node) bool {
		switch knoten := n.(type) {
		case *ast.CallExpr:
			if form := fehlertextAufruf(knoten); form != "" && len(knoten.Args) > 0 {
				pruefe(knoten.Pos(), form, textTeile(knoten.Args[0], konstanten))
			}
		case *ast.FuncDecl:
			if knoten.Recv == nil || knoten.Name.Name != "Error" || knoten.Body == nil {
				return true
			}
			ast.Inspect(knoten.Body, func(m ast.Node) bool {
				if _, ok := m.(*ast.FuncLit); ok {
					return false
				}
				if rueckgabe, ok := m.(*ast.ReturnStmt); ok && len(rueckgabe.Results) == 1 {
					pruefe(rueckgabe.Pos(), "Error()", textTeile(rueckgabe.Results[0], konstanten))
				}
				return true
			})
		}
		return true
	})
}

// fehlertextAufruf nennt die Form eines Aufrufs, der einen Fehlertext nimmt, sonst "".
func fehlertextAufruf(aufruf *ast.CallExpr) string {
	sel, ok := aufruf.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	paket, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	switch name := paket.Name + "." + sel.Sel.Name; name {
	case "errors.New", "fmt.Errorf":
		return name
	}
	return ""
}

// fehlertextTeile ist, was über Anfang und Ende eines Textes bekannt ist. Ein leerer Teil ist
// unbekannt: Der Text entsteht dort erst zur Laufzeit.
type fehlertextTeile struct{ anfang, ende string }

// textTeile liest Anfang und Ende eines Ausdrucks: ein Literal, eine Konstante des Pakets, eine
// Verkettung oder fmt.Sprintf mit festem Format.
func textTeile(e ast.Expr, konstanten map[string]string) fehlertextTeile {
	switch x := e.(type) {
	case *ast.BasicLit:
		if x.Kind == token.STRING {
			if text, err := strconv.Unquote(x.Value); err == nil {
				return fehlertextTeile{text, text}
			}
		}
	case *ast.Ident:
		if text, ok := konstanten[x.Name]; ok {
			return fehlertextTeile{text, text}
		}
	case *ast.ParenExpr:
		return textTeile(x.X, konstanten)
	case *ast.BinaryExpr:
		if x.Op == token.ADD {
			return fehlertextTeile{textTeile(x.X, konstanten).anfang, textTeile(x.Y, konstanten).ende}
		}
	case *ast.CallExpr:
		if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Sprintf" && len(x.Args) > 0 {
			if paket, ok := sel.X.(*ast.Ident); ok && paket.Name == "fmt" {
				return textTeile(x.Args[0], konstanten)
			}
		}
	}
	return fehlertextTeile{}
}

// fehlertextVerstoss nennt, wogegen ein Text verstößt, sonst "".
func fehlertextVerstoss(text fehlertextTeile) string {
	if text.ende != "" {
		letztes, _ := utf8.DecodeLastRuneInString(text.ende)
		if strings.ContainsRune(".:!?…\n", letztes) {
			return "endet mit Satzzeichen"
		}
	}
	if text.anfang == "" {
		return ""
	}
	wort := text.anfang
	if i := strings.IndexAny(wort, " :,;("); i >= 0 {
		wort = wort[:i]
	}
	erstes, _ := utf8.DecodeRuneInString(wort)
	if unicode.IsUpper(erstes) && !darfGrossBeginnen(wort) {
		return "beginnt groß"
	}
	return ""
}

// darfGrossBeginnen lässt ein erstes Wort gelten, das ein Kürzel ist (vor einem Bindestrich nur
// Großbuchstaben oder eine Ziffer: LUSD, CSV-Kopfzeile, B2B), ein Go-Name mit innerem
// Großbuchstaben ohne Bindestrich (LadeKonto) oder mit einem Eigennamen beginnt.
func darfGrossBeginnen(wort string) bool {
	teil, _, _ := strings.Cut(wort, "-")
	if fehlertextEigennamen[teil] {
		return true
	}
	buchstaben, gross := 0, 0
	for _, c := range teil {
		if unicode.IsDigit(c) {
			return true
		}
		if unicode.IsLetter(c) {
			buchstaben++
			if unicode.IsUpper(c) {
				gross++
			}
		}
	}
	if buchstaben >= 2 && gross == buchstaben {
		return true
	}
	return !strings.Contains(wort, "-") && gross >= 2
}

// Gegenprobe am Detektor: jede Form, die er melden muss (markiert mit „// Verstoß"), und jede,
// die er gelten lässt. Der Bezeichner Leser steht im Paket, damit der Fall dabei ist, den
// staticcheck durchlässt.
func TestFehlertextDetektorErkenntDieFormen(t *testing.T) {
	const quelle = `package p

import (
	"errors"
	"fmt"
)

type Leser struct{}

const satz = "Satz für die Oberfläche."

type zeilenFehler struct{ nr int }

func (e *zeilenFehler) Error() string { return fmt.Sprintf("Zeile %d ist kaputt", e.nr) } // Verstoß

type kurzFehler struct{}

func (kurzFehler) Error() string { return "Kurz" + " und knapp" } // Verstoß

type gutFehler struct{}

func (gutFehler) Error() string { return "kurz und " + "knapp" }

func f(name string) {
	_ = errors.New("Konto hat Ausleihen")      // Verstoß
	_ = errors.New("konto hat ausleihen.")     // Verstoß
	_ = fmt.Errorf("Leser %s fehlt", name)     // Verstoß
	_ = errors.New(satz)                       // Verstoß
	_ = errors.New("wert " + "ungültig?")      // Verstoß
	_ = errors.New("zeile\n")                  // Verstoß
	_ = errors.New("LUSD-Datei fehlt")
	_ = errors.New("Excel-Datei ist kaputt")
	_ = fmt.Errorf("LadeKonto: %w", errors.New("verbindung weg"))
	_ = errors.New("ein Satz mit Substantiv")
	_ = errors.New(name + " Fehlt")
	_ = fmt.Errorf("%w: Grund in Worten", errors.New("art"))
}

//nolint:staticcheck // ST1005: begründet // Verstoß
var _ = 1
`
	ordner := t.TempDir()
	if err := os.WriteFile(filepath.Join(ordner, "p.go"), []byte(quelle), 0o600); err != nil {
		t.Fatal(err)
	}
	var erwartet []int
	for nr, zeile := range strings.Split(quelle, "\n") {
		if strings.HasSuffix(zeile, "// Verstoß") {
			erwartet = append(erwartet, nr+1)
		}
	}
	var gemeldet []int
	for _, verstoss := range pruefeFehlertexte(t, ordner).verstoesse {
		_, rest, _ := strings.Cut(verstoss, "p.go:")
		nr, err := strconv.Atoi(rest[:strings.Index(rest, ":")])
		if err != nil {
			t.Fatalf("Zeile in %q: %v", verstoss, err)
		}
		gemeldet = append(gemeldet, nr)
	}
	sort.Ints(gemeldet)
	if diff := cmp.Diff(erwartet, gemeldet); diff != "" {
		t.Errorf("pruefeFehlertexte meldet andere Zeilen (-erwartet +gemeldet):\n%s", diff)
	}
}
