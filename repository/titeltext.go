package repository

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// TiteltextNormalform ist die Form, in der die Datenbank Titel, Untertitel, Autor und Verlag
// speichert (titeltext_normalform, Migration 160): kein Leerraum am Rand, Leerraum in Folge
// ist ein Leerzeichen, Umlaute und Akzente zusammengesetzt (NFC). Wer einen eingegebenen Text
// mit einem gespeicherten vergleicht, bringt ihn zuerst in diese Form: Ein Suchtext mit zwei
// Leerzeichen träfe den Titel sonst nicht.
func TiteltextNormalform(roh string) string {
	return norm.NFC.String(strings.Join(strings.Fields(roh), " "))
}
