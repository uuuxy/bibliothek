package lmfplan

import "time"

// Feiertag ist ein gesetzlicher Feiertag mit Namen — der Grund, der im Planer steht.
type Feiertag struct {
	Datum time.Time
	Name  string
}

// Ostersonntag nach der Gauß'schen Osterformel in der Fassung von Lichtenberg —
// gültig für den gregorianischen Kalender ohne Sonderfälle.
func Ostersonntag(jahr int) time.Time {
	k := jahr / 100
	m := 15 + (3*k+3)/4 - (8*k+13)/25
	s := 2 - (3*k+3)/4
	a := jahr % 19
	d := (19*a + m) % 30
	r := (d + a/11) / 29
	og := 21 + d - r
	sz := 7 - (jahr+jahr/4+s)%7
	oe := 7 - (og-sz)%7
	return time.Date(jahr, time.March, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, og+oe-1)
}

// FeiertageHessen nennt die gesetzlichen Feiertage eines Jahres in Hessen (§ 1 HFeiertagsG):
// die bundesweiten und Fronleichnam. Kein Reformationstag, kein Allerheiligen, kein
// Buß- und Bettag. Der Tag ist ein Kalendertag in UTC; verglichen wird über Jahr,
// Monat und Tag (istGleicherTag), nicht über den Zeitpunkt.
//
// Hessen ist hier fest — die Schule steht in Hessen, und ein Bundesland-Schalter ohne
// zweite Schule wäre eine Einstellung, die niemand je umstellt. Wer das Programm in
// einem anderen Land betreibt, ergänzt hier die Tabelle.
func FeiertageHessen(jahr int) []Feiertag {
	ostern := Ostersonntag(jahr)
	fix := func(m time.Month, d int, name string) Feiertag {
		return Feiertag{Datum: time.Date(jahr, m, d, 0, 0, 0, 0, time.UTC), Name: name}
	}
	beweglich := func(tage int, name string) Feiertag {
		return Feiertag{Datum: ostern.AddDate(0, 0, tage), Name: name}
	}
	return []Feiertag{
		fix(time.January, 1, "Neujahr"),
		beweglich(-2, "Karfreitag"),
		beweglich(1, "Ostermontag"),
		fix(time.May, 1, "Tag der Arbeit"),
		beweglich(39, "Christi Himmelfahrt"),
		beweglich(50, "Pfingstmontag"),
		beweglich(60, "Fronleichnam"),
		fix(time.October, 3, "Tag der Deutschen Einheit"),
		fix(time.December, 25, "1. Weihnachtstag"),
		fix(time.December, 26, "2. Weihnachtstag"),
	}
}

// istGleicherTag vergleicht zwei Zeitpunkte als Kalendertage, unabhängig von der Zone.
func istGleicherTag(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}
