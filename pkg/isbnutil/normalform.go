// Package isbnutil bringt eine ISBN in ihre Normalform: ohne Trenner und, wo das Prüfzeichen
// stimmt, dreizehnstellig. Die Datenbank rechnet dieselbe Form (isbn_normalform).
package isbnutil

import (
	"regexp"
	"strings"
)

var isbnForm = regexp.MustCompile(`^([0-9]{9}[0-9X]|[0-9]{13})$`)

// Normalform liefert die eine Schreibweise einer ISBN, wie die SQL-Funktion isbn_normalform
// sie rechnet. Wer zwei ISBN vergleicht, vergleicht Normalformen; beide Seiten hält
// repository/isbn_normalform_pg_test.go gleich.
//
// Bindestriche und Leerzeichen fallen weg, das Prüfzeichen X steht groß, und eine zehnstellige
// ISBN mit richtigem Prüfzeichen wird dreizehnstellig. Mit falschem Prüfzeichen bleibt sie
// zehnstellig, weil die Rechnung auf die ISBN eines anderen Buchs führte. Was keine ISBN ist,
// bleibt wie geschrieben; was nur aus Trennern besteht, wird leer.
func Normalform(roh string) string {
	ohne := strings.ToUpper(OhneTrenner(roh))
	switch {
	case ohne == "":
		return ""
	case !isbnForm.MatchString(ohne):
		return strings.TrimSpace(roh)
	case len(ohne) == 10 && pruefzeichen10(ohne[:9]) == ohne[9:]:
		zwoelf := "978" + ohne[:9]
		return zwoelf + pruefziffer13(zwoelf)
	default:
		return ohne
	}
}

// OhneTrenner liefert die ISBN ohne Bindestriche und Leerzeichen. Eine Eingabe ohne Trenner
// kommt unverändert zurück, ohne neue Zeichenkette.
func OhneTrenner(isbn string) string {
	if !strings.ContainsAny(isbn, "- ") {
		return isbn
	}

	b := make([]byte, 0, len(isbn))
	for i := 0; i < len(isbn); i++ {
		if isbn[i] != '-' && isbn[i] != ' ' {
			b = append(b, isbn[i])
		}
	}
	return string(b)
}
