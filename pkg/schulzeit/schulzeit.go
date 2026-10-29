// Package schulzeit trägt die Zeitzone der Schule und rechnet mit ihren Kalendertagen.
//
// Der Server und die Sitzung der Datenbank laufen in UTC. Zwischen Mitternacht in Berlin und
// Mitternacht UTC trüge ein Schreiben sonst das Datum des Vortags und eine Frist einen Tag zu
// wenig. Das Paket braucht nur die Standardbibliothek, damit jedes andere es einbinden kann und
// die Zone an einer Stelle steht.
package schulzeit

import (
	"sync"
	"time"
)

var (
	zone     *time.Location
	zoneOnce sync.Once
)

// ZonenName ist der Name der Zeitzone der Schule, für Go (Zone) und für SQL (SQLHeute).
const ZonenName = "Europe/Berlin"

// SQLHeute ist der heutige Kalendertag der Schule als SQL-Ausdruck. Er steht statt
// CURRENT_DATE überall, wo ein Tag gemeint ist: Die Sitzung der Datenbank läuft in UTC, und
// CURRENT_DATE nennt dort bis 2 Uhr Berliner Zeit den Vortag. Der Ausdruck steht nur hier,
// damit ihn niemand ein zweites Mal anders schreibt.
//
// Für den Vergleich von Zeitpunkten ist er nicht gedacht: „überfällig" vergleicht
// rueckgabe_frist mit CURRENT_TIMESTAMP und bleibt so.
const SQLHeute = `(now() AT TIME ZONE '` + ZonenName + `')::date`

// Zone liefert die Zeitzone der Schule. Lässt sie sich nicht laden, weil dem Image die
// Zeitzonendaten fehlen, gilt UTC: lieber eine Stunde daneben als ein Programm, das nicht
// startet.
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

// Jetzt liefert den Zeitpunkt in der Zone der Schule. Er gehört überall hin, wo ein
// Kalendertag entsteht: ein Briefdatum, eine Frist, ein „Stand vom".
func Jetzt() time.Time {
	return time.Now().In(Zone())
}

// Kalendertag liest ein Datum (JJJJ-MM-TT) als Kalendertag der Schule: Mitternacht in ihrer
// Zone. In UTC gelesen läge der Tag um eine oder zwei Stunden daneben.
func Kalendertag(datum string) (time.Time, error) {
	return time.ParseInLocation(time.DateOnly, datum, Zone())
}

// TagesEnde liefert das Ende des Kalendertags von t (23:59:59) in der Zone der Schule. Es ist
// die eine Festlegung von „Ende des Tages".
func TagesEnde(t time.Time) time.Time {
	loc := Zone()
	d := t.In(loc)
	return time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, loc)
}

// Halbjahr liefert das Schulhalbjahr, in dem ein Tag liegt, als ersten und letzten
// Kalendertag.
//
// Die Stichtage sind der 15.3. und der 15.9.: An ihnen verlangt die Arbeitshilfe den Ausdruck
// der Bestandskartei, und „je Schulhalbjahr ein Ausdruck der Neuanschaffungen" meint dieselben
// zwei Schnitte (docs/mittel_konzept.md 7.1). Ein Halbjahr läuft also vom 16.3. bis zum 15.9.
// und vom 16.9. bis zum 15.3. des Folgejahres.
//
// Das Schuljahr (1.8. bis 31.7.) ist etwas anderes. Eine Liste nach Schuljahr verfehlte einen
// Stichtag, und der Nachweis, den die Schule abheftet, deckte einen anderen Zeitraum als den
// genannten.
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

// VorigesHalbjahr liefert das Schulhalbjahr vor dem, in dem ein Tag liegt. Das Zugangsbuch
// schlägt es vor: Ausgedruckt wird „zu Beginn jedes Schulhalbjahres ein Ausdruck der
// Neuanschaffungen des vorhergehenden Halbjahres" (Leitfaden Lernmittelfreiheit, 11.5).
func VorigesHalbjahr(t time.Time) (von, bis time.Time) {
	anfang, _ := Halbjahr(t)
	return Halbjahr(anfang.AddDate(0, 0, -1))
}
