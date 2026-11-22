package auth

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

// Abgelaufene Fehlversuche fallen heraus, sobald die Liste groß wird, schon beim ersten
// Fehlversuch eines neuen Schlüssels: Nur neue Schlüssel lassen sie wachsen, und sonst wüchse sie
// mit jeder durchprobierten Adresse ohne Grenze.
func TestFehlversuchsBremse_RaeumtAbgelaufenes(t *testing.T) {
	b := newFehlversuchsBremse(3, time.Minute)
	vorbei := time.Now().Add(-time.Hour)
	for i := 0; i <= 2000; i++ {
		b.eintraege[fmt.Sprintf("alt-%d", i)] = &fehlversuche{anzahl: 1, fensterEnde: vorbei}
	}

	b.merkeFehlversuch("frisch")

	if len(b.eintraege) != 1 {
		t.Errorf("%d Einträge, erwartet nur den frischen", len(b.eintraege))
	}
}

// Laufende Versuche teilen sich die Grenze mit den gezählten Fehlversuchen. Gezählt wird erst
// nach der Antwort des Mailservers; ohne das gingen beliebig viele gleichzeitige Versuche gegen
// ein Konto durch, bevor der erste zählt.
func TestAnmelden_GleichzeitigeVersucheTeilenDieGrenze(t *testing.T) {
	a, _ := newTestAnmeldung(t, 12*time.Hour)

	versuche := func() (schluessel string, ok bool) {
		return a.pruefeFehlversuche(httptest.NewRecorder(), "ziel@schule.de", "10.0.0.7")
	}
	durch, schluessel := 0, ""
	for range 20 {
		if k, ok := versuche(); ok {
			durch, schluessel = durch+1, k
		}
	}
	if durch != 5 {
		t.Fatalf("%d von 20 gleichzeitigen Versuchen gingen durch, erwartet 5", durch)
	}

	// Ein Versuch, der ohne Fehlversuch endet, gibt seinen Platz frei.
	a.bremse.gibFrei(schluessel)
	if _, ok := versuche(); !ok {
		t.Error("nach einem beendeten Versuch ging kein weiterer durch")
	}
	// Ein gezählter Fehlversuch nicht: Er belegt den Platz bis zum Ende des Fensters.
	a.bremse.merkeFehlversuch(schluessel)
	a.bremse.gibFrei(schluessel)
	if _, ok := versuche(); ok {
		t.Error("nach einem Fehlversuch ging ein weiterer Versuch durch")
	}
}
