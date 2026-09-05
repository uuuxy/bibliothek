// Package lmfplan gießt die Reihenfolge eines LMF-Plans auf Schultage und Stunden.
//
// Der Plan der Schule (Peter, 05.09.2026, Excel „lmf termine 26") ist eine Reihenfolge
// von Klassen: Abschlussklassen zuerst, dann jeder Schultag Stunde 1–6, eine Zeile je
// Stunde, die Reihenfolge läuft über die Tage weiter. Wochentag und Datum standen im
// Excel von Hand — und zweimal falsch. Hier rechnet sie eine Funktion, und zwar die
// EINE: Der Server legt den Plan damit ab, und die Vorschau im Planer ruft denselben
// Server (PUT … "vorschau": true) statt eines JavaScript-Zwillings.
//
// Zwei Ausnahmen von der reinen Reihenfolge (Peter, 05.09.2026 abends): Eine Zeile kann
// einen FESTEN Platz haben (die Klasse mit dem Ausflug), dann fließen die anderen um
// sie herum; und ein Tag kann ausfallen — Wochenende, gesetzlicher Feiertag (Hessen,
// feiertage.go), Ferien/Schließzeit, freier Tag des Plans.
package lmfplan

import (
	"fmt"
	"time"
)

// Rahmen ist, was der Planer vorgibt: erster Tag, Startstunde am ersten Tag (der
// Donnerstag im Juni begann in der 3. Stunde), Stunden je Tag (bei dieser Schule 6).
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

// Zeitraum ist ein geschlossenes Datumsintervall (Ferien, Schließzeit, freier Tag);
// Name ist der Grund, der im Planer steht („Pfingstferien", „Pädagogischer Tag").
type Zeitraum struct {
	Von, Bis time.Time
	Name     string
}

// Ausfall ist ein Werktag, an dem der Plan nicht läuft, mit seinem Grund.
type Ausfall struct {
	Datum time.Time
	Grund string
}

// VerteileMit gibt den Zeilen der Reihenfolge ihre Plätze: Zeile für Zeile eine Stunde
// weiter, nach der letzten Stunde des Tages der nächste Schultag ab Stunde 1. Ist der
// erste Tag kein Schultag, beginnt der Plan am nächsten. istSchultag entscheidet, was
// ein Schultag ist (Wochenende, Feiertage und Ferien sind keine).
//
// fest hat einen Eintrag je Zeile: nil = die Zeile fließt; sonst ist das ihr Platz
// (Datum und Stunde von Hand), und die übrigen Zeilen lassen ihn aus — die Klasse mit
// dem Ausflug nimmt ihre Stunde mit, und niemand wird auf dieselbe gelegt. Ein fester
// Platz darf auf einem Tag liegen, der sonst kein Schultag wäre: Wer ihn setzt, weiß es.
// Ohne feste Plätze: make([]*Platz, n).
func VerteileMit(r Rahmen, fest []*Platz, istSchultag func(time.Time) bool) []Platz {
	n := len(fest)
	plaetze := make([]Platz, n)
	if n == 0 || r.StundenJeTag < 1 {
		return plaetze[:0]
	}
	// Schlüssel als Text, nicht als time.Time: Zwei Zeitpunkte desselben Kalendertags
	// aus verschiedenen Zonen wären als Map-Schlüssel verschieden.
	schluessel := func(p Platz) string { return fmt.Sprintf("%s/%d", p.Datum.Format("2006-01-02"), p.Stunde) }
	belegt := map[string]bool{}
	for i, f := range fest {
		if f != nil {
			plaetze[i] = Platz{Datum: kalendertag(f.Datum), Stunde: f.Stunde}
			belegt[schluessel(plaetze[i])] = true
		}
	}
	tag := naechsterSchultag(kalendertag(r.ErsterTag), istSchultag)
	stunde := r.Startstunde
	if stunde < 1 {
		stunde = 1
	}
	if stunde > r.StundenJeTag {
		// Eine Startstunde hinter dem Tagesende hieße: ein leerer erster Tag. Dann eben
		// der nächste Schultag von vorn.
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

// kalender kennt die Gründe, aus denen ein Tag kein Schultag ist. Feiertage werden je
// Jahr einmal gerechnet und gemerkt — die Suche nach dem nächsten Schultag fragt Tag
// für Tag.
type kalender struct {
	frei      []Zeitraum
	feiertage map[int][]Feiertag
}

func neuerKalender(frei []Zeitraum) *kalender {
	return &kalender{frei: frei, feiertage: map[int][]Feiertag{}}
}

// grund nennt, warum t kein Schultag ist — leer, wenn er einer ist. Reihenfolge:
// Wochenende, gesetzlicher Feiertag, dann die Zeiträume in ihrer Reihenfolge.
func (k *kalender) grund(t time.Time) string {
	if wt := t.Weekday(); wt == time.Saturday || wt == time.Sunday {
		return "Wochenende"
	}
	jahr := t.Year()
	if _, ok := k.feiertage[jahr]; !ok {
		k.feiertage[jahr] = FeiertageHessen(jahr)
	}
	for _, f := range k.feiertage[jahr] {
		if istGleicherTag(f.Datum, t) {
			return f.Name
		}
	}
	d := kalendertag(t)
	for _, z := range k.frei {
		if !d.Before(kalendertag(z.Von)) && !d.After(kalendertag(z.Bis)) {
			if z.Name == "" {
				return "frei"
			}
			return z.Name
		}
	}
	return ""
}

// Schultage liefert die Regel „Montag bis Freitag, kein gesetzlicher Feiertag (Hessen)
// und nicht in einem der Zeiträume".
func Schultage(frei []Zeitraum) func(time.Time) bool {
	k := neuerKalender(frei)
	return func(t time.Time) bool { return k.grund(t) == "" }
}

// Ausfaelle listet die Werktage von von bis bis (einschließlich), die keine Schultage
// sind, mit Grund — das, was der Planer neben dem Rahmen zeigt, damit ein fehlender
// Donnerstag im Plan erklärt ist. Wochenenden stehen nicht drin, die erklärt niemand.
func Ausfaelle(von, bis time.Time, frei []Zeitraum) []Ausfall {
	k := neuerKalender(frei)
	ausfaelle := []Ausfall{}
	for d := kalendertag(von); !d.After(kalendertag(bis)); d = d.AddDate(0, 0, 1) {
		if g := k.grund(d); g != "" && g != "Wochenende" {
			ausfaelle = append(ausfaelle, Ausfall{Datum: d, Grund: g})
		}
	}
	return ausfaelle
}

// naechsterSchultag liefert t selbst, wenn es ein Schultag ist, sonst den nächsten.
// Nach einem Jahr ohne Schultag gibt es keinen — dann bleibt es beim Kalender, statt
// endlos zu laufen (ein Ferien-Eintrag „bis 9999" ist ein Datenfehler, kein Endlos-Plan).
func naechsterSchultag(t time.Time, istSchultag func(time.Time) bool) time.Time {
	for i := 0; i < 366; i++ {
		if istSchultag(t) {
			return t
		}
		t = t.AddDate(0, 0, 1)
	}
	return t
}

// kalendertag schneidet die Uhrzeit ab und behält die Zeitzone.
func kalendertag(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
