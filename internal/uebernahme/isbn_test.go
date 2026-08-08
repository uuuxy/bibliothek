package uebernahme

import (
	"testing"
)

func TestNormalisiereISBN(t *testing.T) {
	faelle := []struct {
		name     string
		eingabe  string
		erwartet string
	}{
		{"ISBN-13 mit Bindestrichen", "978-3-16-148410-0", "9783161484100"},
		{"ISBN-13 mit Leerzeichen", "978 3 16 148410 0", "9783161484100"},
		{"ISBN-10 mit großem X", "3-598-21500-X", "359821500X"},
		{"ISBN-10 mit kleinem x", "3-598-21500-x", "359821500X"},
		{"unerwartete Sonderzeichen", "978-3-16!?-148410-0", "9783161484100"},
		{"leere Eingabe", "", ""},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := NormalisiereISBN(f.eingabe); got != f.erwartet {
				t.Errorf("NormalisiereISBN(%q) = %q, erwartet %q", f.eingabe, got, f.erwartet)
			}
		})
	}
}
