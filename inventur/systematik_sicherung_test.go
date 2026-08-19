package inventur

import (
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// erwarteFachBekannt spielt den Nachschlag der Fach-Registrierung nach: Das Fach ist
// bereits in systematik_kategorien registriert, der Schreibpfad bekommt die kanonische
// Bezeichnung zurück. Jeder pgxmock-Test eines subject-Schreibers braucht diese
// Erwartung VOR seinem eigentlichen Statement (siehe StelleFaecherSicher).
func erwarteFachBekannt(mock pgxmock.PgxPoolIface, fach string) {
	mock.ExpectQuery(`SELECT bezeichnung FROM systematik_kategorien WHERE lower\(bezeichnung\) = lower\(\$1\)`).
		WithArgs(fach).
		WillReturnRows(pgxmock.NewRows([]string{"bezeichnung"}).AddRow(fach))
}

// TestKuerzeRunen sichert die Zeichen-Semantik der Kürzel-Kappung: VARCHAR zählt
// Zeichen, nicht Bytes — ein mittig zerschnittener Umlaut wäre zudem kein gültiges Kürzel.
func TestKuerzeRunen(t *testing.T) {
	faelle := []struct {
		eingabe  string
		max      int
		erwartet string
	}{
		{"Deutsch", 50, "Deutsch"},
		{"Gesellschaftswissenschaften", 10, "Gesellscha"},
		{"ÄÖÜäöüß", 3, "ÄÖÜ"},
		{"", 5, ""},
	}
	for _, f := range faelle {
		if got := kuerzeRunen(f.eingabe, f.max); got != f.erwartet {
			t.Errorf("kuerzeRunen(%q, %d) = %q, erwartet %q", f.eingabe, f.max, got, f.erwartet)
		}
	}
}
