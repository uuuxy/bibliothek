package ratelimit

import (
	"fmt"
	"testing"
	"time"
)

// Der Vorrat gilt je Adresse: Nach limit Anfragen in Folge wird die nächste derselben Adresse
// abgewiesen, eine andere Adresse hat ihren eigenen. Mit einem Vorrat von zwei füllt sich in der
// Laufzeit der Schleife keine ganze Anfrage nach.
func TestLimiter_VorratJeAdresse(t *testing.T) {
	l := New(2)
	for i := range 2 {
		if !l.Allow("10.0.0.1") {
			t.Fatalf("Anfrage %d einer Adresse wurde abgewiesen, der Vorrat war zwei", i+1)
		}
	}
	if l.Allow("10.0.0.1") {
		t.Error("die dritte Anfrage derselben Adresse ging durch, der Vorrat war zwei")
	}
	if !l.Allow("10.0.0.2") {
		t.Error("eine andere Adresse wurde abgewiesen, obwohl ihr Vorrat voll ist")
	}
}

// Der Vorrat füllt sich mit der Zeit und nie über limit hinaus: Nach einer langen Pause gehen
// wieder genau limit Anfragen durch.
func TestLimiter_FuelltSichBisZumLimit(t *testing.T) {
	l := New(2)
	l.Allow("10.0.0.1")
	l.Allow("10.0.0.1")
	// Die letzte Anfrage liegt eine Stunde zurück.
	l.visitors["10.0.0.1"].lastRefill = time.Now().Add(-time.Hour)
	for i := range 2 {
		if !l.Allow("10.0.0.1") {
			t.Fatalf("nach der Pause wurde Anfrage %d abgewiesen, der Vorrat war wieder zwei", i+1)
		}
	}
	if l.Allow("10.0.0.1") {
		t.Error("nach der Pause gingen mehr als zwei Anfragen durch: Der Vorrat wuchs über das Limit")
	}
}

// Über 5000 Adressen räumt eine neue Adresse die auf, die seit fünf Minuten nicht mehr gefragt
// haben; sonst wüchse die Liste mit jedem Besucher.
func TestLimiter_RaeumtAlteAdressen(t *testing.T) {
	l := New(2)
	alt := time.Now().Add(-10 * time.Minute)
	for i := range 5001 {
		l.visitors[fmt.Sprintf("alt-%d", i)] = &visitor{tokens: 2, lastRefill: alt}
	}
	l.Allow("10.0.0.1")
	if n := len(l.visitors); n != 1 {
		t.Errorf("%d Adressen nach dem Aufräumen, erwartet nur die neue", n)
	}
}
