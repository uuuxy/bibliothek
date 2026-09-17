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

// ZonenName ist der Name der Schulzeitzone — für Go (Zone) und für SQL, das den
// Kalendertag selbst bilden muss (SQLHeute). Die Datenbank-Sitzung läuft in UTC;
// CURRENT_DATE ist dort bis 2 Uhr Berliner Zeit noch der Vortag.
const ZonenName = "Europe/Berlin"

// SQLHeute ist der heutige KALENDERTAG der Schule als SQL-Ausdruck — der Ersatz für
// CURRENT_DATE überall dort, wo ein Tag gemeint ist und kein Zeitpunkt.
//
// Er steht hier und nicht in einem der Pakete, die ihn brauchen: Zwischen Mitternacht in
// Berlin und Mitternacht UTC liefert CURRENT_DATE den Vortag, und diese zwei Stunden
// haben in diesem Projekt schon dreimal etwas Falsches erzeugt (Bescheid-Frist,
// Volljährigkeit, Mahnlauf). Eine zweite Formulierung desselben Ausdrucks wäre die
// nächste Gelegenheit dazu.
//
// NICHT für Vergleiche von Zeitpunkten: „überfällig" ist ein Instant-Vergleich
// (rueckgabe_frist < CURRENT_TIMESTAMP) und bleibt einer.
const SQLHeute = `(now() AT TIME ZONE '` + ZonenName + `')::date`

// Zone liefert die feste Zeitzone der Schule (ZonenName). Fällt das Laden fehl
// (fehlende tzdata im Image), wird sicher auf UTC zurückgegriffen — lieber eine
// Stunde daneben als ein Programm, das nicht startet.
func Zone() *time.Location {
	zoneOnce.Do(func() {
		loc, err := time.LoadLocation(ZonenName)
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

// Halbjahr liefert das Schulhalbjahr, in dem ein Tag liegt — als Kalendertage von/bis.
//
// Die Stichtage sind der 15.3. und der 15.9.: An ihnen verlangt die Arbeitshilfe den
// Ausdruck der Bestandskartei, und „je Schulhalbjahr ein Ausdruck der Neuanschaffungen"
// meint dieselben zwei Schnitte (docs/mittel_konzept.md 7.1). Ein Halbjahr läuft also vom
// 16.3. bis 15.9. und vom 16.9. bis 15.3. des Folgejahres.
//
// Das ist NICHT das Schuljahr (1.8.–31.7.), und die Unterscheidung ist der Grund für diese
// Funktion: Wer beides im Kopf zusammenzieht, bekommt eine Liste, die einen Stichtag
// verfehlt — und der Nachweis, den die Schule abheftet, deckt dann einen anderen Zeitraum
// ab als den, den er behauptet.
func Halbjahr(t time.Time) (von, bis time.Time) {
	loc := Zone()
	d := t.In(loc)
	tag := func(jahr int, monat time.Month, tag int) time.Time {
		return time.Date(jahr, monat, tag, 0, 0, 0, 0, loc)
	}
	fruehling := tag(d.Year(), time.March, 16)  // Beginn des Sommerhalbjahres
	herbst := tag(d.Year(), time.September, 16) // Beginn des Winterhalbjahres

	switch {
	case d.Before(fruehling):
		// Januar bis 15.3.: Das Halbjahr hat im Vorjahr begonnen.
		return tag(d.Year()-1, time.September, 16), tag(d.Year(), time.March, 15)
	case d.Before(herbst):
		return fruehling, tag(d.Year(), time.September, 15)
	default:
		return herbst, tag(d.Year()+1, time.March, 15)
	}
}
