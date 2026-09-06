package lmfplan

import "time"

// Sommerferien Hessen nach der langfristigen Sommerferienregelung der Kultusminister-
// konferenz (Beschluss vom 21.09.2022 für 2025–2030, kmk.org/service/ferienregelung).
// Peter, 06.09.2026: „Das Programm kann das doch sicherlich automatisch setzen — es
// endet immer am gleichen Tag: Donnerstags vor den Ferien zur vierten Stunde." Die
// Tabelle IST diese Automatik. Eine Schnittstelle, die ein Schulserver dafür abfragen
// sollte, gibt es nicht; die KMK beschließt in Blöcken von sechs Jahren. Läuft die
// Tabelle aus, sagt es der Planer (ok=false); zwei Jahre vorher warnen die Selbstprüfung
// unter System → Betriebsbereitschaft (Peter, 06.09.2026: „also bekommen wir eine
// Warnung, die Termine nachzutragen?") und TestSommerferienHessen_Horizont. Seit dem
// 06.09.2026 abends trägt die Schule spätere Jahre SELBST ein (Einstellung
// „sommerferien", ferien_einstellung.go) — diese Tabelle ist die Vorbelegung, der
// Horizont-Test die Erinnerung, den nächsten Beschluss auch hier nachzutragen, damit
// eine frische Installation ihn hat. Fest auf Hessen wie die Feiertage (feiertage.go).
var sommerferienHessen = map[int]Zeitraum{
	2025: ferien("2025-07-07", "2025-08-15"),
	2026: ferien("2026-06-29", "2026-08-07"),
	2027: ferien("2027-06-28", "2027-08-06"),
	2028: ferien("2028-07-03", "2028-08-11"),
	2029: ferien("2029-07-16", "2029-08-24"),
	2030: ferien("2030-07-22", "2030-08-30"),
}

// ferien baut einen Tabelleneintrag; ein Tippfehler in der Tabelle ist ein Programm-
// fehler und fällt beim Start, nicht im Juni.
func ferien(von, bis string) Zeitraum {
	v, err := time.Parse("2006-01-02", von)
	if err != nil {
		panic("Ferientabelle: " + err.Error())
	}
	b, err := time.Parse("2006-01-02", bis)
	if err != nil {
		panic("Ferientabelle: " + err.Error())
	}
	return Zeitraum{Von: v, Bis: b, Name: "Sommerferien"}
}

// Ferientabelle sind die Sommerferien, an denen sich die Pläne ausrichten: die
// Programmtabelle oben plus das, was die Schule selbst eingetragen hat (Einstellung
// „sommerferien", ferien_einstellung.go). Ein eigener Eintrag gewinnt über die
// Programmtabelle für dasselbe Jahr. Peter, 06.09.2026: „Es muss doch dann irgendwo
// eingestellt werden" — eine Warnung, die nur ein Entwickler beheben kann, ist für
// den Betreiber keine Abhilfe.
type Ferientabelle struct {
	jahre map[int]Zeitraum
}

// Hessen ist die Programmtabelle allein — der Rückfall, wenn nichts eingestellt ist.
func Hessen() Ferientabelle {
	return Ferientabelle{jahre: sommerferienHessen}
}

// Mit ergänzt die Tabelle um eigene Einträge (gleiches Jahr: der eigene gilt).
func (t Ferientabelle) Mit(eintraege []SommerferienEintrag) Ferientabelle {
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

// LetztesJahr ist das letzte Jahr, für das Sommerferien bekannt sind — die
// Selbstprüfung (System → Betriebsbereitschaft) warnt zwei Jahre vor dem Ende.
func (t Ferientabelle) LetztesJahr() int {
	letztes := 0
	for jahr := range t.jahre {
		letztes = max(letztes, jahr)
	}
	return letztes
}

// Sommerferien nennt die Sommerferien eines Jahres (Kalendertage in UTC, Name
// „Sommerferien"); ok=false, wenn das Jahr nicht bekannt ist.
func (t Ferientabelle) Sommerferien(jahr int) (Zeitraum, bool) {
	z, ok := t.jahre[jahr]
	return z, ok
}

// Naechste nennt die Sommerferien, an denen sich der nächste Plan ausrichtet, gesehen
// von heute: bevorstehend=true die nächsten, die noch nicht begonnen haben (vor ihnen
// liegt der Büchertausch); sonst die nächsten, die noch nicht vorbei sind (nach ihnen
// liegt die Bücherausgabe). Das Jahr kommt immer zurück, auch wenn es nicht bekannt
// ist (ok=false) — der Planer nennt es dann im Hinweis.
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

// istVorbei: Sind diese Ferien für den Zweck schon vorbei — begonnen (bevorstehend)
// oder beendet? Verglichen als Kalendertage, unabhängig von der Zone.
func istVorbei(z Zeitraum, tag time.Time, bevorstehend bool) bool {
	grenze := z.Bis
	if bevorstehend {
		grenze = z.Von.AddDate(0, 0, -1)
	}
	return kalendertagUTC(tag).After(grenze)
}

// DonnerstagVor liefert den letzten Donnerstag vor t (t selbst ausgeschlossen) — der
// Tag, an dem der Büchertausch der Schule endet: Freitag vor den Ferien ist Zeugnistag.
func DonnerstagVor(t time.Time) time.Time {
	d := kalendertag(t).AddDate(0, 0, -1)
	for d.Weekday() != time.Thursday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

// ErsterSchultagNach liefert den ersten Schultag nach dem Zeitraum (Montag nach den
// Ferien, es sei denn, er ist ein Feiertag).
func ErsterSchultagNach(z Zeitraum) time.Time {
	return naechsterSchultag(kalendertag(z.Bis).AddDate(0, 0, 1), Schultage(nil))
}

// kalendertagUTC ist der Kalendertag von t als UTC-Zeitpunkt — für Vergleiche mit den
// UTC-Tagen der Tabellen (Ferien, Feiertage), ohne dass die Zone das Datum verschiebt.
func kalendertagUTC(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
