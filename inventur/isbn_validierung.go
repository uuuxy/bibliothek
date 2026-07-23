package inventur

import (
	"regexp"
)

// isbnRegex prüft auf gültige ISBN-10 oder ISBN-13 Zeichenfolgen.
var isbnRegex = regexp.MustCompile(`^[0-9]{9,13}[0-9xX]?$`)

// cleanISBNChars removes spaces, hyphens, and whitespace efficiently without multiple string allocations.
func cleanISBNChars(s string) string {
	needsCleaning := false
	keepCount := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' || c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			needsCleaning = true
		} else {
			keepCount++
		}
	}
	if !needsCleaning {
		return s
	}
	buf := make([]byte, keepCount)
	idx := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '-' && c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			buf[idx] = c
			idx++
		}
	}
	return string(buf)
}

// validiereISBN prüft, ob die ISBN ein gültiges Format hat.
// Akzeptiert ISBN-10 und ISBN-13 (mit und ohne Bindestriche/Leerzeichen).
func validiereISBN(isbn string) bool {
	sauber := cleanISBNChars(isbn)

	if len(sauber) < 10 || len(sauber) > 13 {
		return false
	}

	return isbnRegex.MatchString(sauber)
}
