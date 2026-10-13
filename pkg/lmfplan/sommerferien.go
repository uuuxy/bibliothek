package lmfplan

import "time"

// sommerferienHessen sind die Sommerferien in Hessen nach der langfristigen Regelung der
// Kultusministerkonferenz, kmk.org/service/ferienregelung (Beschluss vom 21.09.2022 für 2025
// bis 2030). An ihnen richten sich die Pläne aus: Der Büchertausch endet am Donnerstag vor den
// Ferien.
//
// Die Tabelle ist die Vorbelegung; spätere Jahre trägt die Schule als Einstellung selbst ein
// (ferien_einstellung.go). Läuft sie aus, sagt es der Planer. Zwei Jahre vorher warnen die
// Selbstprüfung unter System → Betriebsbereitschaft und TestSommerferienHessen_Horizont, damit
// der nächste Beschluss auch hier nachgetragen wird und eine frische Installation ihn hat.
var sommerferienHessen = map[int]Zeitraum{
	2025: ferien("2025-07-07", "2025-08-15"),
	2026: ferien("2026-06-29", "2026-08-07"),
	2027: ferien("2027-06-28", "2027-08-06"),
	2028: ferien("2028-07-03", "2028-08-11"),
	2029: ferien("2029-07-16", "2029-08-24"),
	2030: ferien("2030-07-22", "2030-08-30"),
}

// ferien baut einen Eintrag der Tabelle. Ein Tippfehler in der Tabelle ist ein Fehler im
// Programm und zeigt sich so beim Start und nicht erst im Juni.
func ferien(von, bis string) Zeitraum {
	v, err := time.Parse(time.DateOnly, von)
	if err != nil {
		panic("Ferientabelle: " + err.Error())
	}
	b, err := time.Parse(time.DateOnly, bis)
	if err != nil {
		panic("Ferientabelle: " + err.Error())
	}
	return Zeitraum{Von: v, Bis: b, Name: "Sommerferien"}
}

// Ferientabelle sind die Sommerferien, an denen sich die Pläne ausrichten: die Tabelle des
// Programms und dazu, was die Schule selbst eingetragen hat. Ein eigener Eintrag gilt vor dem
// des Programms für dasselbe Jahr. Die Schule trägt selbst ein, weil sie eine Warnung, die nur
// ein Entwickler beheben kann, nicht abstellen könnte.
type Ferientabelle struct {
	jahre map[int]Zeitraum
}

// Hessen ist die Tabelle des Programms allein; sie gilt, wenn nichts eingestellt ist.
func Hessen() Ferientabelle {
	return Ferientabelle{jahre: sommerferienHessen}
}

// mit ergänzt die Tabelle um eigene Einträge; für dasselbe Jahr gilt der eigene.
func (t Ferientabelle) mit(eintraege []SommerferienEintrag) Ferientabelle {
	if len(eintraege) == 0 {
		return t
	}
	jahre := make(map[int]Zeitraum, len(t.jahre)+len(eintraege))
	for j, z := range t.jahre {
		jahre[j] = z
	}
	for _, e := range eintraege {
		if z, ok := e.zeitraum(); ok {
			jahre[e.Jahr] = z
		}
	}
	return Ferientabelle{jahre: jahre}
}

// LueckenlosBis nennt das letzte Jahr, bis zu dem die Tabelle ab dem Jahr ab ohne Lücke
// reicht, und ab-1, wenn schon ab fehlt.
//
// Gefragt wird nach der Lücke und nicht nach dem letzten Jahr der Tabelle, weil der Planer
// genau ein Jahr braucht: Trägt die Schule 2032 und 2033 ein und vergisst 2031, steht der
// Planer 2031 ohne Vorgabe da, obwohl die Tabelle bis 2033 reicht.
func (t Ferientabelle) LueckenlosBis(ab int) int {
	jahr := ab
	for {
		if _, ok := t.jahre[jahr]; !ok {
			return jahr - 1
		}
		jahr++
	}
}

// Sommerferien nennt die Sommerferien eines Jahres als Kalendertage in UTC; ok ist false, wenn
// das Jahr nicht bekannt ist.
func (t Ferientabelle) Sommerferien(jahr int) (Zeitraum, bool) {
	z, ok := t.jahre[jahr]
	return z, ok
}

// Naechste nennt die Sommerferien, an denen sich der nächste Plan ausrichtet, von heute aus
// gesehen. Mit bevorstehend sind es die nächsten, die noch nicht begonnen haben (vor ihnen
// liegt der Büchertausch), sonst die nächsten, die noch nicht vorbei sind (nach ihnen liegt
// die Bücherausgabe). Das Jahr kommt auch zurück, wenn es nicht bekannt ist (ok ist false);
// der Planer nennt es dann im Hinweis.
func (t Ferientabelle) Naechste(heute time.Time, bevorstehend bool) (Zeitraum, int, bool) {
	tag := kalendertag(heute)
	jahr := tag.Year()
	z, ok := t.Sommerferien(jahr)
	if !ok {
		return Zeitraum{}, jahr, false
	}
	vorbei := istVorbei(z, tag, bevorstehend)
	if !vorbei {
		return z, jahr, true
	}
	z, ok = t.Sommerferien(jahr + 1)
	return z, jahr + 1, ok
}

// istVorbei sagt, ob die Ferien für den Zweck schon vorbei sind: begonnen (bevorstehend) oder
// beendet. Verglichen werden Kalendertage, unabhängig von der Zone.
func istVorbei(z Zeitraum, tag time.Time, bevorstehend bool) bool {
	grenze := z.Bis
	if bevorstehend {
		grenze = z.Von.AddDate(0, 0, -1)
	}
	return kalendertagUTC(tag).After(grenze)
}

// DonnerstagVor liefert den letzten Donnerstag vor t, t selbst ausgeschlossen. An diesem Tag
// endet der Büchertausch, weil der Freitag vor den Ferien Zeugnistag ist.
func DonnerstagVor(t time.Time) time.Time {
	d := kalendertag(t).AddDate(0, 0, -1)
	for d.Weekday() != time.Thursday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

// ErsterSchultagNach liefert den ersten Schultag nach dem Zeitraum: den Montag nach den
// Ferien, wenn er kein Feiertag ist.
func ErsterSchultagNach(z Zeitraum) time.Time {
	return naechsterSchultag(kalendertag(z.Bis).AddDate(0, 0, 1), Schultage(nil))
}
