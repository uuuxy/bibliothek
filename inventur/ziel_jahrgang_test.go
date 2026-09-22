package inventur

import "testing"

func TestPruefeZielJahrgang(t *testing.T) {
	faelle := []struct {
		name       string
		lernmittel bool
		ziel       int
		ok         bool
	}{
		{"0 ist die Vorgabe, auch am Bibliotheksbuch", false, 0, true},
		{"Lernmittel bis Jahrgang 9", true, 9, true},
		{"Lernmittel bis Jahrgang 5 (Untergrenze)", true, 5, true},
		{"Lernmittel bis Jahrgang 13 (Obergrenze)", true, 13, true},
		{"Bibliotheksbuch mit Zieljahrgang", false, 9, false},
		{"Lernmittel unter der Untergrenze", true, 4, false},
		{"Lernmittel über der Obergrenze", true, 14, false},
		{"negativ", true, -1, false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			err := pruefeZielJahrgang(f.lernmittel, f.ziel)
			if (err == nil) != f.ok {
				t.Errorf("lernmittel=%v ziel=%d: err=%v, erwartet ok=%v", f.lernmittel, f.ziel, err, f.ok)
			}
		})
	}
}
