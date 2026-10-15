package main

import (
	"errors"
	"testing"
)

// Der Rückgabewert ist die Kurzfassung des Berichts. Ein unvollständiger Lauf darf niemals als
// Erfolg enden, auch dann nicht, wenn die Zählungen unauffällig aussehen und nur der Abgleich
// widerspricht.
func TestRueckgabewertSagtDieWahrheit(t *testing.T) {
	faelle := []struct {
		name     string
		bericht  abschlussbericht
		erwartet int
	}{
		{"sauber", abschlussbericht{AbgleichOK: true}, rueckgabeOK},
		{"nur Warnungen", abschlussbericht{Warnungen: 12, AbgleichOK: true}, rueckgabeOK},
		{"übersprungene Titel", abschlussbericht{Uebersprungen: 1, Fehler: 1, AbgleichOK: true}, rueckgabeUnvollstaendig},
		{"Abgleich widerspricht", abschlussbericht{AbgleichOK: false}, rueckgabeUnvollstaendig},
		{"abgebrochen", abschlussbericht{Abbruch: errors.New("Verbindung weg"), AbgleichOK: true}, rueckgabeAbgebrochen},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := f.bericht.rueckgabewert(); got != f.erwartet {
				t.Errorf("exitCode() = %d, erwartet %d", got, f.erwartet)
			}
		})
	}
}
