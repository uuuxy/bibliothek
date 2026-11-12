package littera

import (
	"errors"
	"testing"
	"time"
)

// Das abgelegte Ergebnis folgt derselben Regel wie der Rückgabewert des Befehls: abgebrochen
// mit Abbruch, vollständig nur ohne Fehler und mit jedem Abgleich.
func TestErgebnisAus(t *testing.T) {
	zeit := time.Date(2027, 1, 18, 12, 0, 0, 0, time.UTC)
	gut := Bericht{Warnungen: 2}
	gut.Bestand.AbgleichOK, gut.Schlagworte.AbgleichOK, gut.Personen.AbgleichOK, gut.Ausleihen.AbgleichOK = true, true, true, true

	abgebrochen := gut
	abgebrochen.Abbruch = errors.New("beim Bestand: verbindung weg")
	mitFehler := gut
	mitFehler.Fehler = 1
	ohneAbgleich := gut
	ohneAbgleich.Personen.AbgleichOK = false

	faelle := []struct {
		name string
		b    Bericht
		will Ergebnis
	}{
		{"vollständig", gut, Ergebnis{Zeitpunkt: zeit, Vollstaendig: true, Warnungen: 2, Protokoll: "p.log"}},
		{"abgebrochen", abgebrochen, Ergebnis{Zeitpunkt: zeit, Abgebrochen: true, Warnungen: 2, Protokoll: "p.log"}},
		{"mit Fehler", mitFehler, Ergebnis{Zeitpunkt: zeit, Fehler: 1, Warnungen: 2, Protokoll: "p.log"}},
		{"Abgleich fehlgeschlagen", ohneAbgleich, Ergebnis{Zeitpunkt: zeit, Warnungen: 2, Protokoll: "p.log"}},
	}
	for _, fall := range faelle {
		if got := ErgebnisAus(fall.b, zeit, "p.log"); got != fall.will {
			t.Errorf("%s: %+v, erwartet %+v", fall.name, got, fall.will)
		}
	}
}
