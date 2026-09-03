package inventur

import (
	"testing"
)

func TestParseKlassenStufe(t *testing.T) {
	tests := []struct {
		name     string
		gradeStr string
		title    string
		expected int16
	}{
		{"Valid grade string inside bounds", "7", "Any Title", 7},
		{"Valid grade string lower bound", "5", "Any Title", 5},
		{"Valid grade string Oberstufe 11", "11", "Any Title", 11},
		{"Valid grade string upper bound 13 (Abitur)", "13", "Any Title", 13},
		{"Valid grade string below bounds", "4", "Any Title", 5},
		{"Valid grade string above bounds (>13)", "14", "Any Title", 5},
		{"Invalid grade string, title has valid grade", "abc", "Math 8", 8},
		{"Invalid grade string, title has no grade", "abc", "Random Title", 5},
		{"Invalid grade string, title grade below bounds", "abc", "Level 4", 5},
		{"Invalid grade string, title grade Oberstufe 12", "abc", "Level 12", 12},
		{"Grade string is zero, title has valid grade", "0", "English 9", 9},
		{"Grade string is zero, title has no grade", "0", "Another Title", 5},
		{"Grade string is empty, title has valid grade", "", "Science 6", 6},
		{"Grade string is empty, title has no grade", "", "Science", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseKlassenStufe(tt.gradeStr, tt.title)
			if result != tt.expected {
				t.Errorf("parseKlassenStufe(%q, %q) = %d; want %d", tt.gradeStr, tt.title, result, tt.expected)
			}
		})
	}
}

func TestParseBestand(t *testing.T) {
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBestand(tt.input)
			if result != tt.expected {
				t.Errorf("parseBestand(%q) = %d; want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// Die Fach-Spalte des Excel-Imports ist Freitext, buecher_titel.subject aber ein
// Fremdschlüssel auf die Systematik: Jede eigenständige Zeichenkette wird dort eine
// eigene Kategorie und steht danach in der Fach-Auswahl der Buchmaske und als Fach im
// Portal-Reiter Schulbücher. Bis zum 03.09.2026 ging der Wert ungeprüft durch, und ein
// leeres Feld wurde zum wörtlichen Fach „Unbekannt" — dieselbe Form, die der
// CSV-Bestandsimport an diesem Tag verloren hat (fachDerZeile, Commit 21b8e172), nur
// durch die zweite Tür. Ein Lauf hätte gereicht, um scripts/repair_fach_kategorie.sql
// wieder aufzuheben.
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
		cfg := ImportConfig{
			Row:    []string{"9783161484100", "Ein Titel", "Ein Autor", f.spalte},
			ColIdx: map[string]int{"isbn": 0, "titel": 1, "autor": 2, "fach": 3},
		}
		buch, err := verarbeiteImportZeile(cfg)
		if err != nil || buch == nil {
			t.Fatalf("Spalte %q: %v", f.spalte, err)
		}
		if buch.Subject != f.want {
			t.Errorf("Fach-Spalte %q → Subject %q, erwartet %q", f.spalte, buch.Subject, f.want)
		}
	}
}
