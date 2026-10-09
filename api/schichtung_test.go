package api

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Schichtung des Backends: Eine Tür in api/ formuliert kein SQL und trägt keine Regeln. Jede
// Regel einer Abfrage (NULL-Behandlung, Schutz vor leeren Werten, Reihenfolge in der
// Transaktion) steht in repository/ einmal; eine Abfrage im Handler daneben kennt sie nicht.
//
// Keine Datei von api/ formuliert SQL. Der Bestand dateienOhneTuer nennt, was ohne Tür in api/
// liegt, und kann nur kleiner werden.
//
// Blindheit: SQL, das erst aus Variablen oder Sprintf-Teilen entsteht; Regeln in einer Datei,
// die auch eine Tür trägt; eine Datei, die net/http nur für eine Konstante einbindet.

// Nur Anweisungen, keine Bezeichner: `UPDATE x SET` statt `UPDATE`, sonst schlägt jedes Wort
// "update" in einem Bezeichner an. Hinter dem Tabellennamen steht kein \b: Es verlangte eine
// Wortgrenze nach dem ersten Buchstaben und traf nur einbuchstabige Namen. Welche Formen das
// Muster kennen muss, hält TestSQLAnweisung_ErkenntJedeForm fest.
var sqlAnweisung = regexp.MustCompile(`(?i)\b(` +
	`SELECT\s+[a-z_*(0-9$']` +
	`|INSERT\s+INTO\s+[a-z_]+` +
	`|DELETE\s+FROM\s+[a-z_]+` +
	`|UPDATE\s+(ONLY\s+)?[a-z_.]+(\s+(AS\s+)?[a-z_]+)?\s+SET\b` +
	`|TRUNCATE\s+(TABLE\s+)?[a-z_]+` +
	`|MERGE\s+INTO\s+[a-z_]+` +
	`|LOCK\s+TABLE\s+[a-z_]+)`)

// Kommentare zählen nicht: Ein Satz wie „zwischen SELECT und UPDATE ein Wettlauf-Fenster"
// erklärt eine Abfrage und ist keine.
func ohneKommentare(quelle string) string {
	var b strings.Builder
	for line := range strings.Lines(quelle) {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func sqlAnweisungenIn(quelle string) int {
	return len(sqlAnweisung.FindAllStringIndex(ohneKommentare(quelle), -1))
}

// produktivDateien nennt die Go-Dateien dieses Ordners ohne Tests. Die Untergrenze hält einen
// umbenannten Ordner davon ab, beide Tests still grün zu stellen.
func produktivDateien(t *testing.T) []string {
	t.Helper()
	eintraege, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("api/ nicht lesbar: %v", err)
	}
	var namen []string
	for _, e := range eintraege {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		namen = append(namen, name)
	}
	if len(namen) < 50 {
		t.Fatalf("nur %d Go-Dateien in api/ gefunden — der Test misst offenbar nichts mehr", len(namen))
	}
	return namen
}

func TestHandlerFormulierenKeinSQL(t *testing.T) {
	for _, name := range produktivDateien(t) {
		quelle, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("%s nicht lesbar: %v", name, err)
		}
		if n := sqlAnweisungenIn(string(quelle)); n > 0 {
			t.Errorf("api/%s formuliert SQL (%d Anweisungen). Handler lesen und schreiben über "+
				"repository/ — dort steht jede Regel (COALESCE-Schutz, NULL-Behandlung, Reihenfolge "+
				"in der Transaktion) einmal. Die Anweisung gehört in eine repository-Funktion.", name, n)
		}
	}
}

// Der Zähler findet die Anweisungen dort, wo sie stehen. Fände er in repository/ nichts, wäre
// „kein SQL in api/" keine Aussage.
func TestSQLZaehler_FindetAnweisungenInRepository(t *testing.T) {
	const ordner = "../repository"
	eintraege, err := os.ReadDir(ordner)
	if err != nil {
		t.Fatalf("%s nicht lesbar: %v", ordner, err)
	}
	summe := 0
	for _, e := range eintraege {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		quelle, err := os.ReadFile(filepath.Join(ordner, name))
		if err != nil {
			t.Fatalf("%s nicht lesbar: %v", name, err)
		}
		summe += sqlAnweisungenIn(string(quelle))
	}
	if summe < 300 {
		t.Fatalf("nur %d SQL-Anweisungen in repository/ gezählt — der Zähler misst offenbar nichts mehr", summe)
	}
}

// Das Muster erkennt jede Form, in der ein Handler eine Anweisung schreiben kann. Eine Form,
// die es nicht kennt, ließe einen neuen Handler mit genau dieser Anweisung unbemerkt.
func TestSQLAnweisung_ErkenntJedeForm(t *testing.T) {
	anweisungen := []string{
		"SELECT id FROM leser",
		"SELECT * FROM leser",
		"SELECT count(*) FROM leser",
		"select\n\t\tid from leser",
		"SELECT 1 FROM leser WHERE id = $1",
		"SELECT $1::int",
		"SELECT 'fest'",
		"INSERT INTO leser (vorname) VALUES ($1)",
		"DELETE FROM leser WHERE id = $1",
		"UPDATE leser SET vorname = $1",
		"UPDATE public.leser\n\t\tSET vorname = $1",
		"UPDATE ausleihen a SET rueckgabe_am = NOW()",
		"UPDATE ausleihen AS a SET rueckgabe_am = NOW()",
		"UPDATE ONLY leser SET vorname = $1",
		"TRUNCATE leser",
		"TRUNCATE TABLE leser",
		"MERGE INTO leser l USING neu n ON l.id = n.id",
		"LOCK TABLE leser IN EXCLUSIVE MODE",
	}
	for _, a := range anweisungen {
		if !sqlAnweisung.MatchString(a) {
			t.Errorf("das Muster erkennt die Anweisung nicht: %q", a)
		}
	}
	// Bezeichner und Wörter, die wie der Anfang einer Anweisung aussehen.
	keine := []string{
		"updateSettings(ctx)",
		"selectListe := []string{}",
		"insertInto(ziel)",
		"deleteFromCart()",
		`aktion == "UPDATE"`,
		`meldung := "Update fehlgeschlagen"`,
		"truncated := true",
	}
	for _, k := range keine {
		if sqlAnweisung.MatchString(k) {
			t.Errorf("das Muster hält für eine Anweisung, was keine ist: %q", k)
		}
	}
}

// Der Zähler zählt jede Anweisung einzeln und lässt einen Kommentar aus.
func TestSQLAnweisungenIn_ZaehltJedeAnweisung(t *testing.T) {
	faelle := []struct {
		name   string
		quelle string
		soll   int
	}{
		{"keine", "x := 1\n", 0},
		{"eine", "q := `SELECT id FROM leser`\n", 1},
		{"zwei auf einer Zeile", "a, b := `SELECT 1 FROM leser`, `DELETE FROM leser`\n", 2},
		{"drei über Zeilen", "`INSERT INTO leser (a) VALUES ($1)`\n`UPDATE leser\n SET a = $1`\n`SELECT a FROM leser`\n", 3},
		{"Unterabfrage zählt mit", "`DELETE FROM leser WHERE id IN (SELECT id FROM alt)`\n", 2},
		{"nur im Kommentar", "// erst SELECT id FROM leser, dann UPDATE leser SET a = 1\nx := 1\n", 0},
		{"Kommentar hinter Code", "q := `SELECT id FROM leser` // und kein DELETE FROM leser\n", 1},
	}
	for _, f := range faelle {
		if ist := sqlAnweisungenIn(f.quelle); ist != f.soll {
			t.Errorf("%s: %d Anweisungen gezählt, erwartet %d", f.name, ist, f.soll)
		}
	}
}

// dateienOhneTuer: Dateien in api/, die net/http nicht einbinden. Sie nehmen keine Anfrage an
// und geben keine Antwort; was davon Regel oder Erzeuger ist, zieht in ein eigenes Paket und
// fällt hier weg.
var dateienOhneTuer = []string{
	"abgaenger_fenster.go",
	"action_types.go",
	"bescheid_absender.go",
	"bestellbestaetigung_token.go",
	"bestellmail_anhaenge.go",
	"bestellmail_text.go",
	"bestellmail_versand.go",
	"betriebsbereitschaft_alarm.go",
	"constants.go",
	"lmf_plan_live.go",
	"lmf_plan_vorgabe.go",
	"lmf_termine_frist.go",
	"mahnwesen_mail.go",
	"order_service.go",
	"schueler_kiosk.go",
	"student_klasse_regel.go",
	"verwaltung_protokoll.go",
}

// bindetHTTPEin sagt, ob die Quelle net/http einbindet, unter welchem Namen auch immer.
func bindetHTTPEin(t *testing.T, name string, quelle any) bool {
	t.Helper()
	datei, err := parser.ParseFile(token.NewFileSet(), name, quelle, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("%s nicht lesbar: %v", name, err)
	}
	for _, imp := range datei.Imports {
		if pfad, err := strconv.Unquote(imp.Path.Value); err == nil && pfad == "net/http" {
			return true
		}
	}
	return false
}

func TestNeueDateiInApiTraegtEineTuer(t *testing.T) {
	var gefunden []string
	for _, name := range produktivDateien(t) {
		if !bindetHTTPEin(t, name, nil) {
			gefunden = append(gefunden, name)
		}
	}
	for _, name := range gefunden {
		if !slices.Contains(dateienOhneTuer, name) {
			t.Errorf("api/%s trägt keine Tür (bindet net/http nicht ein). Regeln, Leser fremder "+
				"Dateien und Erzeuger von Dokumenten gehören in ein eigenes Paket unter internal/, "+
				"pkg/ oder pdf/; api/ ruft sie auf. Gehört die Datei zu den Türen (Typen einer "+
				"Anfrage, Helfer mehrerer Handler), trage sie in dateienOhneTuer ein.", name)
		}
	}
	for _, name := range dateienOhneTuer {
		if !slices.Contains(gefunden, name) {
			t.Errorf("api/%s liegt nicht mehr ohne Tür in api/ — bitte aus dateienOhneTuer "+
				"entfernen, damit die Ratsche greift.", name)
		}
	}
}

// Der Detektor liest die Einbindungen, nicht den Text: Ein Wort im Kommentar oder ein
// Nachbarpaket darf er nicht für die Tür halten, einen umbenannten Import muss er erkennen.
func TestBindetHTTPEin_ErkenntJedeForm(t *testing.T) {
	mit := map[string]string{
		"einzeln":     "package x\nimport \"net/http\"\n",
		"im Block":    "package x\nimport (\n\t\"fmt\"\n\t\"net/http\"\n)\n",
		"umbenannt":   "package x\nimport web \"net/http\"\n",
		"mit Punkt":   "package x\nimport . \"net/http\"\n",
		"zwei Blöcke": "package x\nimport \"fmt\"\nimport (\n\t\"net/http\"\n)\n",
	}
	for name, quelle := range mit {
		if !bindetHTTPEin(t, name+".go", quelle) {
			t.Errorf("%s: die Einbindung von net/http wird nicht erkannt", name)
		}
	}
	ohne := map[string]string{
		"keine Einbindung": "package x\n",
		"Nachbarpaket":     "package x\nimport (\n\t\"net/http/httptest\"\n\t\"net/url\"\n)\n",
		"nur im Kommentar": "package x\n// braucht net/http nicht\nimport \"fmt\"\n",
		"nur als Text":     "package x\nimport \"fmt\"\nvar s = \"net/http\"\n",
	}
	for name, quelle := range ohne {
		if bindetHTTPEin(t, name+".go", quelle) {
			t.Errorf("%s: gilt als Tür, obwohl net/http nicht eingebunden ist", name)
		}
	}
}
