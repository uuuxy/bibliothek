package lmfplan

import (
	"time"

	"bibliothek/pkg/schulzeit"
)

// Tagesfrist liefert das Ende einer Frist, die in Tagen zählt (Buch, Medium, Gerät,
// Verlängerung, Abholfrist einer Vormerkung): ab plus tage und, wenn dieser Tag auf ein
// Wochenende, einen Feiertag oder in die Ferien fällt, der nächste Schultag, am Tagesende in
// der Zone der Schule. Wer vor den Ferien ausleiht, wird so nicht gemahnt, weil die Frist in
// die Ferien fiel.
//
// Stichtage und von Hand gesetzte Fristen laufen nicht hierüber: Lernmittel (Stichtag,
// LMF-Plan), Ferien-Leseclub, eine überschriebene Frist und die Jahresfrist der Dauerleihe.
// Der Stichtag der Lernmittel am 31.07. liegt in jedem Jahr der Tabelle in den Sommerferien;
// mit dieser Regel rückte er für jedes Lernmittel auf den Tag der Bücherausgabe.
//
// Die Funktion steht in diesem Paket, weil auch repository eine Abholfrist setzt und
// internal/service nicht einbinden kann.
func (t Ferientabelle) Tagesfrist(ab time.Time, tage int) time.Time {
	zone := schulzeit.Zone()
	tag := t.ersterSchultagAb(ab.In(zone).AddDate(0, 0, tage))
	return schulzeit.TagesEnde(time.Date(tag.Year(), tag.Month(), tag.Day(), 12, 0, 0, 0, zone))
}
