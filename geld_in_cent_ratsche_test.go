package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Geld wird in Cent geführt (betrag.Cent), nicht als float64. float64 kennt einen Betrag wie
// 0,10 nicht genau: Drei Posten zu 0,10 € ergaben 0.30000000000000004, und ein Anteil, der auf
// einen halben Cent fiel, kippte beim Runden (10 % von 10,05 € gaben 1,00 € statt 1,01 €).
//
// Die Regel: Kein Feld, kein Parameter, kein Ergebnis und keine Variable mit einem Namen für
// Geld (Betrag, Preis, Summe, Kosten, Ersatzwert) hat den Typ float64. Was noch umzustellen ist,
// führt geldAlsFloatBestand; er kann nur sinken.
//
// Blindheit: Erkennt Geld am Namen. Ein Betrag unter anderem Namen (wert, x) und eine
// Variable aus := ohne genannten Typ sieht sie nicht; eine Abfrage, die einen Betrag mit
// ::float8 in einen Cent liest, scheitert am Lesen (betrag.Cent.Scan nimmt kein float64).
func TestGeldInCent(t *testing.T) {
	gefunden := map[string]bool{}
	gelesen := 0
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "frontend", "testdata":
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
		gelesen++
		for _, name := range geldAlsFloat(datei) {
			gefunden[filepath.ToSlash(pfad)+": "+name] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum nicht lesbar: %v", err)
	}
	if gelesen < 300 {
		t.Fatalf("nur %d Dateien gelesen — die Prüfung misst nichts", gelesen)
	}

	var neu, weg []string
	for eintrag := range gefunden {
		if !geldAlsFloatBestand[eintrag] {
			neu = append(neu, eintrag)
		}
	}
	for eintrag := range geldAlsFloatBestand {
		if !gefunden[eintrag] {
			weg = append(weg, eintrag)
		}
	}
	sort.Strings(neu)
	sort.Strings(weg)
	for _, eintrag := range neu {
		t.Errorf("%s ist Geld als float64 — als betrag.Cent führen", eintrag)
	}
	for _, eintrag := range weg {
		t.Errorf("%s steht im Bestand, ist aber umgestellt oder fort — aus geldAlsFloatBestand streichen", eintrag)
	}
}

// geldName erkennt einen Namen für Geld, auch zusammengesetzt (Einzelpreis, betragLand).
var geldName = regexp.MustCompile(`(?i)betrag|preis|summe|kosten|ersatzwert`)

// geldAlsFloat nennt die Namen der Felder, Parameter, Ergebnisse und Variablen einer Datei, die
// Geld heißen und float64 sind (auch als Zeiger oder Liste).
func geldAlsFloat(datei *ast.File) []string {
	var namen []string
	pruefe := func(idents []*ast.Ident, typ ast.Expr) {
		if typ == nil || !strings.Contains(types.ExprString(typ), "float64") {
			return
		}
		for _, id := range idents {
			if geldName.MatchString(id.Name) {
				namen = append(namen, id.Name)
			}
		}
	}
	ast.Inspect(datei, func(n ast.Node) bool {
		switch k := n.(type) {
		case *ast.Field:
			pruefe(k.Names, k.Type)
		case *ast.ValueSpec:
			pruefe(k.Names, k.Type)
		case *ast.FuncDecl:
			// Eine Funktion, die Geld heißt und es ohne benanntes Ergebnis liefert (preisAus).
			if k.Type.Results != nil {
				for _, ergebnis := range k.Type.Results.List {
					if len(ergebnis.Names) == 0 {
						pruefe([]*ast.Ident{k.Name}, ergebnis.Type)
					}
				}
			}
		}
		return true
	})
	return namen
}

func TestGeldAlsFloat_ErkenntDieFormen(t *testing.T) {
	quelle := `package p
type T struct {
	Gesamtbetrag float64
	Einzelpreis  *float64
	listenpreise []*float64
	Betrag       betrag.Cent
	Menge        float64
}
func f(betragLand float64, n int) (summe float64) { var kosten float64; return }
func preisAus(roh string) float64 { return 0 }
func menge() float64 { return 0 }`
	datei, err := parser.ParseFile(token.NewFileSet(), "probe.go", quelle, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(geldAlsFloat(datei), " ")
	if want := "Gesamtbetrag Einzelpreis listenpreise betragLand summe kosten preisAus"; got != want {
		t.Errorf("geldAlsFloat = %q, erwartet %q", got, want)
	}
}

// geldAlsFloatBestand sind die Stellen, die noch umzustellen sind: die Bestellungen und die
// Preise an Titel und Exemplar, dazu die Schreibweise für float64 in pkg/betrag, die sie
// brauchen.
var geldAlsFloatBestand = map[string]bool{
	"api/bestellung/bestellhistorie_uebersicht.go: Gesamtbetrag": true,
	"api/bestellung/bestellungen.go: Preis":                      true,
	"api/katalog/titel_aendern.go: preis":                        true,
	"api/katalog/titel_anlegen.go: Listenpreis":                  true,
	"api/katalog/titel_anlegen.go: listenpreisAusNachschlagen":   true,
	"internal/katalogdienste/dnb.go: preis":                      true,
	"internal/katalogdienste/katalogdienste.go: Preis":           true,
	"internal/katalogdienste/preis.go: preisAusVerfuegbarkeit":   true,
	"internal/littera/titel_exemplar.go: Preis":                  true,
	"internal/littera/titel_exemplar.go: preisAus":               true,
	"internal/service/bestellsuche.go: PreisVorschlag":           true,
	"internal/service/bestellung_anlegen.go: Preis":              true,
	"internal/service/bestellung_anlegen.go: betrag":             true,
	"internal/service/bestellung_anlegen.go: gesamtbetrag":       true,
	"pdf/bestellbericht.go: Einzelpreis":                         true,
	"pdf/bestellbericht.go: GesamtBetrag":                        true,
	"pdf/bestellbericht.go: Gesamtbetrag":                        true,
	"pdf/bestellbericht.go: gesamtBetrag":                        true,
	"pdf/bestellbericht.go: summe":                               true,
	"pdf/bestellbericht_uebersicht.go: betrag":                   true,
	"pdf/bestellbericht_uebersicht.go: gesamtBetrag":             true,
	"pkg/betrag/betrag.go: betrag":                               true,
	"repository/bestellbericht.go: Einzelpreis":                  true,
	"repository/bestellbericht.go: Gesamtbetrag":                 true,
	"repository/bestelldetail.go: Einzelpreis":                   true,
	"repository/bestelldetail.go: Gesamtbetrag":                  true,
	"repository/bestelldetail.go: Gesamtpreis":                   true,
	"repository/bestellkennzahlen.go: Gesamtbetrag":              true,
	"repository/bestellkopf.go: Einzelpreis":                     true,
	"repository/bestellkopf.go: Gesamtbetrag":                    true,
	"repository/bestellverlauf.go: Einzelpreis":                  true,
	"repository/bestellverlauf.go: Gesamtbetrag":                 true,
	"repository/bestellverlauf.go: Gesamtpreis":                  true,
	"repository/exemplar.go: Einkaufspreis":                      true,
	"repository/katalog_titel.go: Listenpreis":                   true,
	"repository/titel_anlegen.go: listenpreise":                  true,
	"repository/titel_isbn.go: Listenpreis":                      true,
}
