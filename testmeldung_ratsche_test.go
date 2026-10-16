package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var (
	testmeldungEnglisch        = regexp.MustCompile(`(?i)\b(got|want|wanted|expected|actual)\b`)
	testmeldungBeginntSoll     = regexp.MustCompile(`(?i)^\s*erwartet\b`)
	testmeldungSollVorErgebnis = regexp.MustCompile(`(?i)erwartet\b[^"]*?[,;]\s*(aber\s+)?(bekam|bekommen|war|waren|ist|sind|kam|lieferte|geliefert|gelesen|erhalten|gefunden|gezählt|steht|gesehen|tatsächlich)\b`)
)

// Eine Meldung im Test nennt, was geprüft wurde, dann das Ergebnis, dann das Erwartete:
// „sichereCoverURL(%q) = %q, erwartet %q" (STIL.md 10; Decisions, „Identify the function" und
// „Got before want"). Die Prüfung liest den Formattext jeder Meldung in t.Error, t.Errorf,
// t.Fatal und t.Fatalf und meldet englische Wörter für Ergebnis und Erwartetes, einen Text, der
// mit „erwartet" beginnt, und das Erwartete vor dem Ergebnis („erwartet 2, bekam 3").
//
// Blindheit: Meldungen aus Variablen oder Hilfsfunktionen, andere englische Wörter, und ob das
// genannte Subjekt die geprüfte Funktion ist, liest ein Mensch.
func TestTestmeldung_ErgebnisVorErwartetem(t *testing.T) {
	fset := token.NewFileSet()
	var funde []string
	gelesen := 0
	for pfad, quelle := range leseGoQuellen(t, ".") {
		if !strings.HasSuffix(pfad, "_test.go") {
			continue
		}
		datei, err := parser.ParseFile(fset, pfad, quelle, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s lesen: %v", pfad, err)
		}
		ast.Inspect(datei, func(knoten ast.Node) bool {
			text, ok := testmeldungstext(knoten)
			if !ok {
				return true
			}
			gelesen++
			grund := ""
			switch {
			case testmeldungEnglisch.MatchString(text):
				grund = "englisch"
			case testmeldungBeginntSoll.MatchString(text):
				grund = "beginnt mit „erwartet“"
			case testmeldungSollVorErgebnis.MatchString(text):
				grund = "Erwartetes vor dem Ergebnis"
			default:
				return true
			}
			funde = append(funde, fset.Position(knoten.Pos()).String()+" ("+grund+"): "+strconv.Quote(text))
			return true
		})
	}

	// Gegenprobe am Detektor: Ohne die Meldungen der Tests liest er den falschen Baum.
	if gelesen < 8000 {
		t.Fatalf("nur %d Meldungen gelesen: Der Detektor liest den falschen Baum", gelesen)
	}
	sort.Strings(funde)
	for _, fund := range funde {
		t.Errorf("%s. Form: „Subjekt = Ergebnis, erwartet Soll“.", fund)
	}
}

// testmeldungstext liefert den Formattext eines Aufrufs von Error, Errorf, Fatal oder Fatalf, wenn
// er als Zeichenkette dasteht.
func testmeldungstext(knoten ast.Node) (string, bool) {
	aufruf, ok := knoten.(*ast.CallExpr)
	if !ok || len(aufruf.Args) == 0 {
		return "", false
	}
	auswahl, ok := aufruf.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	switch auswahl.Sel.Name {
	case "Error", "Errorf", "Fatal", "Fatalf":
	default:
		return "", false
	}
	literal, ok := aufruf.Args[0].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	text, err := strconv.Unquote(literal.Value)
	return text, err == nil
}
