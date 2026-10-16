package inventur

import (
	"testing"
)

func TestLeseKlassenstufe(t *testing.T) {
	faelle := []struct {
		name     string
		gradeStr string
		expected int16
	}{
		{"Valid grade string inside bounds", "7", 7},
		{"Valid grade string lower bound", "5", 5},
		{"Valid grade string Oberstufe 11", "11", 11},
		{"Valid grade string upper bound 13 (Abitur)", "13", 13},
		{"Valid grade string below bounds", "4", 0},
		{"Valid grade string above bounds (>13)", "14", 0},
		{"Invalid grade string", "abc", 0},
		{"Grade string is zero", "0", 0},
		{"Grade string is empty", "", 0},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			ergebnis := leseKlassenstufe(f.gradeStr)
			if ergebnis != f.expected {
				t.Errorf("leseKlassenstufe(%q) = %d; want %d", f.gradeStr, ergebnis, f.expected)
			}
		})
	}
}

func TestLeseBestand(t *testing.T) {
	faelle := []struct {
		name     string
		input    string
		expected int
	}{
		{"Normaler Bestand", "12", 12},
		{"Null", "0", 0},
		{"Leerer String", "", 0},
		{"Kein Zahlwert", "abc", 0},
		{"Negativ ist Datenfehler", "-3", 0},
		{"Obergrenze int32 noch erlaubt", "2147483647", 2147483647},
		{"Über int32 würde beim Bulk-Upsert überlaufen", "2147483648", 0},
		{"Absurd großer Wert", "5000000000", 0},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			ergebnis := leseBestand(f.input)
			if ergebnis != f.expected {
				t.Errorf("leseBestand(%q) = %d; want %d", f.input, ergebnis, f.expected)
			}
		})
	}
}

// Die Spalte des Fachs im Import aus Excel ist Freitext, buecher_titel.subject aber ein
// Fremdschlüssel auf die Systematik: Jede eigenständige Zeichenkette wird dort eine eigene
// Kategorie und steht danach in der Auswahl der Fächer in der Maske und als Fach im Reiter
// Schulbücher des Portals. Ein ungeprüfter Wert oder ein leeres Feld als wörtliches Fach
// „Unbekannt" wären dieselbe Form wie im Import des Bestands aus CSV (fachDerZeile), nur durch die
// zweite Tür.
func TestVerarbeiteImportZeile_FachNurWennFach(t *testing.T) {
	faelle := []struct{ spalte, want string }{
		{"Mathematik", "Mathematik"},
		{"Mathe", "Mathematik"},  // bekannte Schreibvariante wird kanonisch
		{"biologie", "Biologie"}, // Kleinschreibung ebenso
		{"Buch Pg/Kaf 078829", ""},
		{"Regal 3 links", ""},
		{"Unbekannt", ""},
		{"", ""},
	}
	for _, f := range faelle {
		cfg := importZeile{
			Zeile:     []string{"9783161484100", "Ein Titel", "Ein Autor", f.spalte},
			Spalten:   map[string]int{"isbn": 0, "titel": 1, "autor": 2, "fach": 3},
			Metadaten: offlineMetadatenClient(),
		}
		buch, err := verarbeiteImportZeile(t.Context(), cfg)
		if err != nil || buch == nil {
			t.Fatalf("Spalte %q: %v", f.spalte, err)
		}
		if buch.Fach != f.want {
			t.Errorf("Fach-Spalte %q → Subject %q, erwartet %q", f.spalte, buch.Fach, f.want)
		}
	}
}
