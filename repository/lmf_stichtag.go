package repository

// lmf_stichtag.go — der globale LMF-Stichtag („MM-TT", Vorgabe 31.07.): EINE Rechnung.
//
// Bis zum 06.09.2026 stand sie zweimal im Haus, und die beiden Fassungen waren sich nur
// bei der Vorgabe einig:
//
//   - internal/service (Ausleihe) rechnete `jahr := now.Year(); if now.Month() >= August
//     { jahr++ }`. Das stimmt nur, solange der Stichtag in die ZWEITE Hälfte des
//     Schuljahres fällt (Januar–Juli). Ein Stichtag im September ergab dort den 30.09.
//     des FOLGENDEN Schuljahres — dreizehn Monate Frist.
//   - api (Rückweg des Plans) rechnete über SchuljahrBeginn und lag damit richtig; der
//     Kommentar daneben behauptete „dieselbe Rechnung wie die Fristberechnung beim
//     Ausleihen". Sie war es nicht.
//
// Dasselbe Schulbuch hätte je nach Weg eine um ein Jahr verschiedene Frist bekommen:
// beim Ausleihen die eine, beim Herausnehmen der Klasse aus dem Plan die andere.
// Bugklasse „Doppelte Wahrheitsquelle" (docs/sweeps.md).
//
// Geblieben sind ZWEI Fragen an dieselbe Rechnung, weil die Pfade wirklich Verschiedenes
// wissen wollen — die eine ist über die andere gebaut, nicht neben sie:
//
//   - LmfStichtagImSchuljahr: „Welcher Tag ist der Stichtag DIESES Schuljahres?" Das ist
//     die Frage des Rückwegs: Eine Klasse verliert ihren Termin, ihre Fristen gehen an
//     den Stichtag des Schuljahres zurück, in dem der Termin lag.
//   - LmfStichtagAbTag: „Welcher Stichtag liegt noch vor mir?" Das ist die Frage der
//     Ausleihe: Eine Frist, die in der Vergangenheit liegt, ist keine Frist, sondern
//     eine Mahnung am Tag der Ausgabe.

import (
	"strconv"
	"strings"
	"time"

	"bibliothek/pkg/schulzeit"
)

// StandardLmfStichtag ist die Vorgabe: 31. Juli, das Ende des hessischen Schuljahres.
// Sie steht hier und wird von den Einstellungen, ihrem Zurücksetzen und den
// Notfallwerten der Ausleihe gelesen — vorher stand das Literal an fünf Stellen.
const StandardLmfStichtag = "07-31"

// lmfStichtagMonatTag zerlegt „MM-TT". Unlesbares fällt auf die Vorgabe zurück: Der
// Wert kommt aus einer Einstellung, und eine kaputte Einstellung darf keine Ausleihe
// verhindern. Ein leeres Feld setzt die Einstellungs-Ebene ohnehin auf die Vorgabe.
func lmfStichtagMonatTag(stichtag string) (time.Month, int) {
	monat, tagImMonat := time.July, 31
	teile := strings.SplitN(stichtag, "-", 2)
	if len(teile) != 2 {
		return monat, tagImMonat
	}
	m, err1 := strconv.Atoi(teile[0])
	d, err2 := strconv.Atoi(teile[1])
	if err1 == nil && err2 == nil && m >= 1 && m <= 12 && d >= 1 && d <= 31 {
		monat, tagImMonat = time.Month(m), d
	}
	return monat, tagImMonat
}

// LmfStichtagImSchuljahr legt den Stichtag in das Schuljahr eines Tages: die erste
// Wiederkehr von „MM-TT" ab dem 1. August dieses Schuljahres. Für die Vorgabe 07-31 ist
// das immer der letzte Tag des Schuljahres; für einen Stichtag ab August der Termin im
// ERSTEN Kalenderjahr des Schuljahres.
//
// Die Uhrzeit ist 12:00 der Schulzeitzone — ein Mittagswert, der bei keiner
// Sommerzeitumstellung auf den Nachbartag kippt. Zur Frist wird daraus erst
// service.TagesEndeInSchulzeitzone (23:59:59); diese Funktion liefert einen KALENDERTAG,
// keine Frist.
func LmfStichtagImSchuljahr(tag time.Time, stichtag string) time.Time {
	monat, tagImMonat := lmfStichtagMonatTag(stichtag)
	beginn := SchuljahrBeginn(tag)
	jahr := beginn.Year() + 1
	if monat >= time.August {
		jahr = beginn.Year()
	}
	return time.Date(jahr, monat, tagImMonat, 12, 0, 0, 0, schulzeit.Zone())
}

// LmfStichtagAbTag liefert den nächsten Stichtag, der nicht vor dem Tag liegt: den des
// laufenden Schuljahres, und wenn der schon vorbei ist, den des nächsten. Verglichen
// werden Kalendertage — wer AM Stichtag ausleiht, bekommt ihn noch (bis 23:59:59), nicht
// erst in einem Jahr.
func LmfStichtagAbTag(tag time.Time, stichtag string) time.Time {
	tag = tag.In(schulzeit.Zone())
	imSchuljahr := LmfStichtagImSchuljahr(tag, stichtag)
	heute := time.Date(tag.Year(), tag.Month(), tag.Day(), 0, 0, 0, 0, schulzeit.Zone())
	if imSchuljahr.Before(heute) {
		imSchuljahr = LmfStichtagImSchuljahr(SchuljahrBeginn(tag).AddDate(1, 0, 0), stichtag)
	}
	return imSchuljahr
}
