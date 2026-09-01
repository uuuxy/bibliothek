package isbnutil

import "testing"

func TestCleanISBN(t *testing.T) {
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
		// Byte-weise Iteration ist UTF-8-sicher: Mehrbyte-Sequenzen enthalten nie
		// ASCII-Bytes, '-' und ' ' können also nicht versehentlich in einem Zeichen treffen.
		{"Mehrbyte bleibt intakt", "978-3-16-ü", "978316ü"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := CleanISBN(f.in); got != f.want {
				t.Errorf("CleanISBN(%q) = %q, erwartet %q", f.in, got, f.want)
			}
		})
	}
}

// TestCleanISBN_OhneTrennerKeineKopie hält die Optimierung fest: Enthält die Eingabe
// keinen Trenner, muss exakt derselbe String zurückkommen (kein neuer Allokat).
func TestCleanISBN_OhneTrennerKeineKopie(t *testing.T) {
	in := "9783161484100"
	if got := CleanISBN(in); got != in {
		t.Fatalf("CleanISBN(%q) = %q", in, got)
	}
}
