package service

import (
	"strings"
	"testing"
)

// Die Sperrmeldung eines Geräts darf kein Kind erfinden.
//
// Trüge ErrGesperrt den Satz „ausleihe für diese/n Schüler/in ist gesperrt" und hängte der
// Geräte-Pfad „Gerät ist aktuell gesperrt" an, stünde an der Theke eine Meldung über ein Kind,
// das an diesem Vorgang gar nicht beteiligt ist — die Bibliothekskraft suchte nach einer
// Sperre, die es nicht gibt.
func TestSperrmeldung_NenntKeinKindWennKeinesBeteiligtIst(t *testing.T) {
	if strings.Contains(strings.ToLower(ErrGesperrt.Error()), "schüler") {
		t.Errorf("ErrGesperrt nennt ein Kind: %q. Der Text ist der Kopf JEDER Sperrmeldung, "+
			"auch der eines gesperrten Geräts.", ErrGesperrt)
	}
	// Gegenprobe: Der Kopf sagt trotzdem, worum es geht.
	if !strings.Contains(strings.ToLower(ErrGesperrt.Error()), "gesperrt") {
		t.Errorf("ErrGesperrt sagt nicht mehr, dass etwas gesperrt ist: %q", ErrGesperrt)
	}
}
