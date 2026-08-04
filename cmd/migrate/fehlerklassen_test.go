package main

import (
	"errors"
	"testing"
)

// TestExitCodeSagtDieWahrheit: der Rückgabewert ist die Kurzfassung des Berichts. Ein
// unvollständiger Lauf darf niemals als Erfolg enden — auch dann nicht, wenn die
// Zählungen unauffällig aussehen und nur der Abgleich widerspricht.
func TestExitCodeSagtDieWahrheit(t *testing.T) {
	faelle := []struct {
		name     string
		bericht  abschlussbericht
		erwartet int
	}{
		{"sauber", abschlussbericht{AbgleichOK: true}, exitOK},
		{"nur Warnungen", abschlussbericht{Warnungen: 12, AbgleichOK: true}, exitOK},
		{"übersprungene Titel", abschlussbericht{Uebersprungen: 1, Fehler: 1, AbgleichOK: true}, exitUnvollstaendig},
		{"Abgleich widerspricht", abschlussbericht{AbgleichOK: false}, exitUnvollstaendig},
		{"abgebrochen", abschlussbericht{Abbruch: errors.New("Verbindung weg"), AbgleichOK: true}, exitAbgebrochen},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := f.bericht.exitCode(); got != f.erwartet {
				t.Errorf("exitCode() = %d, erwartet %d", got, f.erwartet)
			}
		})
	}
}
