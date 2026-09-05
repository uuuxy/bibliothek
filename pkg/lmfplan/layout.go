// Package lmfplan gießt die Reihenfolge eines LMF-Plans auf Schultage und Stunden.
//
// Der Plan der Schule (Peter, 05.09.2026, Excel „lmf termine 26") ist eine Reihenfolge
// von Klassen: Abschlussklassen zuerst, dann jeder Schultag Stunde 1–6, eine Zeile je
// Stunde, die Reihenfolge läuft über die Tage weiter. Wochentag und Datum standen im
// Excel von Hand — und zweimal falsch. Hier rechnet sie eine Funktion, und zwar die
// EINE: Der Server legt den Plan damit ab, und die Vorschau im Planer ruft denselben
// Server (PUT … "vorschau": true) statt eines JavaScript-Zwillings.
package lmfplan

import "time"

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

// Zeitraum ist ein geschlossenes Datumsintervall (Ferien, Schließzeit).
type Zeitraum struct {
	Von, Bis time.Time
}

// Verteile gibt den n Zeilen der Reihenfolge ihre Plätze: Zeile für Zeile eine Stunde
// weiter, nach der letzten Stunde des Tages der nächste Schultag ab Stunde 1. Ist der
// erste Tag kein Schultag, beginnt der Plan am nächsten. istSchultag entscheidet, was
// ein Schultag ist (Wochenende und Ferien sind keine).
func Verteile(r Rahmen, n int, istSchultag func(time.Time) bool) []Platz {
	plaetze := make([]Platz, 0, n)
	if n == 0 || r.StundenJeTag < 1 {
		return plaetze
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
	for i := 0; i < n; i++ {
		plaetze = append(plaetze, Platz{Datum: tag, Stunde: stunde})
		stunde++
		if stunde > r.StundenJeTag {
			tag = naechsterSchultag(tag.AddDate(0, 0, 1), istSchultag)
			stunde = 1
		}
	}
	return plaetze
}

// Schultage liefert die Regel „Montag bis Freitag und nicht in einem der Zeiträume".
func Schultage(frei []Zeitraum) func(time.Time) bool {
	return func(t time.Time) bool {
		if wt := t.Weekday(); wt == time.Saturday || wt == time.Sunday {
			return false
		}
		d := kalendertag(t)
		for _, z := range frei {
			if !d.Before(kalendertag(z.Von)) && !d.After(kalendertag(z.Bis)) {
				return false
			}
		}
		return true
	}
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
