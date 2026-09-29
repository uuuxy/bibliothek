package lmfplan

import (
	"time"

	"bibliothek/pkg/schulzeit"
)

// Tagesfrist ist das Ende einer Frist, die in Tagen zählt — Buch, Medium, Gerät,
// Verlängerung, Abholfrist einer Vormerkung: ab plus tage, und fällt dieser Tag auf ein
// Wochenende, einen Feiertag oder in die Ferien, der nächste Schultag; Tagesende in der
// Schulzeitzone.
//
// Entscheidung vom 24.09.2026: Wer vor den Herbstferien ausleiht, soll nicht gemahnt
// werden, weil die Frist in die Ferien fiel. Nicht hierüber laufen Stichtage und von Hand
// gesetzte Fristen — Lernmittel (Stichtag, LMF-Plan), Ferien-Leseclub, Frist-Überschreibung,
// die Jahresfrist der Dauerleihe. Der LMF-Stichtag 31.07. liegt in jedem Jahr der Tabelle in
// den Sommerferien; über diese Regel stünde jedes Lernmittel am Tag der Bücherausgabe.
//
// Bis zum 29.09.2026 stand sie in internal/service. Die Abholfrist wird aber auch in
// repository gesetzt (Nachrücken in der Warteschlange), und dorthin reicht service nicht.
func (t Ferientabelle) Tagesfrist(ab time.Time, tage int) time.Time {
	zone := schulzeit.Zone()
	tag := t.NaechsterSchultag(ab.In(zone).AddDate(0, 0, tage))
	return schulzeit.TagesEnde(time.Date(tag.Year(), tag.Month(), tag.Day(), 12, 0, 0, 0, zone))
}
