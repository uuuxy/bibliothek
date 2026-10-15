package leser

import (
	"bibliothek/apierrors"
	"errors"
	"strings"
	"testing"
)

// Die Meldung bei Klasse „Lehrer" ist ein Wegweiser: Sie muss den Weg nennen, den es in der
// Oberfläche gibt („Benutzer & Rechte", nicht „Benutzerverwaltung"), und darf keine Wörter
// aus dem Code enthalten (etwa „Handapparat-Weg").
func TestPruefeKlassenname_LehrerMeldungNenntDenEchtenWeg(t *testing.T) {
	for _, klasse := range []string{"Lehrer", " lehrer ", "LEHRER"} {
		var fehler *apierrors.Fehler
		if err := pruefeKlassenname(klasse); !errors.As(err, &fehler) {
			t.Fatalf("pruefeKlassenname(%q) = %v, erwartet die Sperre mit Meldung", klasse, err)
		}
		meldung := fehler.Meldung
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
