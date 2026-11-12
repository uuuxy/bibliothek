package bereitschaft

import (
	"strings"
	"testing"
	"time"

	"bibliothek/internal/littera"
)

// Das Ergebnis der Littera-Übernahme ist eine Woche lang ein Befund: abgebrochen und
// unvollständig kritisch, mit Warnungen eine Warnung. Danach steht es als erledigt da, mit dem
// Ergebnis in Worten; nie gelaufen ist in Ordnung.
func TestPruefeLitteraUebernahme(t *testing.T) {
	jetzt := time.Date(2027, 1, 20, 12, 0, 0, 0, time.UTC)
	vorZweiTagen := jetzt.Add(-48 * time.Hour)
	const protokoll = "littera_import.log"
	faelle := []struct {
		name  string
		e     *littera.Ergebnis
		stufe string
		muss  []string
	}{
		{"nie gelaufen", nil, StufeOK, []string{"Keine Littera-Übernahme verzeichnet."}},
		{"abgebrochen", &littera.Ergebnis{Zeitpunkt: vorZweiTagen, Abgebrochen: true, Fehler: 1, Protokoll: protokoll},
			StufeKritisch, []string{"am 18.01.2027 13:00 ist abgebrochen (1 Fehler, 0 Warnungen", protokoll}},
		{"unvollständig", &littera.Ergebnis{Zeitpunkt: vorZweiTagen, Fehler: 3, Warnungen: 5, Protokoll: protokoll},
			StufeKritisch, []string{"ist unvollständig (3 Fehler, 5 Warnungen", protokoll}},
		{"vollständig mit Warnungen", &littera.Ergebnis{Zeitpunkt: vorZweiTagen, Vollstaendig: true, Warnungen: 3812, Protokoll: protokoll},
			StufeWarnung, []string{"ist vollständig, mit 3812 Warnungen", protokoll}},
		{"vollständig ohne Warnung", &littera.Ergebnis{Zeitpunkt: vorZweiTagen, Vollstaendig: true, Protokoll: protokoll},
			StufeOK, []string{"ist vollständig, ohne Warnung."}},
		{"genau eine Woche her: noch ein Befund", &littera.Ergebnis{Zeitpunkt: jetzt.Add(-litteraBefundDauer), Fehler: 1, Protokoll: protokoll},
			StufeKritisch, []string{"unvollständig"}},
		{"länger als eine Woche her: erledigt", &littera.Ergebnis{Zeitpunkt: jetzt.Add(-8 * 24 * time.Hour), Fehler: 3, Warnungen: 2, Protokoll: protokoll},
			StufeOK, []string{"Letzte Übernahme am 12.01.2027 13:00: unvollständig, 3 Fehler und 2 Warnungen."}},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			b := pruefeLitteraUebernahme(Lage{Jetzt: jetzt, LitteraUebernahme: fall.e})
			if b.Bereich != "Littera-Übernahme" || b.Stufe != fall.stufe {
				t.Errorf("Befund %+v, erwartet Bereich Littera-Übernahme mit Stufe %q", b, fall.stufe)
			}
			text := b.Befund + " " + b.Abhilfe
			for _, muss := range fall.muss {
				if !strings.Contains(text, muss) {
					t.Errorf("Befund ohne %q: %+v", muss, b)
				}
			}
			if b.Stufe != StufeOK && (b.Folge == "" || b.Abhilfe == "") {
				t.Errorf("Befund ohne Folge oder Abhilfe: %+v", b)
			}
		})
	}
}
