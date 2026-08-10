package inventur

import (
	"context"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// Gate gegen die stille Hälfte der Datenverlust-Bugklasse.
//
// Die Klasse hat zwei Hälften. Die eine — „der Client schickt ein Feld, das Struct nimmt
// es nicht an" — findet jeder Payload-Abgleich. Die andere ist unsichtbar: Eine SPALTE
// existiert, aber kein Schreiber füllt sie. Kein Fehler, kein Statuscode, HTTP 200 — der
// Wert bleibt einfach stehen.
//
// Genau diese Hälfte hat auf diesem Projekt schon zugeschlagen: die fehlende Signatur auf
// dem Etikett, die geleerte Klasse im LUSD-Import, autor/verlag/jahr im Buch-Upsert. Jedes
// Mal fiel es erst auf, als jemand die Daten von Hand ansah.
//
// Der Test vergleicht deshalb die Spalten aus schema.sql mit dem SQL, das UpdateBook
// TATSÄCHLICH absetzt — nicht mit einem Struct und nicht mit dem Quelltext. Wer eine
// Migration schreibt und den Schreiber vergisst, bekommt hier eine Meldung mit dem
// Spaltennamen.

// AUSNAHMEN: Spalten, die UpdateBook bewusst nicht anfasst. Jede braucht einen Schreiber
// woanders oder einen Grund, warum sie keinen hat.
var nichtVonUpdateBook = map[string]string{
	"id":            "Schlüssel der WHERE-Klausel, nicht der SET-Liste",
	"erstellt_am":   "wird beim Anlegen gesetzt und danach nie wieder",
	"sort_order":    "manuelle Reihenfolge des Admins, gesetzt in reorder_handler.go (Ziehen und Ablegen)",
	"search_vector": "GENERATED ALWAYS — Postgres pflegt die Spalte, ein Schreibversuch wäre ein Fehler",
	"cover_status":  "gehört der asynchronen Cover-Beschaffung (internal/service/cover_service.go)",
	"ziel_jahrgang": "wird über die Bestandsaufnahme gesetzt (repository/book_inventory.go)",
	"meldebestand": "Altbestand: Die Spalte wird NUR gelesen und von keinem Codepfad geschrieben. " +
		"Die Bestellschwelle kommt seit dem Umbau aus den Einstellungen, der Wert wird laut " +
		"api/reorders.go nur noch informativ mitgeliefert.",
}

// spaltenAusSchema liest die Spalten einer Tabelle aus schema.sql — aus dem CREATE TABLE
// UND aus späteren ALTER TABLE … ADD COLUMN. Ohne den zweiten Teil fehlten hier
// jahrgang_von und jahrgang_bis, und der Test hätte zwei Spalten nie geprüft.
func spaltenAusSchema(t *testing.T, tabelle string) []string {
	t.Helper()

	roh, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatalf("schema.sql nicht lesbar: %v", err)
	}
	sql := string(roh)

	start := strings.Index(sql, "CREATE TABLE "+tabelle+" (")
	if start < 0 {
		t.Fatalf("CREATE TABLE %s steht nicht in schema.sql", tabelle)
	}
	ende := strings.Index(sql[start:], "\n);")
	if ende < 0 {
		t.Fatalf("Ende der Definition von %s nicht gefunden", tabelle)
	}
	block := sql[start : start+ende]

	spalten := map[string]bool{}
	// Eine Spaltenzeile beginnt mit vier Leerzeichen und einem Bezeichner. Constraints
	// (CONSTRAINT, PRIMARY, UNIQUE, CHECK, FOREIGN) sind ausgenommen.
	zeile := regexp.MustCompile(`(?m)^\s{4}([a-z_]+)\s`)
	for _, m := range zeile.FindAllStringSubmatch(block, -1) {
		name := m[1]
		switch name {
		case "constraint", "primary", "unique", "check", "foreign", "exclude":
			continue
		}
		spalten[name] = true
	}

	alter := regexp.MustCompile(`ALTER TABLE ` + tabelle + `\b[\s\S]*?;`)
	addCol := regexp.MustCompile(`ADD COLUMN(?: IF NOT EXISTS)? ([a-z_]+)`)
	for _, block := range alter.FindAllString(sql, -1) {
		for _, m := range addCol.FindAllStringSubmatch(block, -1) {
			spalten[m[1]] = true
		}
	}

	out := make([]string, 0, len(spalten))
	for s := range spalten {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// sqlVonUpdateBook fängt die Anweisung ab, die UpdateBook wirklich absetzt.
//
// Bewusst über den Query-Matcher und nicht durch Lesen der Go-Datei: Ein Test, der
// Quelltext liest, prüft, was jemand geschrieben hat. Dieser prüft, was läuft.
func sqlVonUpdateBook(t *testing.T) string {
	t.Helper()

	var erfasst string
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(
		pgxmock.QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
			erfasst = actualSQL
			return nil
		}),
	))
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	// AnyArg für jedes Argument: Hier interessiert die ANWEISUNG, nicht was hineinfliesst.
	// Ohne diese Zeile erwartet pgxmock null Argumente und bricht mit "expected 0, but got
	// 19 arguments" ab — der Test waere rot, ohne etwas ueber das Schema zu sagen.
	beliebig := make([]any, 19)
	for i := range beliebig {
		beliebig[i] = pgxmock.AnyArg()
	}
	mock.ExpectExec("").WithArgs(beliebig...).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewBookRepository(mock)
	if err := repo.UpdateBook(context.Background(), "irgendeine-id", Book{}); err != nil {
		t.Fatalf("UpdateBook: %v", err)
	}
	if erfasst == "" {
		t.Fatal("kein SQL abgefangen — der Query-Matcher hat nicht gegriffen")
	}
	return erfasst
}

func TestUpdateBookSchreibtJedeSpalte(t *testing.T) {
	spalten := spaltenAusSchema(t, "buecher_titel")
	anweisung := sqlVonUpdateBook(t)

	// Nur die SET-Liste betrachten: Das WHERE nennt id, und ohne diese Eingrenzung
	// gälte die Spalte als geschrieben.
	setTeil := anweisung
	if i := strings.Index(anweisung, "WHERE"); i > 0 {
		setTeil = anweisung[:i]
	}

	geschrieben := func(spalte string) bool {
		return regexp.MustCompile(`(?m)[\s,(]` + spalte + `\s*=`).MatchString(setTeil)
	}

	// Gegenprobe gegen einen stillen Nulllauf, in beide Richtungen: Fände der Leser keine
	// Spalten oder erkennte er keine Zuweisung, wäre der Test aus dem falschen Grund grün.
	if len(spalten) < 15 {
		t.Fatalf("nur %d Spalten aus schema.sql gelesen — der Leser ist kaputt: %v", len(spalten), spalten)
	}
	if !geschrieben("titel") {
		t.Fatal("„titel" + `" wird als ungeschrieben gemeldet — die Erkennung der SET-Liste ist kaputt`)
	}
	if geschrieben("gibtesnicht") {
		t.Fatal("eine erfundene Spalte gilt als geschrieben — die Erkennung ist zu großzügig")
	}

	var fehlend []string
	for _, spalte := range spalten {
		if geschrieben(spalte) {
			continue
		}
		if _, bekannt := nichtVonUpdateBook[spalte]; bekannt {
			continue
		}
		fehlend = append(fehlend, spalte)
	}

	if len(fehlend) > 0 {
		t.Errorf(
			"UpdateBook schreibt %d Spalte(n) von buecher_titel nicht: %s\n"+
				"Eine Spalte ohne Schreiber verliert stillschweigend jede Eingabe — HTTP 200, kein Fehler,\n"+
				"der alte Wert bleibt stehen. Entweder in die SET-Liste aufnehmen oder in\n"+
				"nichtVonUpdateBook eintragen, MIT dem Schreiber, der stattdessen zuständig ist.",
			len(fehlend), strings.Join(fehlend, ", "))
	}

	// Die Ausnahmeliste darf nicht verwildern: Wer eine Spalte entfernt, soll auch ihre
	// Ausnahme entfernen. Sonst steht dort irgendwann eine Begründung für nichts.
	vorhanden := map[string]bool{}
	for _, s := range spalten {
		vorhanden[s] = true
	}
	for spalte := range nichtVonUpdateBook {
		if !vorhanden[spalte] {
			t.Errorf("Ausnahme für %q, aber die Spalte gibt es in buecher_titel nicht mehr", spalte)
		}
	}
}
