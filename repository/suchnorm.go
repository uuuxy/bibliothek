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
// suchnorm_pg_test.go vergleicht beide Seiten, damit die Schülersuche (SQL) und der
// Importschlüssel (Go) nie still verschiedene Menschen meinen.
//
// Wie „unaccent" nachgebaut ist: je Zeichen zuerst die Ausnahmetabelle (unaccentTabelle),
// sonst die Unicode-Zerlegung — Grundbuchstabe plus Akzent, Akzent weg. Die Tabelle ist
// aus der Datenbank ERZEUGT, nicht ausgedacht: Für jedes Zeichen der lateinischen Blöcke
// (Latin-1 Supplement, Extended-A/B, Extended Additional) steht darin, was
// unaccent(lower(x)) liefert, wo das von der Zerlegung abweicht — in beide Richtungen.
// Buchstaben ohne abtrennbaren Akzent, die unaccent trotzdem kennt (ħ→h, ŋ→n, ǆ→dz,
// ß→ss), und Zeichen, die unaccent NICHT kennt und stehen lässt, obwohl sie zerlegbar
// wären (ǣ, ǿ). Die SQL-Seite ist die Referenz samt ihrer Lücken — auf ihr stehen die
// Indizes. Der erste Zwilling (nur neun Sonderfälle von Hand) lag bei 110 Zeichen daneben;
// das fiel erst dem Zeichen-Durchlauf auf, nicht dem Namenskorpus.
//
// TestSuchnorm_ZeichenDurchlaufGoUndSQL prüft jedes Zeichen dieser Blöcke gegen die
// Datenbank. Ändert ein Postgres-Upgrade die unaccent-Regeln, wird er rot, und die
// Tabelle wird aus seiner Ausgabe nachgezogen.
func Suchnorm(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if ersatz, ok := unaccentTabelle[r]; ok {
			b.WriteString(ersatz)
			continue
		}
		b.WriteString(zerlegeRune(r))
	}
	s = b.String()
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

// zerlegeRune wendet die Zerlegung auf ein einzelnes Zeichen an. transform.String
// scheitert nur an ungültigem UTF-8; dann bleibt das Zeichen, wie es ist.
func zerlegeRune(r rune) string {
	z, _, err := transform.String(unaccentZerlegung, string(r))
	if err != nil {
		return string(r)
	}
	return z
}

// unaccentTabelle: erzeugt am 05.09.2026 gegen PostgreSQL 18.6 (unaccent-Regeln der
// Erweiterung) über U+00C0–U+024F und U+1E00–U+1EFF. Schlüssel ist das Zeichen NACH
// strings.ToLower, Wert die Ausgabe von unaccent(lower(x)). Nur die Abweichungen von
// der Zerlegung stehen hier; alles andere (ä, é, ç, ñ, ș …) ergibt sich aus ihr.
var unaccentTabelle = map[rune]string{
	'×': "*",  // U+00D7
	'ß': "ss", // U+00DF
	'æ': "ae", // U+00E6
	'ð': "d",  // U+00F0
	'÷': "/",  // U+00F7
	'ø': "o",  // U+00F8
	'þ': "th", // U+00FE
	'đ': "d",  // U+0111
	'ħ': "h",  // U+0127
	'ı': "i",  // U+0131
	'ĳ': "ij", // U+0133
	'ĸ': "q",  // U+0138
	'ŀ': "l",  // U+0140
	'ł': "l",  // U+0142
	'ŉ': "'n", // U+0149
	'ŋ': "n",  // U+014B
	'œ': "oe", // U+0153
	'ŧ': "t",  // U+0167
	'ſ': "s",  // U+017F
	'ƀ': "b",  // U+0180
	'ƃ': "b",  // U+0183
	'ƈ': "c",  // U+0188
	'ƌ': "d",  // U+018C
	'ƒ': "f",  // U+0192
	'ƕ': "hv", // U+0195
	'ƙ': "k",  // U+0199
	'ƚ': "l",  // U+019A
	'ƞ': "n",  // U+019E
	'ƣ': "oi", // U+01A3
	'ƥ': "p",  // U+01A5
	'ƫ': "t",  // U+01AB
	'ƭ': "t",  // U+01AD
	'ƴ': "y",  // U+01B4
	'ƶ': "z",  // U+01B6
	'ǆ': "dz", // U+01C6
	'ǉ': "lj", // U+01C9
	'ǌ': "nj", // U+01CC
	'ǣ': "ǣ",  // U+01E3
	'ǥ': "g",  // U+01E5
	'ǯ': "ǯ",  // U+01EF
	'ǳ': "dz", // U+01F3
	'ǽ': "ǽ",  // U+01FD
	'ǿ': "ǿ",  // U+01FF
	'ȡ': "d",  // U+0221
	'ȥ': "z",  // U+0225
	'ȴ': "l",  // U+0234
	'ȵ': "n",  // U+0235
	'ȶ': "t",  // U+0236
	'ȷ': "j",  // U+0237
	'ȸ': "db", // U+0238
	'ȹ': "qp", // U+0239
	'ȼ': "c",  // U+023C
	'ȿ': "s",  // U+023F
	'ɀ': "z",  // U+0240
	'ɇ': "e",  // U+0247
	'ɉ': "j",  // U+0249
	'ɍ': "r",  // U+024D
	'ɏ': "y",  // U+024F
	'ɓ': "b",  // U+0253
	'ɖ': "d",  // U+0256
	'ɗ': "d",  // U+0257
	'ɛ': "e",  // U+025B
	'ɠ': "g",  // U+0260
	'ɨ': "i",  // U+0268
	'ɲ': "n",  // U+0272
	'ʀ': "R",  // U+0280
	'ʈ': "t",  // U+0288
	'ʉ': "u",  // U+0289
	'ʋ': "v",  // U+028B
	'ẚ': "a",  // U+1E9A
	'ẛ': "ẛ",  // U+1E9B
	'ẜ': "s",  // U+1E9C
	'ẝ': "s",  // U+1E9D
	'ỻ': "ll", // U+1EFB
	'ỽ': "v",  // U+1EFD
	'ỿ': "y",  // U+1EFF
	'ⱥ': "a",  // U+2C65
	'ⱦ': "t",  // U+2C66
}
