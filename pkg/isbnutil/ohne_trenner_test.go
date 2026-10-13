package isbnutil

import "testing"

func TestOhneTrenner(t *testing.T) {
	faelle := []struct {
		name string
		in   string
		want string
	}{
		{"unverändert ohne Trenner", "9783161484100", "9783161484100"},
		{"Bindestriche", "978-3-16-148410-0", "9783161484100"},
		{"Leerzeichen", "978 3 16 148410 0", "9783161484100"},
		{"gemischt", "978-3 16-148410 0", "9783161484100"},
		{"führende/folgende Leerzeichen", " 9783161484100 ", "9783161484100"},
		{"leer", "", ""},
		{"nur Trenner", "- -", ""},
		{"ISBN-10 mit X", "3-16-148410-X", "316148410X"},
		// Ein Zeichen aus mehreren Bytes enthält nie das Byte eines Bindestrichs oder
		// Leerzeichens; Byte für Byte zu lesen zerschneidet es nicht.
		{"Mehrbyte bleibt intakt", "978-3-16-ü", "978316ü"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := OhneTrenner(f.in); got != f.want {
				t.Errorf("OhneTrenner(%q) = %q, erwartet %q", f.in, got, f.want)
			}
		})
	}
}

// Enthält die Eingabe keinen Trenner, kommt dieselbe Zeichenkette zurück.
func TestOhneTrenner_OhneTrennerKeineKopie(t *testing.T) {
	in := "9783161484100"
	if got := OhneTrenner(in); got != in {
		t.Fatalf("OhneTrenner(%q) = %q", in, got)
	}
}
