package lmfplan

import (
	"testing"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Die Fälle, an denen eine Leihfrist hängt (Entscheidung vom 24.09.2026): erster und
// letzter Ferientag, ein Feiertag am Wochenende vor den Ferien, Weihnachtsferien, die an
// einem Dienstag enden, Ostern bis Ostermontag, Sommerferien aus der Tabelle und aus einem
// eigenen Eintrag.
func TestNaechsterSchultag(t *testing.T) {
	eigene := FerientabelleAus(`[{"jahr":2031,"von":"2031-07-14","bis":"2031-08-22"}]`)
	for _, c := range []struct {
		name      string
		tabelle   Ferientabelle
		tag, soll string
	}{
		{"Schultag bleibt", Hessen(), "2026-10-02", "2026-10-02"},
		{"Samstag: der Montag", Hessen(), "2026-09-26", "2026-09-28"},
		{"erster Herbstferientag", Hessen(), "2026-10-05", "2026-10-19"},
		{"letzter Herbstferientag (Samstag)", Hessen(), "2026-10-17", "2026-10-19"},
		{"Feiertag am Samstag, danach Ferien", Hessen(), "2026-10-03", "2026-10-19"},
		{"Weihnachtsferien enden am Dienstag", Hessen(), "2027-01-12", "2027-01-13"},
		{"Christi Himmelfahrt", Hessen(), "2027-05-06", "2027-05-07"},
		{"Osterferien bis Ostermontag", Hessen(), "2030-04-22", "2030-04-23"},
		{"Sommerferien der Tabelle", Hessen(), "2027-06-28", "2027-08-09"},
		{"Sommerferien aus eigenem Eintrag", eigene, "2031-07-14", "2031-08-25"},
		{"ohne eigenen Eintrag kennt 2031 keine Sommerferien", Hessen(), "2031-07-14", "2031-07-14"},
	} {
		if got := c.tabelle.NaechsterSchultag(tag(c.tag)); !got.Equal(tag(c.soll)) {
			t.Errorf("%s: %s → %s, erwartet %s", c.name, c.tag, got.Format("2006-01-02"), c.soll)
		}
	}
}

// Die Frist entsteht als Berliner Zeitpunkt. Mitternacht in Berlin liegt zwei Stunden vor
// Mitternacht UTC — direkt gegen die UTC-Tage der Tabelle verglichen, fiele der erste
// Ferientag heraus, und die Frist bliebe mitten in den Ferien stehen.
func TestNaechsterSchultag_BerlinerMitternacht(t *testing.T) {
	berlin := schulzeit.Zone()
	for _, zeitpunkt := range []time.Time{
		time.Date(2026, 10, 5, 0, 0, 0, 0, berlin),
		time.Date(2026, 10, 5, 23, 59, 59, 0, berlin),
	} {
		if got := Hessen().NaechsterSchultag(zeitpunkt); !got.Equal(tag("2026-10-19")) {
			t.Errorf("%s: %s, erwartet 2026-10-19", zeitpunkt, got.Format("2006-01-02"))
		}
	}
}

// Die Tabelle von Hand abgetippt: Jedes Schuljahr hat Herbst-, Weihnachts- und Osterferien
// in dieser Reihenfolge, lückenlos, ohne Überschneidung — auch nicht mit den Sommerferien.
// Ein Zahlendreher im Jahr oder ein vergessenes Schuljahr fällt hier auf.
func TestUebrigeFerienHessen_Tabelle(t *testing.T) {
	namen := []string{"Herbstferien", "Weihnachtsferien", "Osterferien"}
	for i, z := range uebrigeFerienHessen {
		if z.Name != namen[i%3] {
			t.Errorf("Eintrag %d heißt %q, erwartet %q", i, z.Name, namen[i%3])
		}
		if z.Bis.Before(z.Von) {
			t.Errorf("%s ab %s endet vor dem Beginn", z.Name, z.Von.Format("2006-01-02"))
		}
		if tage := z.Bis.Sub(z.Von).Hours()/24 + 1; tage < 10 || tage > 21 {
			t.Errorf("%s ab %s: %v Tage", z.Name, z.Von.Format("2006-01-02"), tage)
		}
		if i > 0 && !z.Von.After(uebrigeFerienHessen[i-1].Bis) {
			t.Errorf("%s ab %s überschneidet den Eintrag davor", z.Name, z.Von.Format("2006-01-02"))
		}
		schuljahr := z.Von.Year()
		if z.Name == "Osterferien" {
			schuljahr--
		}
		if soll := 2026 + i/3; schuljahr != soll {
			t.Errorf("%s ab %s gehört ins Schuljahr %d/%d, erwartet %d/%d", z.Name,
				z.Von.Format("2006-01-02"), schuljahr, schuljahr+1, soll, soll+1)
		}
		if s, ok := Hessen().Sommerferien(z.Von.Year()); ok && !z.Bis.Before(s.Von) && !z.Von.After(s.Bis) {
			t.Errorf("%s ab %s überschneidet die Sommerferien", z.Name, z.Von.Format("2006-01-02"))
		}
	}
	if len(uebrigeFerienHessen)%3 != 0 {
		t.Errorf("%d Einträge — ein Schuljahr ist unvollständig", len(uebrigeFerienHessen))
	}
}

// Wie TestSommerferienHessen_Horizont: zwei Jahre vor dem Ende rot, damit die nächsten
// Termine des Ministeriums nachgetragen werden, bevor die Leihfristen sie brauchen.
func TestUebrigeFerienHessen_Horizont(t *testing.T) {
	if letztes := UebrigeFerienBis(); time.Now().Year()+2 > letztes {
		t.Errorf("Herbst-, Weihnachts- und Osterferien Hessen sind nur bis %d hinterlegt — "+
			"Termine von kultus.hessen.de/schulsystem/ferien/ferientermine nachtragen "+
			"(pkg/lmfplan/schulferien.go)", letztes)
	}
}
