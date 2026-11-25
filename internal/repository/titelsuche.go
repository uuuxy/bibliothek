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

// titelKuerzel ersetzt ein ganzes Suchwort: „mathe“ sucht „mathematik“.
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

// titelSuchwort ist ein Wort der Eingabe nach Kürzel und Füllwörtern.
type titelSuchwort struct {
	wort        string
	isbnFormen  []string
	zahl        int64
	hatZahl     bool
	istJahrgang bool
}

// titelSuchwoerter zerlegt eine Eingabe nach der Regel der Titelsuche.
func titelSuchwoerter(eingabe string) []titelSuchwort {
	felder := strings.Fields(strings.ToLower(eingabe))
	hatZahl := false
	for i, f := range felder {
		if ersatz, ok := titelKuerzel[f]; ok {
			felder[i] = ersatz
		}
		if _, ok := zahlAmAnfang(felder[i]); ok {
			hatZahl = true
		}
	}
	woerter := make([]titelSuchwort, 0, len(felder))
	for _, f := range felder {
		if hatZahl && titelFuellwoerter[f] {
			continue
		}
		w := titelSuchwort{wort: f, isbnFormen: isbnutil.Formen(f), istJahrgang: jahrgangWort.MatchString(f)}
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
		norm := sqlLikeMuster("suchnorm(" + platz + ")")
		felder := []string{
			"suchnorm(coalesce(" + alias + ".titel, '')) LIKE '%' || " + norm + " || '%'",
			"suchnorm(coalesce(" + alias + ".autor, '')) LIKE '%' || " + norm + " || '%'",
			"suchnorm(coalesce(" + alias + ".subject, '')) LIKE '%' || " + norm + " || '%'",
			"suchnorm(coalesce(" + alias + ".signatur, '')) LIKE '%' || " + norm + " || '%'",
			alias + ".id IN (SELECT tsw.titel_id FROM " + sqlWortZumTitel + " WHERE suchnorm(sw.wort) LIKE '%' || " + norm + " || '%')",
		}
		// Ein Stück der ISBN, Bindestriche zählen auf keiner Seite.
		if !w.istJahrgang && strings.ReplaceAll(w.wort, "-", "") != "" {
			felder = append(felder, "lower(replace(coalesce("+alias+".isbn, ''), '-', '')) LIKE '%' || "+sqlLikeMuster("lower(replace("+platz+", '-', ''))")+" || '%'")
		}
		if len(w.isbnFormen) > 0 {
			felder = append(felder, fmt.Sprintf("regexp_replace(upper(coalesce(%s.isbn, '')), '[^0-9X]', '', 'g') = ANY($%d::text[])", alias, start+len(args)))
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
	return "(" + strings.Join(teile, "\n\t\tAND ") + ")", args
}
