package bereitschaft

// Wann der Bereitschafts-Alarm dieselbe Mail noch einmal schickt. Der Wächter läuft drei Minuten
// nach jedem Start und danach täglich (main.go); ohne Gedächtnis brächte jedes Update dieselbe
// Mail, und ein Alarm, der ständig kommt, wird überlesen. Wie repeat_interval bei Prometheus
// Alertmanager („How long to wait before repeating the last notification“) geht Unverändertes
// darum einmal am Tag hinaus, ein neuer Punkt mit dem nächsten Lauf.

import (
	"slices"
	"strings"
	"time"

	"bibliothek/pkg/logger"
)

// erinnerungsAbstand ist eine Stunde kürzer als der Takt des Wächters, damit ein Lauf, der um
// Sekunden früher kommt, die Erinnerung nicht um einen Tag verschiebt.
const erinnerungsAbstand = 23 * time.Hour

// GemeldeterAlarm ist, was die letzte verschickte Alarm-Mail nannte, und wann sie hinausging.
type GemeldeterAlarm struct {
	Zeit   time.Time `json:"zeit"`
	Punkte []string  `json:"punkte"`
}

// AlarmPunkte nennt, worüber die Mail berichtet: je kritischem Befund den Bereich, je
// Fehlermeldung Text und Route, mit dem Schalter alarm_warnungen (mitWarnungen) ebenso die
// Warnungen. Zahlen, Uhrzeiten und Ursachen zählen nicht dazu: Eine Meldung, die öfter kam, oder
// ein Backup, das älter wurde, ist kein neuer Punkt.
func AlarmPunkte(befunde []Befund, gruppen []logger.Fehlergruppe, mitWarnungen bool) []string {
	var punkte []string
	for _, b := range befunde {
		switch {
		case b.Stufe == StufeKritisch:
			punkte = append(punkte, "Befund: "+b.Bereich)
		case b.Stufe == StufeWarnung && mitWarnungen:
			punkte = append(punkte, "Warnung: "+b.Bereich)
		}
	}
	for _, g := range gruppen {
		art := "Fehler: "
		if g.Warnung {
			if !mitWarnungen {
				continue
			}
			art = "Warnung im Log: "
		}
		punkt := art + "weitere Meldungen"
		if !g.Sammel {
			punkt = strings.TrimSpace(art + g.Meldung + " " + g.Route)
		}
		punkte = append(punkte, punkt)
	}
	slices.Sort(punkte)
	return punkte
}

// IstMailFaellig sagt, ob die Mail hinausgeht: wenn sie einen Punkt nennt, den die letzte nicht
// nannte, oder wenn die letzte mindestens erinnerungsAbstand zurückliegt.
func IstMailFaellig(zuletzt GemeldeterAlarm, punkte []string, jetzt time.Time) bool {
	if jetzt.Sub(zuletzt.Zeit) >= erinnerungsAbstand {
		return true
	}
	for _, p := range punkte {
		if !slices.Contains(zuletzt.Punkte, p) {
			return true
		}
	}
	return false
}
