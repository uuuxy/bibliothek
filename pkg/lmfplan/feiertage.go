package lmfplan

import "time"

// feiertag ist ein gesetzlicher Feiertag mit seinem Namen; der Name steht im Planer als Grund.
type feiertag struct {
	Datum time.Time
	Name  string
}

// ostersonntag rechnet den Ostersonntag eines Jahres nach der Osterformel von Gauß in der
// Fassung von Lichtenberg; sie gilt für den gregorianischen Kalender ohne Sonderfälle.
func ostersonntag(jahr int) time.Time {
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

// feiertageHessen nennt die gesetzlichen Feiertage eines Jahres in Hessen (§ 1 HFeiertagsG):
// die bundesweiten und Fronleichnam, ohne Reformationstag, Allerheiligen und Buß- und Bettag.
// Der Tag ist ein Kalendertag in UTC; verglichen wird über Jahr, Monat und Tag
// (istGleicherTag), nicht über den Zeitpunkt.
//
// Hessen steht fest, weil die Schule in Hessen liegt und ein Schalter für das Land ohne zweite
// Schule eine Einstellung wäre, die niemand umstellt. Wer das Programm in einem anderen Land
// betreibt, ergänzt hier die Tabelle.
func feiertageHessen(jahr int) []feiertag {
	ostern := ostersonntag(jahr)
	fest := func(m time.Month, d int, name string) feiertag {
		return feiertag{Datum: time.Date(jahr, m, d, 0, 0, 0, 0, time.UTC), Name: name}
	}
	beweglich := func(tage int, name string) feiertag {
		return feiertag{Datum: ostern.AddDate(0, 0, tage), Name: name}
	}
	return []feiertag{
		fest(time.January, 1, "Neujahr"),
		beweglich(-2, "Karfreitag"),
		beweglich(1, "Ostermontag"),
		fest(time.May, 1, "Tag der Arbeit"),
		beweglich(39, "Christi Himmelfahrt"),
		beweglich(50, "Pfingstmontag"),
		beweglich(60, "Fronleichnam"),
		fest(time.October, 3, "Tag der Deutschen Einheit"),
		fest(time.December, 25, "1. Weihnachtstag"),
		fest(time.December, 26, "2. Weihnachtstag"),
	}
}
