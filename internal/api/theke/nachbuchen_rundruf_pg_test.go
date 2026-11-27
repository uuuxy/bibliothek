package theke

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Was die Theke ohne Netz gescannt hat, erfahren die übrigen Arbeitsplätze beim Nachbuchen wie
// eine Buchung an der Theke: Der Rundruf nennt den Leser, dessen Ausleihe beginnt oder endet,
// auch bei einer Rückgabe ohne geöffnete Akte und bei einer fremden Rückgabe. Eine Wiederholung
// mit demselben Schlüssel bucht nichts und ruft nicht.
func TestNachbuchen_RundrufWieAnDerTheke(t *testing.T) {
	w := baueNachbuchWelt(t)
	broker, rundrufeNach := hoereRundrufe(t)
	w.srv.Broker = broker

	start := time.Now().Add(-5 * time.Minute)
	ausleihe := w.eintrag(uuid.NewString(), "ausleihe", &w.ben, start)
	for _, f := range []struct {
		name, ergebnis, ereignis string
		eintrag                  map[string]any
	}{
		{"Ausleihe an Ben", "ausgeliehen", "ausleihe", ausleihe},
		{"Rückgabe ohne geöffnete Akte", "zurueckgegeben", "rueckgabe", w.eintrag(uuid.NewString(), "rueckgabe", nil, start.Add(time.Minute))},
		{"zweite Ausleihe an Ben", "ausgeliehen", "ausleihe", w.eintrag(uuid.NewString(), "ausleihe", &w.ben, start.Add(2*time.Minute))},
		{"Rückgabe bei Annas Akte", "zurueckgegeben", "fremdrueckgabe", w.eintrag(uuid.NewString(), "rueckgabe", &w.anna, start.Add(3*time.Minute))},
	} {
		var ergebnis string
		rundrufe := rundrufeNach(func() { ergebnis = w.nachbuchen(t, f.eintrag)[0].Ergebnis })
		if ergebnis != f.ergebnis {
			t.Fatalf("%s: Ergebnis %q, erwartet %q", f.name, ergebnis, f.ergebnis)
		}
		if len(rundrufe) != 1 || !strings.Contains(rundrufe[0], `"event":"`+f.ereignis+`"`) || !strings.Contains(rundrufe[0], `"student_id":"`+w.ben+`"`) {
			t.Errorf("%s: Rundrufe %q — erwartet genau einen mit event %q und Bens Kennung", f.name, rundrufe, f.ereignis)
		}
	}
	if rundrufe := rundrufeNach(func() { w.nachbuchen(t, ausleihe) }); len(rundrufe) != 0 {
		t.Errorf("Wiederholung mit demselben Schlüssel: Rundrufe %q — erwartet keinen", rundrufe)
	}
}
