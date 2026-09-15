package api

import (
	"strings"
	"testing"
)

// Die Meldung bei Klasse „Lehrer" ist ein Wegweiser: Sie muss den Weg nennen, den es in der
// Oberfläche gibt, und darf keine Wörter aus dem Code enthalten. Bis zum 16.09.2026 schickte
// sie nach „System → Benutzerverwaltung" (der Menüpunkt heißt „Benutzer & Rechte") und
// sprach vom „Handapparat-Weg".
func TestPruefeKlassenname_LehrerMeldungNenntDenEchtenWeg(t *testing.T) {
	for _, klasse := range []string{"Lehrer", " lehrer ", "LEHRER"} {
		err := pruefeKlassenname(klasse)
		if err == nil {
			t.Fatalf("Klasse %q: kein Fehler, erwartet die Sperre", klasse)
		}
		meldung := err.Error()
		if !strings.Contains(meldung, "System → Benutzer & Rechte") {
			t.Errorf("Klasse %q: die Meldung nennt den Menüpunkt nicht: %q", klasse, meldung)
		}
		if !strings.Contains(meldung, "Kollegium") {
			t.Errorf("Klasse %q: die Meldung nennt die Rolle nicht: %q", klasse, meldung)
		}
		for _, internesWort := range []string{"Handapparat", "Benutzerverwaltung"} {
			if strings.Contains(meldung, internesWort) {
				t.Errorf("Klasse %q: die Meldung enthält %q: %q", klasse, internesWort, meldung)
			}
		}
	}
	if err := pruefeKlassenname("05F1"); err != nil {
		t.Errorf("eine gewöhnliche Klasse wird abgelehnt: %v", err)
	}
}
