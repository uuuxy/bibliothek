package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Schichtung des Backends: Eine Anweisung an die Datenbank formuliert und schickt nur die
// Datenbankschicht. Jede Regel einer Abfrage (NULL-Behandlung, Schutz vor leeren Werten,
// Reihenfolge in der Transaktion) steht dort einmal; eine Anweisung in einem anderen Paket
// kennt sie nicht.
//
// Zwei Detektoren je Datei: der Text einer Anweisung und der Aufruf, der sie abschickt. Der
// zweite sieht auch eine Anweisung, deren Text erst aus Variablen entsteht.
//
// Blindheit: Den Aufruf erkennt der Detektor am Namen der Methode, nicht am Typ; eine eigene
// Methode namens Exec oder Query mit zwei Argumenten zählt mit. Eine Anweisung, die ein
// fremdes Programm ausführt (psql), sieht keiner der beiden.

// datenbankschicht nennt die Pakete, die Anweisungen formulieren, und warum. Ein Eintrag mit
// Schrägstrich am Ende gilt für alle Pakete darunter.
var datenbankschicht = map[string]string{
	"repository":       "Abfragen und Schreibpfade der Anwendung",
	"inventur":         "eigene Datenbankschicht des Medienkatalogs (ARCHITEKTUR 5.2.3)",
	"db":               "Pool, Migrationen, Rechte-Seed und Übernahmen beim Start; repository/ bindet db/ ein",
	"internal/littera": "Schreiber der einmaligen Übernahme aus Littera",
	"internal/pgtest":  "Prüfhilfe, legt das Schema der Testdatenbank an",
	"scripts":          "Einmal-Skripte außerhalb des Programms",
	"cmd/":             "Einmal-Werkzeuge mit eigener Verbindung",
}

// sqlFunde zählt je Datei die Texte von Anweisungen und die Aufrufe, die eine abschicken.
type sqlFunde struct{ texte, aufrufe int }

// sqlBestand: Dateien außerhalb der Datenbankschicht, die noch Anweisungen tragen. Die Zahlen
// können nur sinken, und eine Datei ohne Fund fällt aus der Liste.
var sqlBestand = map[string]sqlFunde{
	"auth/blacklist.go":                       {texte: 4, aufrufe: 3},
	"auth/handlers.go":                        {texte: 3, aufrufe: 3},
	"auth/jwt.go":                             {texte: 1, aufrufe: 1},
	"auth/selbstanmeldung.go":                 {texte: 2, aufrufe: 2},
	"auth/sitzungen.go":                       {texte: 9, aufrufe: 9},
	"internal/service/ausleih_sperren.go":     {texte: 1, aufrufe: 1},
	"internal/service/cover_service.go":       {texte: 3, aufrufe: 3},
	"internal/service/device_service.go":      {texte: 5, aufrufe: 5},
	"internal/service/import_dynamic.go":      {texte: 4, aufrufe: 4},
	"internal/service/loan_checkout.go":       {texte: 3, aufrufe: 3},
	"internal/service/loan_checkout_cases.go": {texte: 2, aufrufe: 2},
	"internal/service/loan_return.go":         {texte: 2, aufrufe: 2},
	"internal/service/loan_rules.go":          {texte: 1, aufrufe: 1},
	"internal/service/nachbuchen.go":          {texte: 2, aufrufe: 2},
	"internal/service/omnibox_service.go":     {texte: 1, aufrufe: 1},
	"internal/service/order_service.go":       {texte: 6, aufrufe: 4},
	"internal/service/photo_service.go":       {texte: 2, aufrufe: 2},
	"jobs/cron.go":                            {texte: 1, aufrufe: 1},
	"jobs/cron_audit_retention.go":            {texte: 0, aufrufe: 1},
	"jobs/cron_dsgvo.go":                      {texte: 5, aufrufe: 4},
	"jobs/cron_dsgvo_abgaenger.go":            {texte: 1, aufrufe: 1},
	"jobs/cron_dsgvo_anliegen.go":             {texte: 2, aufrufe: 2},
	"jobs/cron_dsgvo_lesehistorie.go":         {texte: 3, aufrufe: 3},
	"jobs/cron_dsgvo_nachbuch.go":             {texte: 1, aufrufe: 1},
	"jobs/cron_dsgvo_papierkorb.go":           {texte: 1, aufrufe: 1},
	"jobs/restore_probe.go":                   {texte: 2, aufrufe: 5},
	"jobs/restore_probe_hilfen.go":            {texte: 1, aufrufe: 1},
	"mailservice/smtp_konfig.go":              {texte: 1, aufrufe: 1},
}

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
	`|LOCK\s+TABLE\s+[a-z_]+` +
	`|CopyFrom\s*\()`)

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

// sendeMethoden nennt die Methoden von pgx, die eine Anweisung abschicken, mit der Zahl der
// Argumente, ab der ein Aufruf zählt: Kontext und Anweisung. So bleibt `r.URL.Query()` außen vor.
var sendeMethoden = map[string]int{
	"Exec":       2,
	"Query":      2,
	"QueryRow":   2,
	"SendBatch":  2,
	"ExecParams": 2,
	"CopyFrom":   4,
}

// datenbankAufrufeIn zählt die Aufrufe einer Datei, die eine Anweisung abschicken; quelle wie
// bei parser.ParseFile.
func datenbankAufrufeIn(pfad string, quelle any) (int, error) {
	datei, err := parser.ParseFile(token.NewFileSet(), pfad, quelle, parser.SkipObjectResolution)
	if err != nil {
		return 0, err
	}
	n := 0
	ast.Inspect(datei, func(knoten ast.Node) bool {
		ruf, ok := knoten.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := ruf.Fun.(*ast.SelectorExpr); ok {
			if ab, bekannt := sendeMethoden[sel.Sel.Name]; bekannt && len(ruf.Args) >= ab {
				n++
			}
		}
		return true
	})
	return n, nil
}

// inDatenbankschicht sagt, unter welchem Eintrag ein Paket Anweisungen formulieren darf.
func inDatenbankschicht(paket string) (string, bool) {
	for eintrag := range datenbankschicht {
		if paket == eintrag || (strings.HasSuffix(eintrag, "/") && strings.HasPrefix(paket, eintrag)) {
			return eintrag, true
		}
	}
	return "", false
}

// sammleSQLFunde liest jede Produktivdatei des Baums und nennt die Dateien mit Fund.
func sammleSQLFunde(t *testing.T) (funde map[string]sqlFunde, dateien int) {
	t.Helper()
	funde = map[string]sqlFunde{}
	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Versteckte Ordner tragen keinen Quelltext des Programms, wohl aber Arbeitskopien.
			if name := d.Name(); name == "node_modules" || name == "frontend" || name == "testdata" ||
				(strings.HasPrefix(name, ".") && pfad != ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		quelle, err := os.ReadFile(filepath.Clean(pfad))
		if err != nil {
			return err
		}
		aufrufe, err := datenbankAufrufeIn(pfad, quelle)
		if err != nil {
			return err
		}
		dateien++
		if f := (sqlFunde{texte: sqlAnweisungenIn(string(quelle)), aufrufe: aufrufe}); f != (sqlFunde{}) {
			funde[filepath.ToSlash(pfad)] = f
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return funde, dateien
}

func TestSchichtung_SQLNurInDerDatenbankschicht(t *testing.T) {
	funde, dateien := sammleSQLFunde(t)

	jeEintrag := map[string]sqlFunde{}
	var meldungen []string
	for pfad, ist := range funde {
		paket := filepath.ToSlash(filepath.Dir(pfad))
		if eintrag, ok := inDatenbankschicht(paket); ok {
			summe := jeEintrag[eintrag]
			jeEintrag[eintrag] = sqlFunde{summe.texte + ist.texte, summe.aufrufe + ist.aufrufe}
			continue
		}
		soll, imBestand := sqlBestand[pfad]
		switch {
		case !imBestand:
			meldungen = append(meldungen, fmt.Sprintf("%s formuliert oder schickt SQL (%d Texte, %d Aufrufe). "+
				"Die Anweisung gehört nach repository/, als Funktion über DBQueryer; das Paket ruft sie "+
				"auf (ARCHITEKTUR 5.2.2).", pfad, ist.texte, ist.aufrufe))
		case ist.texte > soll.texte || ist.aufrufe > soll.aufrufe:
			meldungen = append(meldungen, fmt.Sprintf("%s trägt mehr SQL als im Bestand: %d Texte und %d Aufrufe "+
				"statt %d und %d. Die neue Anweisung gehört nach repository/.",
				pfad, ist.texte, ist.aufrufe, soll.texte, soll.aufrufe))
		case ist != soll:
			meldungen = append(meldungen, fmt.Sprintf("%s trägt weniger SQL als im Bestand — bitte sqlBestand "+
				"auf {%d, %d} absenken, damit die Ratsche greift.", pfad, ist.texte, ist.aufrufe))
		}
	}
	for pfad := range sqlBestand {
		if _, ok := funde[pfad]; !ok {
			meldungen = append(meldungen, fmt.Sprintf("%s trägt kein SQL mehr — bitte aus sqlBestand entfernen.", pfad))
		}
	}
	for eintrag := range datenbankschicht {
		if jeEintrag[eintrag] == (sqlFunde{}) {
			meldungen = append(meldungen, fmt.Sprintf("%s steht in datenbankschicht, trägt aber kein SQL — "+
				"bitte den Eintrag entfernen.", eintrag))
		}
	}

	// Nicht-leer-Garantie: Fänden die Detektoren in repository/ kaum etwas, wäre „kein SQL
	// außerhalb" keine Aussage.
	if r := jeEintrag["repository"]; dateien < 300 || r.texte < 300 || r.aufrufe < 300 {
		t.Fatalf("%d Dateien gelesen, in repository/ %d Texte und %d Aufrufe gezählt — die Detektoren "+
			"messen offenbar nichts mehr", dateien, r.texte, r.aufrufe)
	}
	sort.Strings(meldungen)
	for _, m := range meldungen {
		t.Error(m)
	}
}

// Das Muster erkennt jede Form, in der ein Paket eine Anweisung schreiben kann. Eine Form,
// die es nicht kennt, ließe eine neue Datei mit genau dieser Anweisung unbemerkt.
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
		"tx.CopyFrom(ctx, pgx.Identifier{\"leser\"}, spalten, pgx.CopyFromRows(zeilen))",
		"pool.CopyFrom (ctx, tabelle, spalten, quelle)",
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
		"quelle := pgx.CopyFromRows(zeilen)",
		"kopiereCopyFromDatei(pfad)",
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
		{"Massenkopie", "_, err := tx.CopyFrom(ctx, pgx.Identifier{\"leser\"}, spalten, pgx.CopyFromRows(zeilen))\n", 1},
	}
	for _, f := range faelle {
		if ist := sqlAnweisungenIn(f.quelle); ist != f.soll {
			t.Errorf("%s: %d Anweisungen gezählt, erwartet %d", f.name, ist, f.soll)
		}
	}
}

// Der zweite Detektor erkennt jeden Weg, auf dem pgx eine Anweisung abschickt, auch wenn ihr
// Text aus einer Variablen kommt, und hält keinen gleichnamigen Aufruf ohne Anweisung dafür.
func TestDatenbankAufrufe_ErkenntJedeForm(t *testing.T) {
	faelle := []struct {
		name  string
		rumpf string
		soll  int
	}{
		{"Exec am Pool", "_, _ = pool.Exec(ctx, q)", 1},
		{"Exec an einem Feld", "_, _ = s.db.Exec(ctx, `DELETE FROM `+tabelle+` WHERE `+b.Where, b.Args...)", 1},
		{"QueryRow mit Scan", "_ = tx.QueryRow(ctx, q, id).Scan(&x)", 1},
		{"Query", "rows, _ := q.Query(ctx, text)", 1},
		{"Sammelauftrag", "_ = tx.SendBatch(ctx, batch)", 1},
		{"Massenkopie", "_, _ = tx.CopyFrom(ctx, tabelle, spalten, quelle)", 1},
		{"rohe Verbindung", "_ = conn.ExecParams(ctx, q, nil, nil, nil, nil)", 1},
		{"zwei in einer Funktion", "_, _ = tx.Exec(ctx, a)\n_ = tx.QueryRow(ctx, b).Scan(&x)", 2},
		{"im Funktionsliteral", "f := func() { _, _ = tx.Exec(ctx, a) }\nf()", 1},
		{"Parameter der Adresse", "_ = r.URL.Query()", 0},
		{"ein Argument", "_ = werte.Query(name)", 0},
		{"Zeile lesen", "_ = rows.Scan(&a, &b)", 0},
		{"Funktion der Datenbankschicht", "_, _ = repository.LoescheErledigteAnliegen(ctx, db, b)", 0},
		{"Transaktion öffnen", "tx, _ := pool.Begin(ctx)\n_ = tx.Commit(ctx)", 0},
	}
	for _, f := range faelle {
		quelle := "package x\nfunc f() {\n" + f.rumpf + "\n}\n"
		ist, err := datenbankAufrufeIn(f.name+".go", quelle)
		if err != nil {
			t.Fatalf("%s: Probe nicht lesbar: %v", f.name, err)
		}
		if ist != f.soll {
			t.Errorf("%s: %d Aufrufe gezählt, erwartet %d", f.name, ist, f.soll)
		}
	}
}

// Ein Eintrag mit Schrägstrich gilt für die Pakete darunter, ein Eintrag ohne nur für sich:
// „db" darf kein Paket „db_export" freistellen.
func TestInDatenbankschicht_GrenztPaketeAb(t *testing.T) {
	drin := []string{"repository", "inventur", "db", "internal/littera", "cmd/seed", "cmd/migrate-fotos"}
	for _, p := range drin {
		if _, ok := inDatenbankschicht(p); !ok {
			t.Errorf("%s gilt nicht als Datenbankschicht", p)
		}
	}
	draussen := []string{"api", "jobs", "auth", "internal/service", "internal", "repository_alt", "db_export", "cmd", "internal/littera/neu", "."}
	for _, p := range draussen {
		if eintrag, ok := inDatenbankschicht(p); ok {
			t.Errorf("%s gilt als Datenbankschicht (Eintrag %q)", p, eintrag)
		}
	}
}
