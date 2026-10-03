package isbnutil

import (
	"regexp"
	"strings"
)

var isbnForm = regexp.MustCompile(`^([0-9]{9}[0-9X]|[0-9]{13})$`)

// Normalform ist die eine Schreibweise einer ISBN — der Go-Spiegel der SQL-Funktion
// isbn_normalform() (Migration 133 und 157), mit der die Datenbank jede geschriebene ISBN an
// jeder Tür in dieselbe Form bringt. Wer eine ISBN vergleicht, vergleicht Normalformen; die
// Parität beider Seiten prüft repository/isbn_normalform_pg_test.go.
//
// Regel: Bindestriche und Leerzeichen weg, Prüfzeichen X groß, und eine Länge — eine
// zehnstellige ISBN mit richtigem Prüfzeichen wird zur dreizehnstelligen (978, die neun
// Ziffern, neu berechnete Prüfziffer). Eine zehnstellige mit falschem Prüfzeichen bleibt
// zehnstellig: Von ihr führte die Rechnung auf die ISBN eines anderen Buchs. Was keine ISBN
// ist, bleibt wie geschrieben (getrimmt); was nur aus Trennern besteht, wird leer.
func Normalform(roh string) string {
	ohne := strings.ToUpper(CleanISBN(roh))
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

// CleanISBN removes hyphens and spaces from an ISBN string.
// ⚡ Bolt: High-performance string cleaning using a single pass to avoid multiple strings.ReplaceAll allocations.
func CleanISBN(isbn string) string {
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
