// Package lmfplan verteilt die Reihenfolge eines LMF-Plans auf Schultage und Stunden und kennt
// den Kalender dafür: Feiertage und Ferien in Hessen und die Frist, die in Tagen zählt.
//
// Der Plan der Schule ist eine Reihenfolge von Klassen: Abschlussklassen zuerst, dann jeder
// Schultag Stunde 1 bis 6, eine Zeile je Stunde, und die Reihenfolge läuft über die Tage
// weiter. Datum und Stunde rechnet der Server an einer Stelle; die Vorschau im Planer ruft
// dieselbe Tür mit "vorschau": true und rechnet nicht selbst.
//
// Zwei Dinge weichen von der Reihenfolge ab: Eine Zeile kann einen festen Platz haben (die
// Klasse mit dem Ausflug), dann fließen die anderen um sie herum. Und ein Tag kann ausfallen:
// Wochenende, gesetzlicher Feiertag, Ferien, Schließzeit oder ein freier Tag des Plans.
package lmfplan

import (
	"fmt"
	"time"
)

// Rahmen ist, was der Planer vorgibt: der erste Tag, die Stunde, mit der er beginnt, und die
// Stunden je Tag (an dieser Schule sechs).
type Rahmen struct {
	ErsterTag    time.Time
	Startstunde  int
	StundenJeTag int
}

// Platz ist Datum und Stunde einer Zeile.
type Platz struct {
	Datum  time.Time
	Stunde int
}

// VerteileMit gibt den Zeilen der Reihenfolge ihre Plätze: Zeile für Zeile eine Stunde weiter,
// nach der letzten Stunde des Tages der nächste Schultag ab Stunde 1. Ist der erste Tag kein
// Schultag, beginnt der Plan am nächsten. Was ein Schultag ist, entscheidet istSchultag.
//
// fest hat einen Eintrag je Zeile. nil heißt, die Zeile fließt; sonst ist es ihr Platz von
// Hand, und die übrigen Zeilen lassen ihn aus. Ein fester Platz darf auf einem Tag liegen, der
// sonst kein Schultag wäre: Wer ihn setzt, weiß es. Ohne feste Plätze: make([]*Platz, n).
func VerteileMit(r Rahmen, fest []*Platz, istSchultag func(time.Time) bool) []Platz {
	plaetze, belegt := festePlaetze(fest)
	if len(fest) == 0 || r.StundenJeTag < 1 {
		return plaetze[:0]
	}
	tag := naechsterSchultag(kalendertag(r.ErsterTag), istSchultag)
	stunde := r.Startstunde
	if stunde < 1 {
		stunde = 1
	}
	if stunde > r.StundenJeTag {
		// Eine Startstunde hinter dem Tagesende ließe den ersten Tag leer; der Plan beginnt
		// dann am nächsten Schultag mit Stunde 1.
		tag = naechsterSchultag(tag.AddDate(0, 0, 1), istSchultag)
		stunde = 1
	}
	weiter := func() {
		stunde++
		if stunde > r.StundenJeTag {
			tag = naechsterSchultag(tag.AddDate(0, 0, 1), istSchultag)
			stunde = 1
		}
	}
	for i := range fest {
		if fest[i] != nil {
			continue
		}
		for belegt[schluessel(Platz{Datum: tag, Stunde: stunde})] {
			weiter()
		}
		plaetze[i] = Platz{Datum: tag, Stunde: stunde}
		weiter()
	}
	return plaetze
}

// Ende ist der feste Punkt eines Plans, der vom Ende her gerechnet wird: Der Büchertausch
// endet am Donnerstag vor den Sommerferien in der vierten Stunde. Die letzte Zeile bekommt
// diesen Platz, die übrigen fließen rückwärts davor; kommen Zeilen dazu, rückt der Beginn nach
// vorn, und das Ende bleibt.
type Ende struct {
	LetzterTag   time.Time
	LetzteStunde int
	StundenJeTag int
}

// VerteileRueckwaerts ist VerteileMit vom Ende her: die letzte Zeile auf den letzten Platz,
// jede davor eine Stunde früher, vor der ersten Stunde des Tages der vorige Schultag ab seiner
// letzten Stunde. Feste Plätze gelten wie vorwärts. Ist der letzte Tag kein Schultag, endet der
// Plan am Schultag davor; eine letzte Stunde hinter dem Tagesende wird auf das Tagesende
// gezogen.
func VerteileRueckwaerts(e Ende, fest []*Platz, istSchultag func(time.Time) bool) []Platz {
	plaetze, belegt := festePlaetze(fest)
	if len(fest) == 0 || e.StundenJeTag < 1 {
		return plaetze[:0]
	}
	tag := vorigerSchultag(kalendertag(e.LetzterTag), istSchultag)
	stunde := e.LetzteStunde
	if stunde > e.StundenJeTag {
		stunde = e.StundenJeTag
	}
	if stunde < 1 {
		stunde = 1
	}
	zurueck := func() {
		stunde--
		if stunde < 1 {
			tag = vorigerSchultag(tag.AddDate(0, 0, -1), istSchultag)
			stunde = e.StundenJeTag
		}
	}
	for i := len(fest) - 1; i >= 0; i-- {
		if fest[i] != nil {
			continue
		}
		for belegt[schluessel(Platz{Datum: tag, Stunde: stunde})] {
			zurueck()
		}
		plaetze[i] = Platz{Datum: tag, Stunde: stunde}
		zurueck()
	}
	return plaetze
}

// Beginn nennt den frühesten Platz, also Datum und Stunde, an denen ein rückwärts gerechneter
// Plan beginnt; ok ist false ohne Plätze.
func Beginn(plaetze []Platz) (Platz, bool) {
	if len(plaetze) == 0 {
		return Platz{}, false
	}
	erster := plaetze[0]
	for _, p := range plaetze[1:] {
		if p.Datum.Before(erster.Datum) || (p.Datum.Equal(erster.Datum) && p.Stunde < erster.Stunde) {
			erster = p
		}
	}
	return erster, true
}

// schluessel ist der Schlüssel eines Platzes in der Map der belegten Plätze. Er ist Text und
// kein time.Time, weil zwei Zeitpunkte desselben Kalendertags aus verschiedenen Zonen als
// Schlüssel verschieden wären.
func schluessel(p Platz) string { return fmt.Sprintf("%s/%d", p.Datum.Format(time.DateOnly), p.Stunde) }

// festePlaetze legt die Ergebnisliste an, trägt die festen Plätze ein und merkt sie als
// belegt; so beginnen beide Richtungen.
func festePlaetze(fest []*Platz) ([]Platz, map[string]bool) {
	plaetze := make([]Platz, len(fest))
	belegt := map[string]bool{}
	for i, f := range fest {
		if f != nil {
			plaetze[i] = Platz{Datum: kalendertag(f.Datum), Stunde: f.Stunde}
			belegt[schluessel(plaetze[i])] = true
		}
	}
	return plaetze, belegt
}
