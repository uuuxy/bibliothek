package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Ein Hinweis schickt mit „A → B" an einen Ort der Oberfläche. Der Weg beginnt bei einem
// Menüpunkt, den es gibt (frontend/src/lib/menu.js), wahlweise mit seiner Gruppe davor
// („System → Benutzer & Rechte"); hinter „Einstellungen →" steht eine Kategorie der
// Einstellungen (settings/kategorien.js). Nach einer Umbenennung zeigt der Hinweis sonst weiter
// auf den alten Namen. Der Pfeil steht in Texten deshalb nur für einen Weg; was er sonst
// verbindet, steht mit Grund in wegKeinWeg.
//
// Gelesen werden die Zeichenketten des Go-Codes und die Texte der Oberfläche (frontend/src).
//
// Blind für einen Ort, der ohne Pfeil oder mit einem Pfeil ohne Leerzeichen genannt wird
// („unter Klassensätze"), für die Schritte hinter dem Menüpunkt (Reiter, Knöpfe) außerhalb der
// Einstellungen, für einen Weg, in dem ein Schritt ausgezeichnet oder vom Programm eingesetzt
// ist, und dafür, ob sich am genannten Ort tun lässt, was der Text verspricht.
func TestWegweiserNennenOrteDieEsGibt(t *testing.T) {
	orte := wegOrteDerOberflaeche(t)
	trifft := make([]int, len(wegKeinWeg))
	for _, quelle := range []struct {
		name       string
		texte      []wegText
		mindestens int
	}{
		{"Go-Code", wegTexteAusGo(t, ".."), 8},
		{"Oberfläche", wegTexteDerOberflaeche(t, "../frontend/src"), 3},
	} {
		wege := 0
	texte:
		for _, text := range quelle.texte {
			if len(wegeZuOrten(text.text)) == 0 {
				continue
			}
			for i, ausnahme := range wegKeinWeg {
				if ausnahme.datei == text.datei && strings.Contains(text.text, ausnahme.stueck) {
					trifft[i]++
					continue texte
				}
			}
			wege += len(wegeZuOrten(text.text))
			for _, fehler := range wegweiserFehler(text.text, orte) {
				t.Errorf("%s: %s\n    im Text: %s", text.datei, fehler, strings.TrimSpace(text.text))
			}
		}
		// Nicht-leer-Garantie je Quelle: Ohne gefundene Wege prüft der Test nichts.
		if wege < quelle.mindestens {
			t.Errorf("%s: nur %d Wege gefunden — der Sammler greift nicht mehr", quelle.name, wege)
		}
	}
	for i, ausnahme := range wegKeinWeg {
		if trifft[i] == 0 {
			t.Errorf("Ausnahme ohne Treffer: %s, „%s“ — den Eintrag in wegKeinWeg löschen",
				ausnahme.datei, ausnahme.stueck)
		}
	}
	if t.Failed() {
		t.Log("Nennt der Pfeil in einem beanstandeten Text keinen Weg durch das Menü, gehört der Text mit Grund in wegKeinWeg.")
	}
}

// wegKeinWeg nennt die Texte, in denen der Pfeil keinen Weg durch das Menü zeigt. Eine Ausnahme
// zählt nur an einem Text, den die Prüfung sonst beanstanden könnte.
var wegKeinWeg = []struct{ datei, stueck, grund string }{
	{"repository/schuljahreswechsel.go", "Abgänger werden gesperrt → chk_schueler_block_reason", "Kommentar in einer Abfrage"},
	{"db/seed.go", "vererbung manage_", "Fehlertext über zwei Rechte"},
	{"jobs/backup.go", "S3 upload successful → s3://", "Protokollzeile mit dem Ziel der Auslagerung"},
	{"frontend/src/lib/UserManagementZugangsanfragen.svelte", "„Bearbeiten“ → Aktiv", "Knopf und Wert im Dialog"},
	{"frontend/src/lib/components/settings/kategorien.js", "Klasse → Lehrkraft", "Zuordnung, die die Kategorie pflegt"},
	{"frontend/src/lib/components/students/PromoteStudentsView.svelte", "5a → 6a", "Beispiel einer Versetzung"},
}

// Der Detektor an den Formen, die er finden und durchlassen muss.
func TestWegweiserDetektor_Selbstprobe(t *testing.T) {
	orte := wegOrte{
		gruppen: map[string]map[string]bool{
			"System":   {"Einstellungen": true, "Benutzer & Rechte": true},
			"Berichte": {"Bestandsbücher": true},
		},
		menue: map[string]bool{"Leserdatei": true, "Einstellungen": true, "Ausleihe": true,
			"Benutzer & Rechte": true, "Bestandsbücher": true},
		kategorien: map[string]bool{"Schule": true, "Mahnwesen": true, "Mahnwesen-Routing": true, "LUSD & Versetzung": true},
	}
	for _, fall := range []struct {
		text string
		soll []string
	}{
		{"Leserdatei → Ehemalige / Archiv → Akte öffnen: das Buch melden.", nil},
		{"Einstellungen → Mahnwesen-Routing: Zuordnung nachtragen.", nil},
		{"Angaben eintragen; die Anschrift steht unter Einstellungen → Schule.", nil},
		{"Anschrift der Schule (Einstellungen → Schule)", nil},
		{"Termine unter Einstellungen → LUSD & Versetzung → Sommerferien eintragen, je Jahr.", nil},
		{"Drift nach Code-Änderung: unter Benutzer & Rechte → Rollen & Rechte angleichen.", nil},
		{"Drift nach Code-Änderung: System → Benutzer & Rechte → Rollen & Rechte angleichen.", nil},
		{"System → Einstellungen → Schule: Namen eintragen.", nil},
		{"System → Einstellungen: dort die Kategorie wählen.", nil},
		{"Theke → Meldungen: jede Zeile prüfen.", nil},
		{"Im Zugangsbuch (Berichte → Bestandsbücher). Was im Zulauf bleibt, fehlt dort.", nil},
		{"Schülerdatei → Ehemalige / Archiv → Akte öffnen: das Buch melden.", []string{"Schülerdatei"}},
		{"Drift nach Code-Änderung: System → Berechtigungen angleichen.", []string{"Berechtigungen angleichen"}},
		{"Einstellungen → Klassenlehrer: Zuordnung nachtragen.", []string{"Klassenlehrer"}},
		{"Die Anschrift steht unter Einstellungen → Schulen.", []string{"Schulen"}},
		{"System → Einstellungen → Schulen: Namen eintragen.", []string{"Schulen"}},
		{"Die Adresse (Einstellungen → Allgemein → Schule).", []string{"Allgemein"}},
		{"Im Zugangsbuch (System → Bestandsbücher). Was im Zulauf bleibt, fehlt dort.", []string{"Bestandsbücher"}},
		{"Bewusste Entscheidung → so lassen.", []string{"Bewusste Entscheidung"}},
		// Der Pfeil verbindet Werte, die das Programm einsetzt: kein Weg.
		{"  → %d Titel, %d Exemplare eingetragen", nil},
		{"DRY-RUN abgeschlossen: %d Anmerkungen → %s", nil},
		{"Backup: completed successfully → %s (%.2f MB)", nil},
		{"Zeilen 1 → ", nil},
		{" → ", nil},
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

// Der Sammler für die Oberfläche an den Formen, die er lesen und übergehen muss.
func TestWegweiserStuecke_Selbstprobe(t *testing.T) {
	quelle := "<!-- Kommentar: Alt → Neu -->\n" +
		"<script>\n" +
		"\t// Zeile: Alt → Neu\n" +
		"\tlet a = 1; // dahinter: Alt → Neu\n" +
		"\t/* Block:\n\t   Alt → Neu */\n" +
		"\tconst adresse = 'Unter https://example.org/a steht: Einstellungen → Mail öffnen';\n" +
		"\tconst satz = `Stand ${jahr} fehlt — unter Einstellungen → Schule eintragen, für ${wen}`;\n" +
		"</script>\n" +
		"<p class=\"x\">Im Zugangsbuch (Berichte →\n\t\tBestandsbücher). Mehr</p>\n" +
		"<span>{alt} → {neu}</span>\n"
	soll := []string{
		"Unter https://example.org/a steht: Einstellungen → Mail öffnen",
		"fehlt — unter Einstellungen → Schule eintragen, für $",
		"Im Zugangsbuch (Berichte → Bestandsbücher). Mehr",
		"→",
	}
	var ist []string
	for _, stueck := range wegStuecke(quelle) {
		ist = append(ist, strings.TrimSpace(stueck))
	}
	if !slices.Equal(ist, soll) {
		t.Errorf("gelesen %q, erwartet %q", ist, soll)
	}
}

const wegPfeil = " → "

// wegAndereWoerter: Namen, die im Haus für einen Menüpunkt stehen.
var wegAndereWoerter = map[string]string{"Theke": "Ausleihe"}

// wegOrte sind die Namen, die ein Weg nennen darf; gruppen nennt je Gruppe ihre Menüpunkte.
type wegOrte struct {
	gruppen           map[string]map[string]bool
	menue, kategorien map[string]bool
}

// wegText ist ein Text mit Pfeil und die Datei, in der er steht.
type wegText struct{ datei, text string }

// wegweiserFehler nennt je Weg im Text den Schritt, der an keinen Ort führt.
func wegweiserFehler(text string, orte wegOrte) []string {
	var fehler []string
	for _, schritte := range wegeZuOrten(text) {
		if anderes, ok := wegAndereWoerter[schritte[0]]; ok {
			schritte = append([]string{anderes}, schritte[1:]...)
		}
		gruppe := ""
		if _, ok := orte.gruppen[schritte[0]]; ok {
			gruppe, schritte = schritte[0], schritte[1:]
		}
		punkt, ok := wegBekannt(schritte[0], orte.menue)
		if !ok {
			fehler = append(fehler, "„"+schritte[0]+"“ ist kein Menüpunkt")
			continue
		}
		if gruppe != "" && !orte.gruppen[gruppe][punkt] {
			fehler = append(fehler, "„"+punkt+"“ steht nicht in der Gruppe "+gruppe)
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

// wegeZuOrten sind die Wege eines Texts ohne die, in die das Programm einen Wert einsetzt: Steht
// am Pfeil nichts oder ein Platzhalter, verbindet er Werte.
func wegeZuOrten(text string) [][]string {
	return slices.DeleteFunc(wegeIm(text), func(weg []string) bool {
		return slices.ContainsFunc(weg, func(schritt string) bool {
			return schritt == "" || strings.Contains(schritt, "%")
		})
	})
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
// angleichen"); er gilt als bekannt, wenn er mit einem Namen beginnt. Beginnt er mit mehreren,
// gilt der längste.
func wegBekannt(schritt string, namen map[string]bool) (string, bool) {
	bekannt := ""
	for name := range namen {
		if (schritt == name || strings.HasPrefix(schritt, name+" ")) && len(name) > len(bekannt) {
			bekannt = name
		}
	}
	return bekannt, bekannt != ""
}

var wegTrenner = []string{": ", ". ", ", ", "; ", "(", ")"}

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

// wegOrteDerOberflaeche liest Gruppen und Menüpunkte aus menu.js und die Kategorien der
// Einstellungen aus kategorien.js. Ein Menüpunkt gehört zur Gruppe, deren Name vor ihm steht.
func wegOrteDerOberflaeche(t *testing.T) wegOrte {
	t.Helper()
	orte := wegOrte{
		gruppen:    map[string]map[string]bool{},
		menue:      map[string]bool{},
		kategorien: map[string]bool{},
	}
	gruppe := ""
	for _, treffer := range wegTreffer(t, "../frontend/src/lib/menu.js", `(?m)^\t\tname: '([^']+)'|label: '([^']+)'`) {
		if treffer[1] != "" {
			gruppe = treffer[1]
			orte.gruppen[gruppe] = map[string]bool{}
			continue
		}
		if gruppe == "" {
			t.Fatalf("menu.js: Menüpunkt „%s“ steht vor der ersten Gruppe — Form geändert?", treffer[2])
		}
		orte.gruppen[gruppe][treffer[2]] = true
		orte.menue[treffer[2]] = true
	}
	for _, treffer := range wegTreffer(t, "../frontend/src/lib/components/settings/kategorien.js", `titel: '([^']+)'`) {
		orte.kategorien[treffer[1]] = true
	}
	// Nicht-leer-Garantie je Quelle.
	if len(orte.gruppen) < 5 || len(orte.menue) < 10 || len(orte.kategorien) < 10 {
		t.Fatalf("nur %d Gruppen, %d Menüpunkte, %d Kategorien gefunden — Form geändert? Dann das Muster nachziehen.",
			len(orte.gruppen), len(orte.menue), len(orte.kategorien))
	}
	for name, punkte := range orte.gruppen {
		if len(punkte) == 0 {
			t.Fatalf("menu.js: Gruppe „%s“ ohne Menüpunkt gelesen — Form geändert?", name)
		}
	}
	return orte
}

// wegTreffer liest die Treffer eines Musters aus einer Datei der Oberfläche.
func wegTreffer(t *testing.T, datei, muster string) [][]string {
	t.Helper()
	quelle, err := os.ReadFile(datei)
	if err != nil {
		t.Fatalf("%s lesen: %v", datei, err)
	}
	return regexp.MustCompile(muster).FindAllStringSubmatch(string(quelle), -1)
}

// wegOhneGoCode: Ordner, deren Go-Dateien keine Texte des Programms tragen. docs/docs.go
// entsteht aus Kommentaren.
var wegOhneGoCode = []string{"frontend", "node_modules", "tmp", "docs"}

// wegTexteAusGo liest die Zeichenketten mit Pfeil aus dem Go-Code unter der Wurzel, ohne Tests.
func wegTexteAusGo(t *testing.T, wurzel string) []wegText {
	t.Helper()
	var texte []wegText
	fset := token.NewFileSet()
	err := filepath.WalkDir(wurzel, func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := eintrag.Name()
		if eintrag.IsDir() {
			if pfad != wurzel && (strings.HasPrefix(name, ".") || slices.Contains(wegOhneGoCode, name)) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		datei, err := parser.ParseFile(fset, pfad, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(datei, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if s, err := strconv.Unquote(lit.Value); err == nil && strings.Contains(s, wegPfeil) {
					texte = append(texte, wegText{wegDatei(wurzel, pfad), s})
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("Go-Code lesen: %v", err)
	}
	return texte
}

// wegTexteDerOberflaeche liest die Texte mit Pfeil aus den Quelldateien der Oberfläche.
func wegTexteDerOberflaeche(t *testing.T, wurzel string) []wegText {
	t.Helper()
	var texte []wegText
	err := filepath.WalkDir(wurzel, func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := eintrag.Name()
		if eintrag.IsDir() {
			if name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(name, ".test.js") || (!strings.HasSuffix(name, ".svelte") && !strings.HasSuffix(name, ".js")) {
			return nil
		}
		quelle, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		for _, stueck := range wegStuecke(string(quelle)) {
			texte = append(texte, wegText{wegDatei("..", pfad), stueck})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Oberfläche lesen: %v", err)
	}
	return texte
}

// wegDatei nennt eine Datei so, wie sie im Repository steht.
func wegDatei(wurzel, pfad string) string {
	rel, err := filepath.Rel(wurzel, pfad)
	if err != nil {
		return pfad
	}
	return filepath.ToSlash(rel)
}

var (
	// Ein Zeilenkommentar beginnt am Zeilenanfang oder nach Leerraum; „https://" bleibt stehen.
	wegKommentar = regexp.MustCompile(`(?s)<!--.*?-->|/\*.*?\*/|(?m:(?:^|\s)//[^\n]*)`)
	wegGrenze    = regexp.MustCompile("[<>{}'\"`]")
)

// wegStuecke sind die Texte einer Quelldatei der Oberfläche, die einen Pfeil tragen: ohne
// Kommentare, Leerraum zusammengezogen, getrennt an Auszeichnung, Anführungszeichen und
// eingesetzten Werten. Ein Stück beginnt und endet mit einem Leerzeichen, damit auch ein Pfeil
// an seinem Rand als Pfeil gilt.
func wegStuecke(quelle string) []string {
	var stuecke []string
	for _, stueck := range wegGrenze.Split(wegKommentar.ReplaceAllString(quelle, " "), -1) {
		stueck = " " + strings.Join(strings.Fields(stueck), " ") + " "
		if strings.Contains(stueck, wegPfeil) {
			stuecke = append(stuecke, stueck)
		}
	}
	return stuecke
}
