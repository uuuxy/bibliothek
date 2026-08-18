package service

import "testing"

// Die beiden Referenzfälle sind ECHTE Scans aus der Schulbibliothek (18.08.2026),
// keine konstruierten Werte — die Regel muss genau diese Messungen erklären.
func TestDekodiereLitteraEtikett(t *testing.T) {
	faelle := []struct {
		scan   string
		nummer string
		ok     bool
		grund  string
	}{
		{"5896800039556", "58968", true, "echter Scan, 5-stellige Mediennummer"},
		{"1241170039561", "124117", true, "echter Scan, 6-stellige Mediennummer"},
		{"5896800039557", "", false, "falsche EAN-Prüfziffer"},
		{"9783464034408", "", false, "Verlags-ISBN (echt, gültige EAN): Stellenzahl 0 → kein Etikett"},
		{"589680003955", "", false, "nur 12 Stellen"},
		{"58968000395568", "", false, "14 Stellen"},
		{"B97601826457", "", false, "Schülerausweis-Herstellernummer, keine reine Zahl"},
		{"5896810039553", "", false, "Polsterung verletzt: Ziffer im Null-Bereich (EAN selbst gültig)"},
		{"5896800039501", "", false, "Stellenzahl 0 ist unzulässig (EAN selbst gültig)"},
		{"0896800039551", "", false, "Mediennummer mit führender Null (EAN selbst gültig)"},
		{"", "", false, "leer"},
	}
	for _, f := range faelle {
		nummer, ok := dekodiereLitteraEtikett(f.scan)
		if ok != f.ok || nummer != f.nummer {
			t.Errorf("%s (%q): got (%q, %v), erwartet (%q, %v)", f.grund, f.scan, nummer, ok, f.nummer, f.ok)
		}
	}
}
