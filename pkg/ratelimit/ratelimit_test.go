package ratelimit

import (
	"fmt"
	"testing"
	"time"
)

// Der Vorrat gilt je Adresse: Nach limit Anfragen in Folge wird die nächste derselben Adresse
// abgewiesen, eine andere Adresse hat ihren eigenen. Mit einem Vorrat von zwei füllt sich in der
// Laufzeit der Schleife keine ganze Anfrage nach.
func TestBremse_VorratJeAdresse(t *testing.T) {
	l := New(2)
	for i := range 2 {
		if !l.LaesstDurch("10.0.0.1") {
			t.Fatalf("Anfrage %d einer Adresse wurde abgewiesen, der Vorrat war zwei", i+1)
		}
	}
	if l.LaesstDurch("10.0.0.1") {
		t.Error("die dritte Anfrage derselben Adresse ging durch, der Vorrat war zwei")
	}
	if !l.LaesstDurch("10.0.0.2") {
		t.Error("eine andere Adresse wurde abgewiesen, obwohl ihr Vorrat voll ist")
	}
}

// Der Vorrat füllt sich mit der Zeit und nie über limit hinaus: Nach einer langen Pause gehen
// wieder genau limit Anfragen durch.
func TestBremse_FuelltSichBisZumLimit(t *testing.T) {
	l := New(2)
	l.LaesstDurch("10.0.0.1")
	l.LaesstDurch("10.0.0.1")
	// Die letzte Anfrage liegt eine Stunde zurück.
	l.jeAdresse["10.0.0.1"].zuletzt = time.Now().Add(-time.Hour)
	for i := range 2 {
		if !l.LaesstDurch("10.0.0.1") {
			t.Fatalf("nach der Pause wurde Anfrage %d abgewiesen, der Vorrat war wieder zwei", i+1)
		}
	}
	if l.LaesstDurch("10.0.0.1") {
		t.Error("nach der Pause gingen mehr als zwei Anfragen durch: Der Vorrat wuchs über das Limit")
	}
}

// Über 5000 Adressen räumt eine neue Adresse die auf, die seit fünf Minuten nicht mehr gefragt
// haben; sonst wüchse die Liste mit jedem Besucher.
func TestBremse_RaeumtAlteAdressen(t *testing.T) {
	l := New(2)
	alt := time.Now().Add(-10 * time.Minute)
	for i := range 5001 {
		l.jeAdresse[fmt.Sprintf("alt-%d", i)] = &besucher{vorrat: 2, zuletzt: alt}
	}
	l.LaesstDurch("10.0.0.1")
	if n := len(l.jeAdresse); n != 1 {
		t.Errorf("%d Adressen nach dem Aufräumen, erwartet nur die neue", n)
	}
}
