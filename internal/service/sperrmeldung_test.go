package service

import (
	"strings"
	"testing"
)

// Die Sperrmeldung eines GERÄTS darf kein Kind erfinden.
//
// Fund (OFFEN.md 5.9, offen seit 3899cf18): ErrBlocked trug den Satz „ausleihe für
// diese/n Schüler/in ist gesperrt", und der Geräte-Pfad hängte „Gerät ist aktuell
// gesperrt" an. An der Theke stand damit eine Meldung über ein Kind, das an diesem
// Vorgang gar nicht beteiligt ist — die Bibliothekskraft sucht nach einer Sperre, die
// es nicht gibt.
func TestSperrmeldung_NenntKeinKindWennKeinesBeteiligtIst(t *testing.T) {
	if strings.Contains(strings.ToLower(ErrBlocked.Error()), "schüler") {
		t.Errorf("ErrBlocked nennt ein Kind: %q. Der Text ist der Kopf JEDER Sperrmeldung, "+
			"auch der eines gesperrten Geräts.", ErrBlocked)
	}
	// Gegenprobe: Der Kopf sagt trotzdem, worum es geht.
	if !strings.Contains(strings.ToLower(ErrBlocked.Error()), "gesperrt") {
		t.Errorf("ErrBlocked sagt nicht mehr, dass etwas gesperrt ist: %q", ErrBlocked)
	}
}
