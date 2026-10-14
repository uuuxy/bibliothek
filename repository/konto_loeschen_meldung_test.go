package repository

import (
	"strings"
	"testing"
)

// ErrKontoHatAusleihen erscheint in der Verwaltung der Konten als Meldung (409 beim Löschen,
// api/benutzer/konto_loeschen.go). Sie beschreibt den Zustand so, wie ihn die Bibliothek
// kennt, mit ausgeliehenen Büchern und ohne ein Wort aus dem Code.
func TestErrKontoHatAusleihen_OhneWortAusDemCode(t *testing.T) {
	meldung := ErrKontoHatAusleihen.Error()
	if strings.Contains(meldung, "Handapparat") {
		t.Errorf("die Meldung enthält „Handapparat“: %q", meldung)
	}
	if !strings.Contains(meldung, "ausgeliehen") {
		t.Errorf("die Meldung sagt nicht, dass noch Bücher ausgeliehen sind: %q", meldung)
	}
}
