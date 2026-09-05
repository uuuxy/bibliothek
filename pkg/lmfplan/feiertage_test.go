package lmfplan

import "testing"

// Osterdaten, an denen die Formel scheitert, wenn sie falsch abgeschrieben ist: 2026
// (05.04.), 2027 (28.03.), 2024 (31.03.), 2038 (25.04., der späteste mögliche Termin).
func TestOstersonntag(t *testing.T) {
	for jahr, soll := range map[int]string{2024: "2024-03-31", 2026: "2026-04-05", 2027: "2027-03-28", 2038: "2038-04-25"} {
		if got := Ostersonntag(jahr).Format("2006-01-02"); got != soll {
			t.Errorf("Ostern %d: %s, erwartet %s", jahr, got, soll)
		}
	}
}

// Die Feiertage, die in die Plan-Zeit fallen: Fronleichnam ist in Hessen frei — der
// Donnerstag, der im Juni-Plan sonst sechs Klassen bekäme. Dazu die Gegenprobe, dass
// Tage anderer Länder NICHT drinstehen.
func TestFeiertageHessen_2026(t *testing.T) {
	namen := map[string]string{}
	for _, f := range FeiertageHessen(2026) {
		namen[f.Datum.Format("2006-01-02")] = f.Name
	}
	for datum, soll := range map[string]string{
		"2026-04-03": "Karfreitag", "2026-04-06": "Ostermontag", "2026-05-14": "Christi Himmelfahrt",
		"2026-05-25": "Pfingstmontag", "2026-06-04": "Fronleichnam", "2026-10-03": "Tag der Deutschen Einheit",
	} {
		if namen[datum] != soll {
			t.Errorf("%s: %q, erwartet %q", datum, namen[datum], soll)
		}
	}
	for _, fremd := range []string{"2026-10-31", "2026-11-01", "2026-11-18", "2026-01-06", "2026-08-15"} {
		if name, ok := namen[fremd]; ok {
			t.Errorf("%s (%s) ist in Hessen kein Feiertag", fremd, name)
		}
	}
}
