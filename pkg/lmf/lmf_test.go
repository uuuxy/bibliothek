package lmf

import "testing"

func TestIstTitel(t *testing.T) {
	faelle := []struct {
		titel string
		want  bool
	}{
		{"lmf-Deutsch 5", true},
		{"LMF-Deutsch 5", true},
		{"LMF - Deutsch 5", true},  // Leerzeichen um den Bindestrich (der gemeldete Bug)
		{"LMF Deutsch 5", true},    // nur Leerzeichen als Trenner
		{"  lmf-Mathe", true},      // führender Whitespace
		{"Der kleine Hobbit", false},
		{"LMFP-Roman", false},      // kein Trenner nach lmf
		{"lmfao Witzebuch", false}, // kein Trenner nach lmf
		{"", false},
		{"lmf", false}, // Kürzel allein ohne Trenner/Rest
	}

	for _, f := range faelle {
		if got := IstTitel(f.titel); got != f.want {
			t.Errorf("IstTitel(%q) = %v, want %v", f.titel, got, f.want)
		}
	}
}

func TestSQLBedingung(t *testing.T) {
	got := SQLBedingung("t.titel")
	want := "LOWER(t.titel) ~ '^lmf[ -]'"
	if got != want {
		t.Errorf("SQLBedingung = %q, want %q", got, want)
	}
}
