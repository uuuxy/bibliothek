package repository

import (
	"strings"
	"testing"
)

// ErrUserHasActiveLoans erscheint in der Benutzerverwaltung als Meldung (409 beim Löschen,
// api/user_admin_loeschen.go). Sie beschreibt den Zustand so, wie ihn die Bibliothek kennt —
// ausgeliehene Bücher —, nicht mit einem Wort aus dem Code. Bis zum 16.09.2026 hieß sie
// „Benutzer hat noch aktive Handapparat-Ausleihen".
func TestErrUserHasActiveLoans_OhneWortAusDemCode(t *testing.T) {
	meldung := ErrUserHasActiveLoans.Error()
	if strings.Contains(meldung, "Handapparat") {
		t.Errorf("die Meldung enthält „Handapparat“: %q", meldung)
	}
	if !strings.Contains(meldung, "ausgeliehen") {
		t.Errorf("die Meldung sagt nicht, dass noch Bücher ausgeliehen sind: %q", meldung)
	}
}
