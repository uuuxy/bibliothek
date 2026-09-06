package lmfplan

import (
	"strings"
	"testing"
	"time"
)

// Die Sommerferien als Einstellung: Ein eigener Eintrag verlängert die Tabelle, ein
// eigener Eintrag für ein Programmjahr gewinnt, und die Prüfung lässt keinen
// Zahlendreher durch — der Planer hängt an diesen Tagen jede Frist auf.
func TestFerientabelleAus(t *testing.T) {
	tab := FerientabelleAus(`[{"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"},{"jahr":2027,"von":"2027-07-05","bis":"2027-08-13"}]`)
	if tab.LetztesJahr() != 2031 {
		t.Errorf("letztes Jahr %d, erwartet 2031", tab.LetztesJahr())
	}
	z, ok := tab.Sommerferien(2031)
	if !ok || z.Von.Format("2006-01-02") != "2031-07-07" {
		t.Errorf("2031 fehlt oder falsch: %v %v", ok, z.Von)
	}
	// Der eigene Eintrag gewinnt über die Programmtabelle (dort beginnt 2027 am 28.06.).
	z, _ = tab.Sommerferien(2027)
	if z.Von.Format("2006-01-02") != "2027-07-05" {
		t.Errorf("eigener Eintrag 2027 gilt nicht: %s", z.Von.Format("2006-01-02"))
	}
	// Die Programmtabelle selbst bleibt unberührt.
	if z, _ := Hessen().Sommerferien(2027); z.Von.Format("2006-01-02") != "2027-06-28" {
		t.Errorf("Programmtabelle verändert: %s", z.Von.Format("2006-01-02"))
	}
	// Naechste sieht den eigenen Eintrag: Am 01.03.2031 sind die nächsten Ferien bekannt.
	heute := time.Date(2031, 3, 1, 12, 0, 0, 0, time.UTC)
	if _, jahr, ok := tab.Naechste(heute, true); jahr != 2031 || !ok {
		t.Errorf("Naechste 2031: %d/%v", jahr, ok)
	}
	if _, _, ok := Hessen().Naechste(heute, true); ok {
		t.Error("ohne Einstellung dürfte 2031 nicht bekannt sein")
	}
	// Leer und unlesbar: die Programmtabelle.
	for _, text := range []string{"", "   ", "kaputt", `[{"jahr":2031,"von":"x","bis":"y"}]`} {
		if FerientabelleAus(text).LetztesJahr() != Hessen().LetztesJahr() {
			t.Errorf("%q: nicht die Programmtabelle", text)
		}
	}
}

func TestNormalisiereSommerferien(t *testing.T) {
	// Sortiert, kompakt, ohne Rauschen.
	norm, err := NormalisiereSommerferien(` [ {"jahr":2032, "von":"2032-07-05","bis":"2032-08-13"}, {"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"} ] `)
	if err != nil {
		t.Fatal(err)
	}
	if want := `[{"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"},{"jahr":2032,"von":"2032-07-05","bis":"2032-08-13"}]`; norm != want {
		t.Errorf("Normalform:\n%s\nerwartet\n%s", norm, want)
	}
	if norm, err := NormalisiereSommerferien("  "); err != nil || norm != "" {
		t.Errorf("leer: %q %v", norm, err)
	}
	for name, text := range map[string]string{
		"unlesbar":       `{"jahr":2031}`,
		"Datum":          `[{"jahr":2031,"von":"07.07.2031","bis":"2031-08-15"}]`,
		"falsches Jahr":  `[{"jahr":2031,"von":"2030-07-07","bis":"2031-08-15"}]`,
		"Ende vor Start": `[{"jahr":2031,"von":"2031-08-15","bis":"2031-07-07"}]`,
		"zu kurz":        `[{"jahr":2031,"von":"2031-07-07","bis":"2031-07-10"}]`,
		"zu lang":        `[{"jahr":2031,"von":"2031-06-01","bis":"2031-08-31"}]`,
		"doppelt":        `[{"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"},{"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"}]`,
	} {
		if _, err := NormalisiereSommerferien(text); err == nil || !strings.Contains(err.Error(), "Sommerferien") {
			t.Errorf("%s: erwartet Fehler mit Sommerferien, bekommen %v", name, err)
		}
	}
}

func TestProgrammEintraege(t *testing.T) {
	e := ProgrammEintraege()
	if len(e) == 0 || e[0].Jahr != 2025 || e[len(e)-1].Jahr != Hessen().LetztesJahr() {
		t.Errorf("Programmeinträge: %+v", e)
	}
}

// Rasterdurchgang 06.09.2026 (Frage 3): Die Selbstprüfung fragte nach dem MAXIMUM der
// Tabelle, der Planer nach GENAU EINEM Jahr. Eine Lücke — oder ein Zahlendreher im Jahr —
// machte die Prüfung stumm, während der Planer ohne Vorgabe dastand.
func TestFerientabelle_LueckeUndZahlendreher(t *testing.T) {
	// 2032 und 2033 eingetragen, 2031 vergessen: ab 2031 reicht die Tabelle nicht.
	mit := `[{"jahr":2032,"von":"2032-07-05","bis":"2032-08-14"},{"jahr":2033,"von":"2033-07-04","bis":"2033-08-13"}]`
	tab := FerientabelleAus(mit)
	if got := tab.LetztesJahr(); got != 2033 {
		t.Errorf("Maximum: %d", got)
	}
	// Die Programmtabelle endet 2030 — lückenlos ab 2031 heißt also: gar nicht.
	if got := tab.LueckenlosBis(2031); got != 2030 {
		t.Errorf("lückenlos ab 2031: %d (die Lücke 2031 muss zählen)", got)
	}
	// Gegenprobe: mit 2031 reicht sie bis 2033.
	voll := `[{"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"},{"jahr":2032,"von":"2032-07-05","bis":"2032-08-14"},{"jahr":2033,"von":"2033-07-04","bis":"2033-08-13"}]`
	if got := FerientabelleAus(voll).LueckenlosBis(2031); got != 2033 {
		t.Errorf("ohne Lücke ab 2031: %d", got)
	}
	// Und ein Jahr, das die Programmtabelle schon kennt, hält die Kette ebenfalls.
	if got := FerientabelleAus("").LueckenlosBis(2026); got != 2030 {
		t.Errorf("Programmtabelle ab 2026: %d", got)
	}

	// Zahlendreher: 2131 statt 2031 ging durch beide Türen.
	if _, err := ParseSommerferien(`[{"jahr":2131,"von":"2131-07-07","bis":"2131-08-15"}]`); err == nil {
		t.Error("Jahr 2131 muss abgelehnt werden — sonst schweigt die Selbstprüfung hundert Jahre")
	}
	if _, err := ParseSommerferien(`[{"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"}]`); err != nil {
		t.Errorf("Gegenprobe: ein plausibles Jahr muss durchgehen: %v", err)
	}
}
