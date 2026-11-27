package repository

import (
	"slices"
	"testing"
)

// Die Vorschläge stehen wie ein deutsches Register, wie die Schlagwortsuche der Datenbank
// (COLLATE deutsch) und die Listen der Oberfläche: Umlaute bei ihrem Grundbuchstaben, Zahlen
// nach ihrer Größe. Doppelte fallen ohne Rücksicht auf Groß- und Kleinschreibung weg.
func TestAlphabetischOhneDoppelte_Deutsch(t *testing.T) {
	ist := alphabetischOhneDoppelte([]string{"Ökologie", "Zoologie", "Band 10", "Ägypten", "Biologie", "Band 2", "ökologie"})
	erwartet := []string{"Ägypten", "Band 2", "Band 10", "Biologie", "Ökologie", "Zoologie"}
	if !slices.Equal(ist, erwartet) {
		t.Errorf("alphabetischOhneDoppelte = %q, erwartet %q", ist, erwartet)
	}
}
