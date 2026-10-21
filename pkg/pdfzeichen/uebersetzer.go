// Package pdfzeichen bringt Text in die Form, die gofpdf drucken kann, und kürzt ihn auf die
// Breite einer Zelle.
//
// gofpdf zeichnet die Standardschriften in cp1252. Was diese Kodierung nicht kennt, macht ihr
// Übersetzer (UnicodeTranslatorFromDescriptor) zu einem Punkt: „Ayşe" käme als „Ay.e" aus dem
// Drucker, „Łukasz" als „.ukasz". Ein Etikett, ein Brief oder ein Bescheid mit entstelltem
// Namen ist schlimmer als einer ohne Häkchen unter dem s; deshalb wird vorher ersetzt.
//
// Jeder Übersetzer eines Dokuments läuft durch Uebersetzer; pdfzeichen_ratsche_test.go im
// Wurzelpaket hält es.
package pdfzeichen

import "strings"

// ersatz bildet Buchstaben ab, die cp1252 nicht kennt. Abgedeckt sind die Sprachen, die an
// einer hessischen Schule vorkommen: Türkisch, Polnisch, Rumänisch, Tschechisch, Slowakisch,
// Kroatisch, Serbisch, Ungarisch, Litauisch und Lettisch. Was cp1252 kennt (ä, ö, ü, ß, é, à,
// â, î, ç, ñ, ø, æ …), steht nicht darin, weil es richtig druckt.
var ersatz = strings.NewReplacer(
	"ş", "s", "Ş", "S", "ğ", "g", "Ğ", "G", "ı", "i", "İ", "I",
	"ć", "c", "Ć", "C", "č", "c", "Č", "C", "ł", "l", "Ł", "L",
	"ń", "n", "Ń", "N", "ś", "s", "Ś", "S", "ż", "z", "Ż", "Z", "ź", "z", "Ź", "Z",
	"ą", "a", "Ą", "A", "ę", "e", "Ę", "E", "ő", "o", "Ő", "O", "ű", "u", "Ű", "U",
	"ș", "s", "Ș", "S", "ț", "t", "Ț", "T", "ă", "a", "Ă", "A",
	"ř", "r", "Ř", "R", "ě", "e", "Ě", "E", "ů", "u", "Ů", "U",
	"ľ", "l", "Ľ", "L", "ť", "t", "Ť", "T", "ď", "d", "Ď", "D", "ň", "n", "Ň", "N", "ŕ", "r", "Ŕ", "R",
	"š", "s", "Š", "S", "ž", "z", "Ž", "Z", "đ", "d", "Đ", "D",
	"ā", "a", "Ā", "A", "ē", "e", "Ē", "E", "ī", "i", "Ī", "I", "ū", "u", "Ū", "U",
	"ų", "u", "Ų", "U", "ė", "e", "Ė", "E", "į", "i", "Į", "I",
)

// Uebersetzer setzt die Ersetzung vor den cp1252-Übersetzer eines gofpdf-Dokuments (dessen
// UnicodeTranslatorFromDescriptor). Durch das Ergebnis läuft jede Zeichenkette, bevor sie aufs
// Papier geht.
//
// Die Funktion nimmt den Übersetzer und nicht das Dokument; so kommt das Paket ohne gofpdf aus.
func Uebersetzer(cp1252 func(string) string) func(string) string {
	return func(text string) string { return cp1252(ersatz.Replace(text)) }
}
