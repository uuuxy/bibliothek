package repository

import "testing"

// TestAbgangsgrundText: Die Wörter stehen auf dem gedruckten Abgangsbuch. Geprüft werden
// alle Gründe, die chk_aussonderung_grund zulässt, und der leere Grund der Altdaten aus der
// Zeit vor Migration 043. api/abgangsbuch_pg_test.go prüft über die Datenbank nur „Verlust".
func TestAbgangsgrundText(t *testing.T) {
	faelle := []struct {
		grund, erwartet string
	}{
		{"VERLUST", "Verlust"},
		{"BESCHAEDIGUNG", "Beschädigung"},
		{"AUSSORTIERT", "Aussortiert"},
		{"BESTANDSKORREKTUR", "Bestandskorrektur"},
		{"", "ohne Angabe"},
	}

	for _, f := range faelle {
		if got := AbgangsgrundText(f.grund); got != f.erwartet {
			t.Errorf("AbgangsgrundText(%q) = %q, erwartet %q", f.grund, got, f.erwartet)
		}
	}
}
