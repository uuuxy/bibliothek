package lmf

import "testing"

func TestIstSchulbuch(t *testing.T) {
	faelle := []struct {
		titel    string
		signatur string
		want     bool
	}{
		{"lmf-Deutsch 5", "", true},
		{"LMF-Deutsch 5", "", true},
		{"LMF - Deutsch 5", "", true}, // Leerzeichen um den Bindestrich (der gemeldete Bug)
		{"LMF Deutsch 5", "", true},   // nur Leerzeichen als Trenner
		{"  lmf-Mathe", "", true},     // führender Whitespace
		{"Der kleine Hobbit", "", false},
		{"LMFP-Roman", "", false},  // kein Trenner nach lmf
		{"lmfao Witzebuch", "", false}, // kein Trenner nach lmf
		{"", "", false},
		{"lmf", "", false}, // Kürzel allein ohne Trenner/Rest

		// Nur die Signatur trägt das Kennzeichen — der Regelfall bei manuell über die
		// Admin-Oberfläche angelegten Schulbüchern (Auto-Vorschlag setzt "LMF <Kürzel>"
		// NUR in die Signatur, der Titel bleibt der Klartext-Buchtitel).
		{"Mathematik Neue Wege 9", "LMF Ma", true},
		{"Mathematik Neue Wege 9", "LMF-Ma", true},
		{"Mathematik Neue Wege 9", "BIB Rom", false},
		{"Mathematik Neue Wege 9", "", false},
	}

	for _, f := range faelle {
		if got := IstSchulbuch(f.titel, f.signatur); got != f.want {
			t.Errorf("IstSchulbuch(%q, %q) = %v, want %v", f.titel, f.signatur, got, f.want)
		}
	}
}

func TestSQLBedingung(t *testing.T) {
	got := SQLBedingung("t.titel", "t.signatur")
	want := "(LOWER(t.titel) ~ '^lmf[ -]' OR LOWER(COALESCE(t.signatur, '')) ~ '^lmf[ -]')"
	if got != want {
		t.Errorf("SQLBedingung = %q, want %q", got, want)
	}
}
