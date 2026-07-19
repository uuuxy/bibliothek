package inventur

import (
	"bibliothek/pkg/isbnutil"
	"regexp"
	"strings"
)

// isbnRegex prüft auf gültige ISBN-10 oder ISBN-13 Zeichenfolgen.
var isbnRegex = regexp.MustCompile(`^[0-9]{9,13}[0-9xX]?$`)

// validiereISBN prüft, ob die ISBN ein gültiges Format hat.
// Akzeptiert ISBN-10 und ISBN-13 (mit und ohne Bindestriche/Leerzeichen).
func validiereISBN(isbn string) bool {
	sauber := isbnutil.Clean(isbn)
	// removed duplicate space cleaning handled by isbnutil.Clean
	sauber = strings.TrimSpace(sauber)

	if len(sauber) < 10 || len(sauber) > 13 {
		return false
	}

	return isbnRegex.MatchString(sauber)
}
