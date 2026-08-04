package littera

import "testing"

// Die Abschlussklassen der Schule sind 9H, 10R und 13 (Gymnasium/Oberstufe) — dieselbe
// Regel, die api/student_promotion.go beim Schuljahreswechsel anwendet. Liefen die
// beiden auseinander, haette ein importierter Schueler ein anderes Abgangsjahr als
// derselbe Schueler nach dem ersten Versetzungslauf.
func TestAbgaengerJahr(t *testing.T) {
	const schuljahrEnde = 2027 // Schuljahr 2026/27

	faelle := []struct {
		klasse   string
		erwartet int
		ok       bool
		warum    string
	}{
		// Hauptschulzweig endet nach 9.
		{"09H1", 2027, true, "steht in der Abschlussklasse, geht dieses Jahr"},
		{"07H1", 2029, true, "noch zwei Jahre bis 9H"},
		{"05H2", 2031, true, "vier Jahre bis 9H"},

		// Realschulzweig endet nach 10.
		{"10R1", 2027, true, "Abschlussklasse"},
		{"08R2", 2029, true, "zwei Jahre bis 10R"},

		// Gymnasium und Oberstufe enden nach 13.
		{"13T5", 2027, true, "Abschlussklasse der Oberstufe"},
		{"10G4", 2030, true, "drei Jahre bis 13"},
		{"12T3", 2028, true, "ein Jahr bis 13"},

		// Foerderstufe: Zweig steht noch nicht fest, laengster Weg wird angesetzt.
		// Ein zu frueher Wert wuerde einen Schueler archivieren, der noch da ist.
		{"05F1", 2035, true, "Foerderstufe rechnet vorsichtig bis 13"},

		// Wiederholer ueber dem Abschlussjahrgang: geht am Ende DIESES Jahres,
		// nicht rueckwirkend.
		{"10H1", 2027, true, "ueber dem Abschlussjahrgang — nicht in die Vergangenheit"},

		// Nichts zu holen: kein Wert erfinden.
		{"Lehrer", 0, false, "keine Klasse"},
		{"Ab", 0, false, "keine Klasse"},
		{"07X1", 0, false, "unbekannter Zweig"},
		{"", 0, false, "leer"},
	}

	for _, f := range faelle {
		jahr, ok := AbgaengerJahr(f.klasse, schuljahrEnde)
		if ok != f.ok {
			t.Errorf("AbgaengerJahr(%q): ok=%v, erwartet %v (%s)", f.klasse, ok, f.ok, f.warum)
			continue
		}
		if ok && jahr != f.erwartet {
			t.Errorf("AbgaengerJahr(%q) = %d, erwartet %d (%s)", f.klasse, jahr, f.erwartet, f.warum)
		}
	}
}

func TestIstAbschlussklasse(t *testing.T) {
	abschluss := []string{"09H1", "10R2", "13T5", "13G1"}
	for _, k := range abschluss {
		if !IstAbschlussklasse(k) {
			t.Errorf("%q muss Abschlussklasse sein", k)
		}
	}
	weiter := []string{"07H1", "09R1", "12T3", "05F1", "10G4"}
	for _, k := range weiter {
		if IstAbschlussklasse(k) {
			t.Errorf("%q ist KEINE Abschlussklasse", k)
		}
	}
}

// TestAbschlussJahrgang haelt die Zweig-Zuordnung einzeln fest — sie ist die Stelle,
// an der eine Schulform-Aenderung ansetzen wuerde.
func TestAbschlussJahrgang(t *testing.T) {
	faelle := map[string]int{
		"H1": 9, "h2": 9,
		"R1": 10, "r3": 10,
		"G4": 13, "T5": 13, "F1": 13,
	}
	for zweig, erwartet := range faelle {
		got, ok := AbschlussJahrgang(zweig)
		if !ok || got != erwartet {
			t.Errorf("AbschlussJahrgang(%q) = %d/%v, erwartet %d", zweig, got, ok, erwartet)
		}
	}
	if _, ok := AbschlussJahrgang("X1"); ok {
		t.Error("unbekannter Zweig darf nicht auflösen")
	}
	if _, ok := AbschlussJahrgang(""); ok {
		t.Error("leerer Zweig darf nicht auflösen")
	}
}
