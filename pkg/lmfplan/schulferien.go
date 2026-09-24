package lmfplan

import "time"

// schulferien.go — die übrigen Ferien Hessens (Herbst, Weihnachten, Ostern) und die Frage,
// die eine Leihfrist an den Kalender stellt: Ist an diesem Tag Schule?
//
// Entscheidung vom 24.09.2026: Die Termine stehen als Tabelle im Programm, wie die
// Sommerferien (ferien.go) — nicht als Datei, die die Schule hochlädt. Das Ministerium legt
// sie für vier Schuljahre im Voraus fest; Littera hatte eine Liste von Hand
// („Schließtage"), und die Schule hat sie dort nie gefüllt (gemessen in der Sicherung
// littera_sav.mdb: Tabelle leer, keine der 15.615 Ausleihen verschoben).
//
// Quelle: Hessisches Ministerium für Kultus, Bildung und Chancen,
// kultus.hessen.de/schulsystem/ferien/ferientermine, abgerufen am 24.09.2026 — „Angegeben
// ist der erste und letzte Ferientag." Die Schuljahre 2026/27 bis 2028/29 stimmen Tag für
// Tag mit den Ferienkalendern der KMK überein (kmk.org/service/ferien, FER2026_27 bis
// FER2028_29); 2029/30 steht bisher nur beim Ministerium. Die Sommerferien derselben Seite
// sind die von sommerferienHessen.
//
// Nicht enthalten sind die beweglichen Ferientage (drei oder vier je Schuljahr), die jede
// Schule selbst legt. Fällt eine Frist auf einen davon, bleibt sie dort, und wer am
// nächsten Schultag zurückgibt, ist einen Tag drüber — gemahnt wird von Hand, und die
// Sperre an der Theke greift erst nach den Karenztagen (max_overdue_days).
var uebrigeFerienHessen = []Zeitraum{
	ferienZeitraum("Herbstferien", "2026-10-05", "2026-10-17"),
	ferienZeitraum("Weihnachtsferien", "2026-12-23", "2027-01-12"),
	ferienZeitraum("Osterferien", "2027-03-22", "2027-04-02"),
	ferienZeitraum("Herbstferien", "2027-10-04", "2027-10-16"),
	ferienZeitraum("Weihnachtsferien", "2027-12-23", "2028-01-11"),
	ferienZeitraum("Osterferien", "2028-04-03", "2028-04-14"),
	ferienZeitraum("Herbstferien", "2028-10-09", "2028-10-20"),
	ferienZeitraum("Weihnachtsferien", "2028-12-27", "2029-01-12"),
	ferienZeitraum("Osterferien", "2029-03-29", "2029-04-13"),
	ferienZeitraum("Herbstferien", "2029-10-15", "2029-10-26"),
	ferienZeitraum("Weihnachtsferien", "2029-12-24", "2030-01-11"),
	ferienZeitraum("Osterferien", "2030-04-08", "2030-04-22"),
}

// ferienZeitraum baut einen benannten Tabelleneintrag; beide Tage zählen mit.
func ferienZeitraum(name, von, bis string) Zeitraum {
	z := ferien(von, bis)
	z.Name = name
	return z
}

// UebrigeFerienBis nennt das Jahr, in dem die letzten Ferien der Tabelle oben enden. Bis
// dahin kennen die Leihfristen Herbst-, Weihnachts- und Osterferien; danach rücken sie nur
// noch über Wochenenden, Feiertage und Sommerferien.
func UebrigeFerienBis() int {
	letztes := 0
	for _, z := range uebrigeFerienHessen {
		letztes = max(letztes, z.Bis.Year())
	}
	return letztes
}

// NaechsterSchultag liefert den Tag selbst, wenn an ihm Schule ist, sonst den nächsten
// Schultag: kein Wochenende, kein gesetzlicher Feiertag in Hessen, keine Ferien — die
// Sommerferien dieser Tabelle samt eigener Einträge und die übrigen Ferien oben.
//
// Maßgeblich ist der Kalendertag in der Zone von tag; das Ergebnis ist dieser Tag in UTC wie
// die Tabellen. Ein Berliner Zeitpunkt darf nicht direkt verglichen werden: 05.10. 00:00
// Berlin liegt vor 05.10. 00:00 UTC, und der erste Ferientag fiele heraus.
func (t Ferientabelle) NaechsterSchultag(tag time.Time) time.Time {
	frei := make([]Zeitraum, 0, len(t.jahre)+len(uebrigeFerienHessen))
	for _, z := range t.jahre {
		frei = append(frei, z)
	}
	frei = append(frei, uebrigeFerienHessen...)
	return naechsterSchultag(kalendertagUTC(tag), Schultage(frei))
}
