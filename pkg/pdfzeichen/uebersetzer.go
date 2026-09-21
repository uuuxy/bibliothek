// Package pdfzeichen ist die eine Tür für die Zeichenersetzung in den PDFs.
//
// gofpdf zeichnet die Standardschriften in cp1252. Was diese Kodierung nicht kennt,
// wird von ihrem Übersetzer (UnicodeTranslatorFromDescriptor) zu einem Punkt: „Ayşe" kam
// als „Ay.e" aus dem Drucker, „Łukasz" als „.ukasz". Ein Etikett, ein Brief oder ein
// Bescheid mit entstelltem Namen ist schlimmer als einer ohne Häkchen unter dem s —
// deshalb wird vorher ersetzt statt hinterher verstümmelt.
//
// Bis zum 21.09.2026 stand die Ersetzung nur an zwei von siebzehn Stellen (Schüler-
// Etikett, Bescheid); die Buchetiketten, die Mahnbriefe und alle anderen Renderer
// kannten sie nicht. Jetzt läuft jeder Übersetzer durch Uebersetzer, und eine Ratsche
// (pdfzeichen_ratsche_test.go im Wurzelpaket) hält es dabei.
package pdfzeichen

import "strings"

// ersatz bildet Buchstaben ab, die cp1252 nicht kennt. Abgedeckt sind die, die an einer
// hessischen Schule tatsächlich vorkommen: türkisch, polnisch, rumänisch, tschechisch/
// slowakisch, kroatisch/serbisch, ungarisch, litauisch/lettisch. Was cp1252 kennt (ä, ö,
// ü, ß, é, à, â, î, ç, ñ, ø, æ …), steht hier bewusst NICHT — das druckt richtig.
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

// Uebersetzer nimmt den cp1252-Übersetzer eines gofpdf-Dokuments (dessen
// UnicodeTranslatorFromDescriptor) und setzt die Ersetzung davor. Das Ergebnis ist die
// Funktion, durch die jede Zeichenkette läuft, bevor sie aufs Papier geht.
//
// Die Signatur nimmt die Funktion, nicht das Dokument: Im Haus sind zwei gofpdf-
// Fassungen im Einsatz (jung-kurt und phpdave11), und beide liefern denselben Typ.
func Uebersetzer(cp1252 func(string) string) func(string) string {
	return func(text string) string { return cp1252(ersatz.Replace(text)) }
}
