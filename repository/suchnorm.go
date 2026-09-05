package repository

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Suchnorm ist der Go-Zwilling der SQL-Funktion suchnorm(text) aus Migration 054 —
// die EINE Normalform für Namensvergleiche in diesem Projekt. Zwei Schritte wie dort:
//
//  1. Kleinschreibung, dann Diakritika weg (unaccent): Öztürk → ozturk, García → garcia,
//     Nguyễn → nguyen, Łukasz → lukasz, Straße → strasse.
//  2. Die deutschen Ersatzschreibungen auf denselben Nenner: ss→s, ue→u, oe→o, ae→a —
//     in DIESER Reihenfolge, so treffen sich „Mueller" und „Müller" (beide → muller).
//
// Warum ein Zwilling und nicht ein SELECT suchnorm($1): Der LUSD-Import bildet seinen
// Namensschlüssel aus geparsten CSV-Zeilen UND aus geladenen Bestandszeilen in Go
// (LusdNamensSchluessel); bis zum 05.09.2026 war das nur lower+trim, und „Anna Müller"
// von Hand angelegt wurde vom Export „Anna Mueller" nie gefunden — Neuanlage plus
// „nicht im Export" statt Zuordnung. Ein Zwilling verpflichtet zur Parität:
// suchnorm_pg_test.go vergleicht beide Seiten über einen Namenskorpus, damit die
// Schülersuche (SQL) und der Importschlüssel (Go) nie still verschiedene Menschen meinen.
//
// unaccent kennt mehr als die Unicode-Zerlegung: ß, ø, ł, æ, œ, đ, ı, þ, ð haben keine
// abtrennbaren Akzente und stehen deshalb als Sonderfälle vor der Zerlegung.
func Suchnorm(s string) string {
	s = strings.ToLower(s)
	s = unaccentSonderfaelle.Replace(s)
	// transform.String scheitert nur an ungültigem UTF-8; dann bleibt der Name unzerlegt
	// (kleingeschrieben, Sonderfälle ersetzt) — der Paritätstest sähe eine Abweichung.
	if zerlegt, _, err := transform.String(unaccentZerlegung, s); err == nil {
		s = zerlegt
	}
	// Sequenziell wie replace(replace(replace(replace(…)))) in SQL — NewReplacer wäre ein
	// einziger Durchlauf mit anderer Überlappungsregel („sss").
	s = strings.ReplaceAll(s, "ss", "s")
	s = strings.ReplaceAll(s, "ue", "u")
	s = strings.ReplaceAll(s, "oe", "o")
	s = strings.ReplaceAll(s, "ae", "a")
	return s
}

// unaccentZerlegung zerlegt in Grundbuchstabe + Akzent (NFD), wirft die Akzente
// (Kategorie Mn) weg und setzt wieder zusammen.
var unaccentZerlegung = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// unaccentSonderfaelle sind die Buchstaben, die unaccent.rules kennt, NFD aber nicht.
var unaccentSonderfaelle = strings.NewReplacer(
	"ß", "ss", "ø", "o", "ł", "l", "æ", "ae", "œ", "oe", "đ", "d", "ı", "i", "þ", "th", "ð", "d",
)
