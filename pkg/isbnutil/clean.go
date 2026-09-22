package isbnutil

import (
	"regexp"
	"strings"
)

var isbnForm = regexp.MustCompile(`^([0-9]{9}[0-9X]|[0-9]{13})$`)

// Normalform ist die EINE Schreibweise einer ISBN — der Go-Spiegel der SQL-Funktion
// isbn_normalform() (Migration 133), mit der die Datenbank jede geschriebene ISBN an
// jeder Tür in dieselbe Form bringt. Wer eine ISBN VERGLEICHT, vergleicht Normalformen;
// die Parität beider Seiten prüft repository/isbn_normalform_pg_test.go.
//
// Regel: Bindestriche und Leerzeichen weg, Prüfzeichen X groß — aber nur, wenn das
// Ergebnis eine ISBN ist (10 oder 13 Zeichen aus Ziffern und X). Was keine ISBN ist,
// bleibt wie geschrieben (getrimmt); was nur aus Trennern besteht, wird leer.
func Normalform(roh string) string {
	ohne := strings.ToUpper(CleanISBN(roh))
	switch {
	case ohne == "":
		return ""
	case isbnForm.MatchString(ohne):
		return ohne
	default:
		return strings.TrimSpace(roh)
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
