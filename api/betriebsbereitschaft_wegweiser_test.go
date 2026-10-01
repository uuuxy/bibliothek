package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Ein Hinweis der Selbstprüfung schickt mit „A → B" an einen Ort der Oberfläche. Der Weg beginnt
// bei einem Menüpunkt, den es gibt (frontend/src/lib/menu.js), wahlweise mit seiner Gruppe davor
// („System → Benutzer & Rechte"); hinter „Einstellungen →" steht eine Kategorie der
// Einstellungen (settings/kategorien.js). Nach einer Umbenennung zeigt der Hinweis sonst weiter
// auf den alten Namen. Der Pfeil steht in diesen Texten deshalb nur für einen Weg.
//
// Blind für einen Ort, der ohne Pfeil genannt wird („unter Klassensätze"), und für die Schritte
// hinter dem Menüpunkt (Reiter, Knöpfe) außerhalb der Einstellungen.
func TestWegweiserDerSelbstpruefungNennenOrteDieEsGibt(t *testing.T) {
	orte := wegOrte{
		gruppen:    wegNamenAus(t, "../frontend/src/lib/menu.js", `(?m)^\t\tname: '([^']+)'`, 5),
		menue:      wegNamenAus(t, "../frontend/src/lib/menu.js", `label: '([^']+)'`, 10),
		kategorien: wegNamenAus(t, "../frontend/src/lib/components/settings/kategorien.js", `titel: '([^']+)'`, 10),
	}

	datei, err := parser.ParseFile(token.NewFileSet(), "betriebsbereitschaft.go", nil, 0)
	if err != nil {
		t.Fatalf("betriebsbereitschaft.go lesen: %v", err)
	}
	var texte []string
	ast.Inspect(datei, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if s, err := strconv.Unquote(lit.Value); err == nil && strings.Contains(s, wegPfeil) {
				texte = append(texte, s)
			}
		}
		return true
	})
	// Nicht-leer-Garantie: Ohne gefundene Wege prüft der Test nichts.
	if len(texte) < 8 {
		t.Fatalf("nur %d Texte mit einem Weg gefunden — der Sammler greift nicht mehr", len(texte))
	}
	for _, text := range texte {
		for _, fehler := range wegweiserFehler(text, orte) {
			t.Errorf("%s\n    im Text: %s", fehler, text)
		}
	}
}

// Der Detektor an den Formen, die er finden und durchlassen muss.
func TestWegweiserDetektor_Selbstprobe(t *testing.T) {
	orte := wegOrte{
		gruppen:    map[string]bool{"System": true},
		menue:      map[string]bool{"Leserdatei": true, "Einstellungen": true, "Ausleihe": true, "Benutzer & Rechte": true},
		kategorien: map[string]bool{"Schule": true, "Mahnwesen-Routing": true, "LUSD & Versetzung": true},
	}
	for _, fall := range []struct {
		text string
		soll []string
	}{
		{"Leserdatei → Ehemalige / Archiv → Akte öffnen: das Buch melden.", nil},
		{"Einstellungen → Mahnwesen-Routing: Zuordnung nachtragen.", nil},
		{"Angaben eintragen; die Anschrift steht unter Einstellungen → Schule.", nil},
		{"Termine unter Einstellungen → LUSD & Versetzung → Sommerferien eintragen, je Jahr.", nil},
		{"Drift nach Code-Änderung: unter Benutzer & Rechte → Rollen & Rechte angleichen.", nil},
		{"Drift nach Code-Änderung: System → Benutzer & Rechte → Rollen & Rechte angleichen.", nil},
		{"System → Einstellungen → Schule: Namen eintragen.", nil},
		{"System → Einstellungen: dort die Kategorie wählen.", nil},
		{"Theke → Meldungen: jede Zeile prüfen.", nil},
		{"Schülerdatei → Ehemalige / Archiv → Akte öffnen: das Buch melden.", []string{"Schülerdatei"}},
		{"Drift nach Code-Änderung: System → Berechtigungen angleichen.", []string{"Berechtigungen angleichen"}},
		{"Einstellungen → Klassenlehrer: Zuordnung nachtragen.", []string{"Klassenlehrer"}},
		{"Die Anschrift steht unter Einstellungen → Schulen.", []string{"Schulen"}},
		{"System → Einstellungen → Schulen: Namen eintragen.", []string{"Schulen"}},
		{"Bewusste Entscheidung → so lassen.", []string{"Bewusste Entscheidung"}},
	} {
		var ist []string
		for _, fehler := range wegweiserFehler(fall.text, orte) {
			ist = append(ist, regexp.MustCompile(`„([^“]+)“`).FindStringSubmatch(fehler)[1])
		}
		if !slices.Equal(ist, fall.soll) {
			t.Errorf("%q: beanstandet %v, erwartet %v", fall.text, ist, fall.soll)
		}
	}
}

const wegPfeil = " → "

// wegAndereWoerter: Namen, die im Haus für einen Menüpunkt stehen.
var wegAndereWoerter = map[string]string{"Theke": "Ausleihe"}

// wegOrte sind die Namen, die ein Weg nennen darf.
type wegOrte struct {
	gruppen, menue, kategorien map[string]bool
}

// wegweiserFehler nennt je Weg im Text den Schritt, der an keinen Ort führt.
func wegweiserFehler(text string, orte wegOrte) []string {
	var fehler []string
	for _, weg := range wegeIm(text) {
		schritte := weg
		if anderes, ok := wegAndereWoerter[schritte[0]]; ok {
			schritte = append([]string{anderes}, schritte[1:]...)
		}
		if orte.gruppen[schritte[0]] {
			schritte = schritte[1:]
		}
		punkt, ok := wegBekannt(schritte[0], orte.menue)
		if !ok {
			fehler = append(fehler, "„"+schritte[0]+"“ ist kein Menüpunkt")
			continue
		}
		// Ein Weg, der bei den Einstellungen endet, nennt keine Kategorie.
		if punkt == "Einstellungen" && len(schritte) > 1 {
			if _, ok := wegBekannt(schritte[1], orte.kategorien); !ok {
				fehler = append(fehler, "„"+schritte[1]+"“ ist keine Kategorie der Einstellungen")
			}
		}
	}
	return fehler
}

// wegeIm zerlegt einen Text in seine Wege, jeden in seine Schritte. Zwischen zwei Pfeilen steht
// entweder ein weiterer Schritt oder, mit Satzzeichen, das Ende eines Wegs und der Anfang des
// nächsten.
func wegeIm(text string) [][]string {
	teile := strings.Split(text, wegPfeil)
	var wege [][]string
	weg := []string{wegSchrittAmEnde(teile[0])}
	for _, teil := range teile[1 : len(teile)-1] {
		if ende := wegSchrittAmAnfang(teil); ende != strings.TrimSpace(teil) {
			wege = append(wege, append(weg, ende))
			weg = []string{wegSchrittAmEnde(teil)}
			continue
		}
		weg = append(weg, strings.TrimSpace(teil))
	}
	return append(wege, append(weg, wegSchrittAmAnfang(teile[len(teile)-1])))
}

// wegBekannt: Der letzte Schritt eines Wegs trägt oft noch das Verb („Rollen & Rechte
// angleichen"); er gilt als bekannt, wenn er mit einem Namen beginnt.
func wegBekannt(schritt string, namen map[string]bool) (string, bool) {
	for name := range namen {
		if schritt == name || strings.HasPrefix(schritt, name+" ") {
			return name, true
		}
	}
	return "", false
}

var wegTrenner = []string{": ", ". ", ", ", "; ", "("}

// wegSchrittAmEnde schneidet vom Text vor dem Pfeil alles ab, was vor dem Namen des Orts steht.
func wegSchrittAmEnde(vor string) string {
	for _, trenner := range append(wegTrenner, "unter ", "Unter ") {
		if i := strings.LastIndex(vor, trenner); i >= 0 {
			vor = vor[i+len(trenner):]
		}
	}
	return strings.TrimSpace(vor)
}

// wegSchrittAmAnfang schneidet vom Text hinter dem Pfeil alles ab, was nach dem Schritt kommt.
func wegSchrittAmAnfang(nach string) string {
	for _, trenner := range wegTrenner {
		if i := strings.Index(nach, strings.TrimSpace(trenner)); i >= 0 {
			nach = nach[:i]
		}
	}
	return strings.TrimSpace(nach)
}

// wegNamenAus liest die Namen eines Musters aus einer Datei der Oberfläche.
func wegNamenAus(t *testing.T, datei, muster string, mindestens int) map[string]bool {
	t.Helper()
	quelle, err := os.ReadFile(datei)
	if err != nil {
		t.Fatalf("%s lesen: %v", datei, err)
	}
	namen := map[string]bool{}
	for _, treffer := range regexp.MustCompile(muster).FindAllStringSubmatch(string(quelle), -1) {
		namen[treffer[1]] = true
	}
	// Nicht-leer-Garantie je Quelle.
	if len(namen) < mindestens {
		t.Fatalf("%s: nur %d Namen gefunden — Form geändert? Dann das Muster nachziehen.", datei, len(namen))
	}
	return namen
}
