package inventur

import (
	"testing"
)

func TestValidiereISBN(t *testing.T) {
	faelle := []struct {
		name     string
		isbn     string
		expected bool
	}{
		// Valid ISBN-10
		{"Valid ISBN-10", "123456789X", true},
		{"Valid ISBN-10 with hyphens", "1-234-56789-X", true},
		{"Valid ISBN-10 lowercase x", "123456789x", true},
		{"Valid ISBN-10 numeric check digit", "1234567890", true},

		// Valid ISBN-13
		{"Valid ISBN-13", "9781234567890", true},
		{"Valid ISBN-13 with hyphens", "978-1-234-56789-0", true},
		{"Valid ISBN-13 with spaces", "978 1 234 56789 0", true},
		{"Valid ISBN-13 with trailing X (valid per regex)", "978123456789X", true},

		// Invalid formats
		{"Empty string", "", false},
		{"Too short", "123456789", false},
		{"Too long", "97812345678901", false},
		{"Invalid characters", "12345a7890", false},
		{"Only hyphens", "---", false},

		// Grenzfälle mit Leerzeichen.
		{"Leading and trailing spaces", "  123456789X  ", true},
		{"Only spaces", "     ", false},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			ergebnis := validiereISBN(f.isbn)
			if ergebnis != f.expected {
				t.Errorf("validiereISBN(%q) = %v; expected %v", f.isbn, ergebnis, f.expected)
			}
		})
	}
}
