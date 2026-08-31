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
		{"LMFP-Roman", "", false},      // kein Trenner nach lmf
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
	// btrim seit dem 31.08.2026 auf BEIDEN Spalten — sonst erkennt Go ein Buch mit
	// führendem Leerzeichen als Lernmittel und das SQL nicht (siehe Paarungstest unten).
	want := "(LOWER(btrim(t.titel, E' \\t\\n\\r')) ~ '^lmf[ -]' OR " +
		"LOWER(btrim(COALESCE(t.signatur, ''), E' \\t\\n\\r')) ~ '^lmf[ -]')"
	if got != want {
		t.Errorf("SQLBedingung = %q, want %q", got, want)
	}
}

// Go und SQL müssen DASSELBE Buch als Lernmittel erkennen.
//
// Fund des Komplett-Durchgangs 31.08.2026: IstSchulbuch trimmt (strings.TrimSpace),
// SQLBedingung nicht (`LOWER(col) ~ '^lmf[ -]'`). Ein Titel oder eine Signatur mit
// führendem Leerzeichen — Import, Copy-Paste aus Excel — war damit für Go ein
// Lernmittel (Ausleihlimit, Schuljahresfrist) und für repository.OeffentlichSichtbar
// keins: Genau dieses Buch erschien im öffentlichen Katalog und als „Buch des Monats"
// auf dem Flurbildschirm — der Zustand, den die Vereinheitlichung vom 30.08. beseitigen
// sollte. Der Bestandstest hält den Go-Fall („  lmf-Mathe" = true) ausdrücklich fest.
func TestSQLBedingung_ErkenntDasselbeWieIstSchulbuch(t *testing.T) {
	faelle := []struct {
		titel, signatur string
	}{
		{"  lmf-Mathe", ""},
		{"", "  LMF Deu 7"},
		{"\tLMF - Bio", ""},
		{"LMF-Mathe 5", ""},
		{"Der Hobbit", "Jug Tol"},
		{" Der Hobbit", " Jug Tol"},
		{"LMFP-Roman", ""},
	}
	for _, f := range faelle {
		if got := IstSchulbuch(f.titel, f.signatur); got != sqlErkennt(t, f.titel, f.signatur) {
			t.Errorf("Titel=%q Signatur=%q: Go sagt %v, SQL sagt %v — dasselbe Buch, zwei Antworten",
				f.titel, f.signatur, got, !got)
		}
	}
}
