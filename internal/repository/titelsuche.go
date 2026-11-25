package repository

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"bibliothek/internal/pkg/isbnutil"
)

// Die Titelsuche des Programms ist eine Regel: Jedes Wort der Eingabe muss den Titel in einem
// Feld treffen, Groß- und Kleinschreibung und Umlaute zählen nicht (suchnorm). Zwilling von
// buecherSuchen im Medienkatalog (frontend/src/inventur/lib/startseiten_api.js); beide prüfen
// die Fälle in frontend/src/inventur/lib/titelsuche.faelle.json.

// titelKuerzel nennt zu einem ganzen Suchwort seine Langform: „mathe“ sucht auch „mathematik“.
// Das Wort selbst gilt weiter, wie ein Synonym bei Google („ninja OR assassin“, Programmable
// Search Engine, Rewriting Queries): „Mathe live“ bleibt über „mathe“ zu finden.
var titelKuerzel = map[string]string{
	"powi":  "politik",
	"mathe": "mathematik",
	"eng":   "englisch",
	"deu":   "deutsch",
	"franz": "französisch",
	"bio":   "biologie",
	"che":   "chemie",
	"phy":   "physik",
	"geo":   "geographie",
	"info":  "informatik",
	"lat":   "latein",
	"span":  "spanisch",
	"rel":   "religion",
	"reli":  "religion",
}

// titelFuellwoerter fallen weg, sobald ein Wort eine Zahl ist: „Klasse 7“ sucht den Jahrgang 7.
var titelFuellwoerter = map[string]bool{"klasse": true, "kl": true, "kl.": true, "jahrgang": true, "jg": true, "jg.": true}

// jahrgangWort ist ein Wort aus ein oder zwei Ziffern: ein Jahrgang und kein Stück einer ISBN,
// denn „7“ steht in jeder ISBN-13.
var jahrgangWort = regexp.MustCompile(`^\d{1,2}\.?$`)

// titelSuchwort ist ein Wort der Eingabe ohne Füllwörter.
type titelSuchwort struct {
	wort string
	// langform ist die Langform eines Kürzels (titelKuerzel), sonst leer.
	langform    string
	isbnFormen  []string
	zahl        int64
	hatZahl     bool
	istJahrgang bool
}

// titelSuchwoerter zerlegt eine Eingabe nach der Regel der Titelsuche.
func titelSuchwoerter(eingabe string) []titelSuchwort {
	felder := strings.Fields(strings.ToLower(eingabe))
	hatZahl := false
	for _, f := range felder {
		if _, ok := zahlAmAnfang(f); ok {
			hatZahl = true
		}
	}
	woerter := make([]titelSuchwort, 0, len(felder))
	for _, f := range felder {
		if hatZahl && titelFuellwoerter[f] {
			continue
		}
		w := titelSuchwort{wort: f, langform: titelKuerzel[f], isbnFormen: isbnutil.Formen(f), istJahrgang: jahrgangWort.MatchString(f)}
		w.zahl, w.hatZahl = zahlAmAnfang(f)
		woerter = append(woerter, w)
	}
	return woerter
}

// zahlAmAnfang liest die Zahl am Anfang eines Worts wie Number.parseInt im Browser: ein
// Vorzeichen, dann Ziffern; „10r1“ ist 10, „r10“ keine Zahl. Eine zu lange Zahl trifft keinen
// Jahrgang und bleibt eine Zahl.
func zahlAmAnfang(wort string) (int64, bool) {
	ziffern := strings.TrimLeft(wort, "+-")
	if len(wort)-len(ziffern) > 1 {
		return 0, false
	}
	ende := 0
	for ende < len(ziffern) && ziffern[ende] >= '0' && ziffern[ende] <= '9' {
		ende++
	}
	if ende == 0 {
		return 0, false
	}
	zahl, err := strconv.ParseInt(ziffern[:ende], 10, 64)
	if err != nil {
		zahl = math.MaxInt64
	}
	if strings.HasPrefix(wort, "-") {
		zahl = -zahl
	}
	return zahl, true
}

// sqlLikeMuster macht aus einem SQL-Ausdruck einen wörtlichen LIKE-Teilstring, nach der
// Normalisierung: suchnorm könnte sonst aus einem Zeichen einen Joker machen.
func sqlLikeMuster(ausdruck string) string {
	return `replace(replace(replace(` + ausdruck + `, '\', '\\'), '%', '\%'), '_', '\_')`
}

// sqlTitelTrifft liefert die Bedingung „der Titel mit dem Alias trifft die Eingabe“ samt
// Argumenten ab $start. Ohne Wort ist sie TRUE.
func sqlTitelTrifft(alias, eingabe string, start int) (string, []any) {
	woerter := titelSuchwoerter(eingabe)
	if len(woerter) == 0 {
		return "TRUE", nil
	}
	teile := make([]string, 0, len(woerter))
	args := make([]any, 0, 2*len(woerter))
	for _, w := range woerter {
		platz := fmt.Sprintf("$%d::text", start+len(args))
		args = append(args, w.wort)
		felder := sqlTextfelderEnthalten(alias, sqlLikeMuster("suchnorm("+platz+")"))
		if w.langform != "" {
			langPlatz := fmt.Sprintf("$%d::text", start+len(args))
			args = append(args, w.langform)
			felder = append(felder, sqlTextfelderEnthalten(alias, sqlLikeMuster("suchnorm("+langPlatz+")"))...)
		}
		// Ein Stück der ISBN, Bindestriche zählen auf keiner Seite.
		if !w.istJahrgang && strings.ReplaceAll(w.wort, "-", "") != "" {
			felder = append(felder, "lower(replace(coalesce("+alias+".isbn, ''), '-', '')) LIKE '%' || "+sqlLikeMuster("lower(replace("+platz+", '-', ''))")+" || '%'")
		}
		if len(w.isbnFormen) > 0 {
			felder = append(felder, sqlISBNInFormen(alias, start+len(args)))
			args = append(args, w.isbnFormen)
		}
		if strings.Contains("lernmittel", w.wort) {
			felder = append(felder, alias+".ist_lernmittel")
		}
		if w.hatZahl {
			felder = append(felder, fmt.Sprintf("(coalesce(%[1]s.jahrgang_von, 0) <> 0 AND coalesce(%[1]s.jahrgang_bis, 0) <> 0 AND %[2]d BETWEEN %[1]s.jahrgang_von AND %[1]s.jahrgang_bis)", alias, w.zahl))
		}
		teile = append(teile, "("+strings.Join(felder, "\n\t\t\tOR ")+")")
	}
	bedingung := "(" + strings.Join(teile, "\n\t\tAND ") + ")"
	// Eine ISBN mit Leerzeichen („978 3 16 148410 0“) zerfällt in Wörter; als Ganzes ist sie eine.
	if formen := isbnMitLeerzeichen(eingabe); len(formen) > 0 {
		bedingung = "(" + bedingung + "\n\t\tOR " + sqlISBNInFormen(alias, start+len(args)) + ")"
		args = append(args, formen)
	}
	return bedingung, args
}

// sqlTextfelderEnthalten liefert die Bedingungen „Titel, Autor, Fach, Signatur oder ein
// Schlagwort enthält das Muster“; muster ist ein Suchwort nach suchnorm als LIKE-Teilstring.
func sqlTextfelderEnthalten(alias, muster string) []string {
	return []string{
		"suchnorm(coalesce(" + alias + ".titel, '')) LIKE '%' || " + muster + " || '%'",
		"suchnorm(coalesce(" + alias + ".autor, '')) LIKE '%' || " + muster + " || '%'",
		"suchnorm(coalesce(" + alias + ".subject, '')) LIKE '%' || " + muster + " || '%'",
		"suchnorm(coalesce(" + alias + ".signatur, '')) LIKE '%' || " + muster + " || '%'",
		sqlTitelUeberSchlagwort(alias, muster),
	}
}

// sqlISBNInFormen ist die Bedingung „die ISBN des Titels ist eine der Schreibweisen im
// Parameter $platz“ (isbnutil.Formen); Trenner zählen auf der Seite des Titels nicht.
func sqlISBNInFormen(alias string, platz int) string {
	return fmt.Sprintf("regexp_replace(upper(coalesce(%s.isbn, '')), '[^0-9X]', '', 'g') = ANY($%d::text[])", alias, platz)
}

// isbnWort ist ein Wort aus Ziffern, X und Bindestrichen, ein mögliches Stück einer ISBN.
var isbnWort = regexp.MustCompile(`^[0-9Xx-]+$`)

// isbnMitLeerzeichen liefert die Schreibweisen einer mit Leerzeichen getippten ISBN; leer, wenn
// die Eingabe nur ein Wort hat oder ein Wort kein Stück einer ISBN ist („atlas 3827049156“).
func isbnMitLeerzeichen(eingabe string) []string {
	felder := strings.Fields(eingabe)
	if len(felder) < 2 {
		return nil
	}
	for _, f := range felder {
		if !isbnWort.MatchString(f) {
			return nil
		}
	}
	return isbnutil.Formen(eingabe)
}
