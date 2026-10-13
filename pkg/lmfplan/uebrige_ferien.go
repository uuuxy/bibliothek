package lmfplan

import "time"

// uebrigeFerienHessen sind die Herbst-, Weihnachts- und Osterferien in Hessen, je mit erstem
// und letztem Ferientag. Sie stehen wie die Sommerferien als Tabelle im Programm und nicht als
// Datei, die die Schule hochlädt, weil das Ministerium sie für vier Schuljahre im Voraus
// festlegt.
//
// Quelle: Hessisches Ministerium für Kultus, Bildung und Chancen,
// kultus.hessen.de/schulsystem/ferien/ferientermine, abgerufen am 24.09.2026.
//
// Die beweglichen Ferientage, drei oder vier je Schuljahr, legt jede Schule selbst; sie stehen
// nicht darin. Fällt eine Frist auf einen davon, bleibt sie dort.
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

// ferienZeitraum baut einen Eintrag der Tabelle mit Namen; beide Tage zählen mit.
func ferienZeitraum(name, von, bis string) Zeitraum {
	z := ferien(von, bis)
	z.Name = name
	return z
}

// UebrigeFerienBis nennt das Jahr, in dem die letzten Ferien der Tabelle enden. Bis dahin
// kennen die Leihfristen Herbst-, Weihnachts- und Osterferien; danach rücken sie nur noch über
// Wochenenden, Feiertage und Sommerferien.
func UebrigeFerienBis() int {
	letztes := 0
	for _, z := range uebrigeFerienHessen {
		letztes = max(letztes, z.Bis.Year())
	}
	return letztes
}

// ersterSchultagAb liefert den Tag selbst, wenn an ihm Schule ist, sonst den nächsten
// Schultag: kein Wochenende, kein gesetzlicher Feiertag in Hessen und keine Ferien, weder die
// Sommerferien dieser Tabelle noch die übrigen.
//
// Es zählt der Kalendertag in der Zone von tag; das Ergebnis ist dieser Tag in UTC wie die
// Tabellen. Ein Berliner Zeitpunkt lässt sich nicht direkt vergleichen: Mitternacht in Berlin
// liegt vor Mitternacht UTC, und der erste Ferientag fiele heraus.
func (t Ferientabelle) ersterSchultagAb(tag time.Time) time.Time {
	frei := make([]Zeitraum, 0, len(t.jahre)+len(uebrigeFerienHessen))
	for _, z := range t.jahre {
		frei = append(frei, z)
	}
	frei = append(frei, uebrigeFerienHessen...)
	return naechsterSchultag(kalendertagUTC(tag), Schultage(frei))
}
