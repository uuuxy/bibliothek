package lmfplan

import "time"

// Sommerferien Hessen nach der langfristigen Sommerferienregelung der Kultusminister-
// konferenz (Beschluss vom 21.09.2022 für 2025–2030, kmk.org/service/ferienregelung).
// Peter, 06.09.2026: „Das Programm kann das doch sicherlich automatisch setzen — es
// endet immer am gleichen Tag: Donnerstags vor den Ferien zur vierten Stunde." Die
// Tabelle IST diese Automatik. Eine Schnittstelle, die ein Schulserver dafür abfragen
// sollte, gibt es nicht; die KMK beschließt in Blöcken von sechs Jahren. Läuft die
// Tabelle aus, sagt es der Planer (ok=false) und TestSommerferienHessen_Horizont
// wird zwei Jahre vorher rot — das ist die Erinnerung, den nächsten Beschluss
// nachzutragen. Fest auf Hessen wie die Feiertage (feiertage.go).
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

// SommerferienHessen nennt die Sommerferien eines Jahres (Kalendertage in UTC, Name
// „Sommerferien"); ok=false, wenn das Jahr nicht hinterlegt ist.
func SommerferienHessen(jahr int) (Zeitraum, bool) {
	z, ok := sommerferienHessen[jahr]
	return z, ok
}

// NaechsteSommerferien nennt die Sommerferien, an denen sich der nächste Plan
// ausrichtet, gesehen von heute: bevorstehend=true die nächsten, die noch nicht
// begonnen haben (vor ihnen liegt der Büchertausch); sonst die nächsten, die noch nicht
// vorbei sind (nach ihnen liegt die Bücherausgabe). Das Jahr kommt immer zurück, auch
// wenn es nicht hinterlegt ist (ok=false) — der Planer nennt es dann im Hinweis.
func NaechsteSommerferien(heute time.Time, bevorstehend bool) (Zeitraum, int, bool) {
	tag := kalendertag(heute)
	jahr := tag.Year()
	z, ok := SommerferienHessen(jahr)
	if !ok {
		return Zeitraum{}, jahr, false
	}
	vorbei := istVorbei(z, tag, bevorstehend)
	if !vorbei {
		return z, jahr, true
	}
	z, ok = SommerferienHessen(jahr + 1)
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
