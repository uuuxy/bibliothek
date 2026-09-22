package isbnutil

import "testing"

// Die Fälle sind dieselben wie in repository/isbn_normalform_pg_test.go — dort läuft die
// SQL-Seite und die Parität, hier die reine Go-Regel.
func TestNormalform(t *testing.T) {
	faelle := []struct{ in, want string }{
		{"978-3-16-148410-0", "9783161484100"},
		{" 978 3 16 148410 0 ", "9783161484100"},
		{"3-16-148410-x", "316148410X"},
		{"9783161484100", "9783161484100"},
		{"ISBN-0000000001", "ISBN-0000000001"},         // keine ISBN: bleibt
		{"3-12-345678-9 kart.", "3-12-345678-9 kart."}, // Beiwerk: bleibt
		{" - ", ""},
		{"", ""},
	}
	for _, f := range faelle {
		if got := Normalform(f.in); got != f.want {
			t.Errorf("Normalform(%q) = %q, erwartet %q", f.in, got, f.want)
		}
	}
}
