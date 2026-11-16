package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"bibliothek/internal/adressabgleich"
)

// Der Vorrat holt das Adressbuch höchstens einmal je adressbuchAbstand, auch nach einem Fehler.
// Nach einem Fehler bleibt das vorige Adressbuch; ist der Zugang nicht mehr eingerichtet, gilt
// keins.
func TestAdressbuchVorrat_FetchesAtMostOncePerInterval(t *testing.T) {
	buch := &testAdressbuch{eintraege: []adressabgleich.Eintrag{eintragMit("Vera", "Vorrat", "kvor")}}
	v := &adressbuchVorrat{quelle: buch.hole}
	ctx := context.Background()

	for range 3 {
		if n := len(v.aktuell(ctx)); n != 1 {
			t.Fatalf("aktuell = %d Einträge, erwartet 1", n)
		}
	}
	if n := buch.anzahlAbrufe(); n != 1 {
		t.Errorf("%d Abrufe in einem Abstand, erwartet 1", n)
	}

	setzeQuelle := func(fehler error) {
		buch.mu.Lock()
		buch.eintraege, buch.fehler = nil, fehler
		buch.mu.Unlock()
		v.geholtAm = time.Now().Add(-adressbuchAbstand)
	}
	setzeQuelle(errors.New("webmail weg"))
	for range 2 {
		if n := len(v.aktuell(ctx)); n != 1 {
			t.Errorf("nach gescheitertem Abruf: %d Einträge, erwartet das vorige Adressbuch", n)
		}
	}
	if n := buch.anzahlAbrufe(); n != 2 {
		t.Errorf("%d Abrufe, erwartet 2: Nach dem Fehler wartet der nächste Abruf den Abstand ab", n)
	}

	setzeQuelle(adressabgleich.ErrNichtEingerichtet)
	if n := len(v.aktuell(ctx)); n != 0 {
		t.Errorf("ohne eingerichteten Zugang: %d Einträge, erwartet keine", n)
	}
}

// Melden sich mehrere zum ersten Mal gleichzeitig an, holt nur eine Anmeldung das Adressbuch.
func TestAdressbuchVorrat_ConcurrentCallsFetchOnce(t *testing.T) {
	buch := &testAdressbuch{eintraege: []adressabgleich.Eintrag{eintragMit("Gina", "Gleichzeitig", "kgle")}}
	v := &adressbuchVorrat{quelle: buch.hole}
	var gruppe sync.WaitGroup
	for range 20 {
		gruppe.Go(func() {
			if n := len(v.aktuell(context.Background())); n != 1 {
				t.Errorf("aktuell = %d Einträge, erwartet 1", n)
			}
		})
	}
	gruppe.Wait()
	if n := buch.anzahlAbrufe(); n != 1 {
		t.Errorf("%d Abrufe bei gleichzeitigen Anmeldungen, erwartet 1", n)
	}
}
