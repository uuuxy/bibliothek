package pruefung_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// Prüfung: Die Texte des Servers setzen Klammern und Schrägstriche wie die Oberfläche: Meldungen,
// PDFs, Mails, Betriebsbereitschaft, Protokoll und Beschriftungen.
//
// Warum: M3 (Content design, Grammar and punctuation): „Parentheses can be used to define acronyms
// or jargon or when referencing a source. They shouldn’t be used when adding a side note or an
// afterthought of a sentence.“ Googles developer documentation style guide (Slashes): „Don't use
// slashes to separate alternatives.“ Die Regel steht in docs/STIL.md 11; für die Oberfläche hält
// sie frontend-hygiene-satzzeichen.test.js, mit denselben Fällen (nebenzeichen in
// frontend/src/lib/utils/satzzeichen.faelle.json).
//
// Regel: In keiner Zeichenkette unter internal/ steht eine Klammer außer um ein Kürzel oder eine
// Quelle, und kein Schrägstrich zwischen Wörtern. Ein Platzhalter (%s, {{.Datum}}) zählt wie {…}
// in der Oberfläche. Code bleibt außen vor: SQL, reguläre Ausdrücke, Routen, Kopfzeilen und
// Medientypen, Zeitzonen, Rechte in der Form ROLLE/recht, der User-Agent der Katalogdienste und
// Muster, mit denen der Server fremde Daten vergleicht (strings.HasPrefix). Ein Dateipfad ist
// Code wie eine Webadresse und in Klammern eine Quelle; Googles developer documentation style
// guide (Slashes): „Use forward slashes, as appropriate, in computer file paths and URLs.“
//
// Sieht nicht: Text, der erst zur Laufzeit entsteht, Texte aus der Datenbank (die Mail-Vorlagen
// der Schule), Fehlertexte und Log-Zeilen (errors.New, fmt.Errorf, die Methode Error, slog) und
// SQL, das die Erkennung für Text hält, solange es keine Klammer trägt.

var (
	nebenKlammer = regexp.MustCompile(`\(([^()]*)\)`)
	nebenWort    = regexp.MustCompile(`^[\p{L}\d.-]+$`)
	nebenQuelle  = regexp.MustCompile(`^(?:Art\.|§|DIN) `)
	nebenCode    = regexp.MustCompile(`^\.{0,2}/|^data:|https?://|^[^\s\p{Lu}]+$`)
	nebenAdresse = regexp.MustCompile(`[\w-]+(?:\.[\w-]+)*\.[a-z]{2,}/[^\s()]*|(?:[\w-]+/)+[\w-]+\.[a-z]{2,4}\b`)
	nebenZitat   = regexp.MustCompile(`„[^“]*“|"[^"]*"`)
	nebenStrich  = regexp.MustCompile(`[\p{L}}]\s?/\s?[\p{L}{]`)
	nebenCSS     = regexp.MustCompile(`^[a-z-]+: |^[a-z-]+\(|:not\(|^\([a-z-]+(?::| [<>])`)
	nebenLeer    = regexp.MustCompile(`\s+`)
	nebenPlatz   = regexp.MustCompile(`%[-+# 0]*\d*(?:\.\d+)?[a-zA-Z]|\{\{[^}]*\}\}`)
	nebenSQL     = regexp.MustCompile(`\b(?:SELECT|INSERT INTO|DELETE FROM|ON CONFLICT|COALESCE|GREATEST|WHERE|RETURNING|AND|OR|NULL|CASE|END|JOIN|AS|IN|NOT|IS|ANY)\b` +
		`|\$%?\d|::|= '|~ '|'\)|\b(?:coalesce|count|to_char|lower|upper|now|jsonb?_[a-z_]+|nullif|greatest|least|sum|max|min|exists|array_agg|string_agg|date_trunc|extract)\(`)
	// nebenCodeForm sind reguläre Ausdrücke, Werte von Kopfzeilen, Rechte als ROLLE/recht und
	// der User-Agent, mit dem die Katalogdienste anfragen.
	nebenCodeForm  = regexp.MustCompile(`\(\?|\\[dDsSwWb]|^\^|\[\^|=\(|\)=\(|^Mozilla/|^[A-Z_]+/[a-z_]+$`)
	nebenTechnisch = regexp.MustCompile(`^(?:Content-Type|Content-Transfer-Encoding|Content-Disposition|MIME-Version|Subject|From|To|Date|Message-ID|Reply-To): ` +
		`|^[a-z]+/[a-z0-9.+-]+(?:;|$)|^(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) /|^[A-Z][a-z]+/[A-Z][A-Za-z_]+$`)
)

// nebenAufrufe sind Aufrufe, deren Zeichenketten kein Text für einen Menschen sind: Fehler,
// Log, reguläre Ausdrücke, Zeitzonen, SQL und Muster für fremde Daten.
var nebenAufrufe = map[string]bool{
	"New": true, "Errorf": true, "Error": true, "Warn": true, "Info": true, "Debug": true,
	"ErrorContext": true, "WarnContext": true, "InfoContext": true, "DebugContext": true,
	"Printf": true, "Println": true, "Fatal": true, "Fatalf": true,
	"MustCompile": true, "Compile": true, "LoadLocation": true,
	"HasPrefix": true, "TrimPrefix": true, "CutPrefix": true, "HasSuffix": true, "TrimSuffix": true, "CutSuffix": true,
	"Exec": true, "Query": true, "QueryRow": true,
}

// nebenVorlage sind Sätze aus dem Musteranschreiben, das die Schule für den Bescheid vorgelegt hat
// (internal/pdf/bescheid.go, „Wortlaut aus der Vorlage“). Ein Bescheid steht im Wortlaut seiner
// Vorlage; die Regel gilt für Texte, die das Programm selbst formuliert.
var nebenVorlage = map[string]bool{
	"die Schule hat Ihnen/Ihrem Kind im Rahmen der Lernmittelfreiheit Lernmittel zur ": true,
	"Sie haben/Ihr Kind hat die Lernmittel":                                            true,
	"An die Erziehungsberechtigten des/der Schülers/in":                                true,
	"Name des/der":       true,
	"Schülers/Schülerin": true,
}

// nebenzeichen ist der Zwilling von nebenzeichen() in frontend-hygiene-satzzeichen.test.js: eine
// Klammer um mehr als ein Kürzel (zwei Großbuchstaben, kein Leerzeichen) oder eine Quelle
// (Art., §, DIN, eine Webadresse, ein Dateipfad), ein Schrägstrich zwischen Wörtern außer in Code.
func nebenzeichen(roh string) []string {
	t := strings.ReplaceAll(roh, "%%", "%")
	t = strings.TrimSpace(nebenLeer.ReplaceAllString(nebenPlatz.ReplaceAllString(t, "{…}"), " "))
	if nebenCSS.MatchString(t) || nebenSQL.MatchString(t) || nebenTechnisch.MatchString(t) || nebenCodeForm.MatchString(t) {
		return nil
	}
	var funde []string
	for _, m := range nebenKlammer.FindAllStringSubmatch(t, -1) {
		if !istKuerzelOderQuelle(m[1]) && nebenAdresse.ReplaceAllString(m[1], "") != "" {
			funde = append(funde, "Klammer ("+m[1]+")")
		}
	}
	ohneZitat := nebenAdresse.ReplaceAllString(nebenZitat.ReplaceAllString(t, "„…“"), "…")
	if !nebenCode.MatchString(ohneZitat) && nebenStrich.MatchString(ohneZitat) {
		funde = append(funde, "Schrägstrich")
	}
	return funde
}

func istKuerzelOderQuelle(s string) bool {
	if nebenQuelle.MatchString(s) {
		return true
	}
	if !nebenWort.MatchString(s) {
		return false
	}
	gross := 0
	for _, r := range s {
		if unicode.IsUpper(r) {
			gross++
		}
	}
	return gross >= 2
}

// serverDateien liefert die Dateien unter internal/ ohne Tests und ohne die Hilfen, die nur Tests
// einbinden (Testdaten, Datenbank-, PDF- und SMTP-Prüfhilfe).
func serverDateien(t *testing.T) []string {
	t.Helper()
	ohne := map[string]bool{"internal/testdaten": true, "internal/pdftest": true, "internal/smtptest": true, "internal/pgtest": true}
	var dateien []string
	err := filepath.WalkDir("internal", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && ohne[pfad] {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(pfad, ".go") && !strings.HasSuffix(pfad, "_test.go") {
			dateien = append(dateien, pfad)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("filepath.WalkDir(internal): %v", err)
	}
	// Nicht-leer-Garantie: Zieht der Code um, prüfte die Regel sonst still nichts.
	if len(dateien) < 400 {
		t.Fatalf("%d Dateien unter internal/, erwartet mindestens 400", len(dateien))
	}
	return dateien
}

// texteDerDatei liefert die Zeichenketten einer Datei mit Zeile, ohne Importe, Struct-Tags und
// die Argumente von nebenAufrufe.
func texteDerDatei(t *testing.T, fset *token.FileSet, pfad string) map[*ast.BasicLit]string {
	t.Helper()
	datei, err := parser.ParseFile(fset, pfad, nil, 0)
	if err != nil {
		t.Fatalf("parser.ParseFile(%s): %v", pfad, err)
	}
	aus := map[*ast.BasicLit]bool{}
	alleAus := func(n ast.Node) {
		ast.Inspect(n, func(m ast.Node) bool {
			if b, ok := m.(*ast.BasicLit); ok {
				aus[b] = true
			}
			return true
		})
	}
	ast.Inspect(datei, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			// Die Rückgabe einer Methode Error ist ein Fehlertext (STIL.md 11).
			if x.Recv != nil && x.Name.Name == "Error" && x.Body != nil {
				alleAus(x.Body)
			}
		case *ast.ImportSpec:
			aus[x.Path] = true
		case *ast.Field:
			if x.Tag != nil {
				aus[x.Tag] = true
			}
		case *ast.CallExpr:
			name := ""
			switch fn := x.Fun.(type) {
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			case *ast.Ident:
				name = fn.Name
			}
			if nebenAufrufe[name] {
				alleAus(x)
			}
		}
		return true
	})
	texte := map[*ast.BasicLit]string{}
	ast.Inspect(datei, func(n ast.Node) bool {
		if b, ok := n.(*ast.BasicLit); ok && b.Kind == token.STRING && !aus[b] {
			if s, err := strconv.Unquote(b.Value); err == nil {
				texte[b] = s
			}
		}
		return true
	})
	return texte
}

func TestNebenzeichen_TexteDesServers(t *testing.T) {
	wechsleInDieWurzel(t)
	fset := token.NewFileSet()
	var funde []string
	geprueft := 0
	for _, pfad := range serverDateien(t) {
		for lit, text := range texteDerDatei(t, fset, pfad) {
			geprueft++
			if pfad == "internal/pdf/bescheid.go" && nebenVorlage[text] {
				continue
			}
			for _, f := range nebenzeichen(text) {
				funde = append(funde, pfad+":"+strconv.Itoa(fset.Position(lit.Pos()).Line)+": "+f+" in "+strconv.Quote(text))
			}
		}
	}
	if geprueft < 5000 {
		t.Fatalf("nur %d Zeichenketten geprüft, erwartet mindestens 5000", geprueft)
	}
	for _, f := range funde {
		t.Error(f)
	}
}

func TestNebenzeichen_GoUndJavaScriptTeilenDieFaelle(t *testing.T) {
	geteilt, _ := leseSatzzeichen(t, "..")
	if len(geteilt.Nebenzeichen.Ohne) < 5 || len(geteilt.Nebenzeichen.Mit) < 5 {
		t.Fatalf("nebenzeichen: %d Fälle ohne und %d mit Fund, erwartet je mindestens 5",
			len(geteilt.Nebenzeichen.Ohne), len(geteilt.Nebenzeichen.Mit))
	}
	for _, text := range geteilt.Nebenzeichen.Ohne {
		if got := nebenzeichen(text); len(got) != 0 {
			t.Errorf("nebenzeichen(%q) = %q, erwartet keinen Fund", text, got)
		}
	}
	for _, text := range geteilt.Nebenzeichen.Mit {
		if got := nebenzeichen(text); len(got) == 0 {
			t.Errorf("nebenzeichen(%q) ohne Fund, erwartet einen", text)
		}
	}
}

// Die Formen, die es nur im Go-Code gibt: Platzhalter von fmt und text/template, SQL, Routen,
// Kopfzeilen, Medientypen und Zeitzonen.
func TestNebenzeichen_Formen(t *testing.T) {
	for _, text := range []string{
		"Summe (%s)", "Schüler/in: %s", "Buchbestellung {{.Datum}} (Kundennummer {{.Kundennummer}})",
		"Rabatt von 5 %% (%s)",
	} {
		if len(nebenzeichen(text)) == 0 {
			t.Errorf("nebenzeichen(%q) ohne Fund, erwartet einen", text)
		}
	}
	for _, text := range []string{
		"Postausgang (SMTP) für %s", "SELECT coalesce(a, '') FROM b WHERE (c = $1)",
		"GET /api/bestand/zugangsbuch", "Content-Type: text/plain; charset=utf-8",
		"text/plain; charset=utf-8", "Europe/Berlin",
	} {
		if got := nebenzeichen(text); len(got) != 0 {
			t.Errorf("nebenzeichen(%q) = %q, erwartet keinen Fund", text, got)
		}
	}
}
