package lmfplan

import "time"

// Zeitraum ist eine Spanne von Tagen, beide Enden eingeschlossen: Ferien, Schließzeit oder ein
// freier Tag. Name ist der Grund, der im Planer steht („Pfingstferien", „Pädagogischer Tag").
type Zeitraum struct {
	Von, Bis time.Time
	Name     string
}

// Ausfall ist ein Werktag, an dem der Plan nicht läuft, mit seinem Grund.
type Ausfall struct {
	Datum time.Time
	Grund string
}

// kalender kennt die Gründe, aus denen ein Tag kein Schultag ist. Die Feiertage eines Jahres
// rechnet er einmal und merkt sie, weil die Suche nach dem nächsten Schultag Tag für Tag fragt.
type kalender struct {
	frei      []Zeitraum
	feiertage map[int][]feiertag
}

func neuerKalender(frei []Zeitraum) *kalender {
	return &kalender{frei: frei, feiertage: map[int][]feiertag{}}
}

// grund nennt, warum t kein Schultag ist, und ist leer, wenn er einer ist. Geprüft wird in
// dieser Folge: Wochenende, gesetzlicher Feiertag, dann die Zeiträume in ihrer Reihenfolge.
func (k *kalender) grund(t time.Time) string {
	if wt := t.Weekday(); wt == time.Saturday || wt == time.Sunday {
		return "Wochenende"
	}
	jahr := t.Year()
	if _, ok := k.feiertage[jahr]; !ok {
		k.feiertage[jahr] = feiertageHessen(jahr)
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

// Schultage liefert die Regel für einen Schultag: Montag bis Freitag, kein gesetzlicher
// Feiertag in Hessen und kein Tag in einem der Zeiträume.
func Schultage(frei []Zeitraum) func(time.Time) bool {
	k := neuerKalender(frei)
	return func(t time.Time) bool { return k.grund(t) == "" }
}

// Ausfaelle nennt die Werktage von von bis bis (beide eingeschlossen), die keine Schultage
// sind, mit ihrem Grund. Der Planer zeigt sie neben dem Rahmen, damit ein fehlender Donnerstag
// im Plan erklärt ist; Wochenenden stehen nicht darin.
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
func naechsterSchultag(t time.Time, istSchultag func(time.Time) bool) time.Time {
	return schultagAb(t, 1, istSchultag)
}

// vorigerSchultag liefert t selbst, wenn es ein Schultag ist, sonst den davor.
func vorigerSchultag(t time.Time, istSchultag func(time.Time) bool) time.Time {
	return schultagAb(t, -1, istSchultag)
}

// schultagAb sucht von t aus in Schritten von schritt Tagen den ersten Schultag. Die Suche
// endet nach einem Jahr, damit ein Zeitraum „bis 9999" sie nicht endlos laufen lässt; ohne
// Schultag kommt der Tag zurück, an dem sie endet.
func schultagAb(t time.Time, schritt int, istSchultag func(time.Time) bool) time.Time {
	for i := 0; i < 366; i++ {
		if istSchultag(t) {
			return t
		}
		t = t.AddDate(0, 0, schritt)
	}
	return t
}

// istGleicherTag vergleicht zwei Zeitpunkte als Kalendertage, unabhängig von der Zone.
func istGleicherTag(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

// kalendertag schneidet die Uhrzeit ab und behält die Zeitzone.
func kalendertag(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// kalendertagUTC liefert den Kalendertag von t als Zeitpunkt in UTC. So lässt er sich mit den
// Tagen der Tabellen (Ferien, Feiertage) vergleichen, ohne dass die Zone das Datum verschiebt.
func kalendertagUTC(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
