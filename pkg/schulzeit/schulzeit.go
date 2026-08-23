// Package schulzeit trägt die Zeitzone der Schule — und sonst nichts.
//
// Sie lag bis zum 23.08.2026 als unexportierte `schoolLocation()` in
// `internal/service` und war damit für alles unerreichbar, was nicht die Ausleihlogik
// ist. Die vier Datumsangaben in den gedruckten Dokumenten nahmen deshalb `time.Now()`
// in der Container-Zeit (im Image UTC): das Datum auf dem Schadensbescheid, die
// 14-Tage-Zahlungsfrist darauf, das Rechnungsdatum und der "Stand" des Kontoauszugs.
// Zwischen 22 und 24 Uhr UTC ist in Berlin schon der Folgetag — das Schreiben trüge
// dann den Vortag und die Frist einen Tag zu wenig.
//
// Ein zweites `time.LoadLocation("Europe/Berlin")` neben dem in service wäre die
// bequeme Lösung gewesen und genau der Fehler: zwei Wahrheitsquellen für dieselbe
// Zeitzone. Deshalb hier, in einem Paket, das nichts außer der Standardbibliothek
// braucht und das jeder importieren kann — auch `pdf`, das sonst nichts aus diesem
// Projekt kennt.
package schulzeit

import (
	"sync"
	"time"
)

var (
	zone     *time.Location
	zoneOnce sync.Once
)

// Zone liefert die feste Zeitzone der Schule (Europe/Berlin). Fällt das Laden fehl
// (fehlende tzdata im Image), wird sicher auf UTC zurückgegriffen — lieber eine
// Stunde daneben als ein Programm, das nicht startet.
func Zone() *time.Location {
	zoneOnce.Do(func() {
		loc, err := time.LoadLocation("Europe/Berlin")
		if err != nil {
			loc = time.UTC
		}
		zone = loc
	})
	return zone
}

// Jetzt ist "jetzt" aus Sicht der Schule. Überall dort zu benutzen, wo ein
// KALENDERTAG entsteht — ein Briefdatum, eine Frist, ein "Stand vom".
func Jetzt() time.Time {
	return time.Now().In(Zone())
}

// TagesEnde normalisiert einen Zeitpunkt auf das Ende seines Kalendertags (23:59:59)
// in der Schul-Zeitzone. Dies ist die EINZIGE Definition von "Ende des Tages" im
// System; internal/service reicht seine TagesEndeInSchulzeitzone hierher durch.
func TagesEnde(t time.Time) time.Time {
	loc := Zone()
	d := t.In(loc)
	return time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, loc)
}
