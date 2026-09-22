package inventur

import "testing"

func TestPruefeMehrjahresband(t *testing.T) {
	faelle := []struct {
		name       string
		lernmittel bool
		an         bool
		von, bis   int
		ok         bool
	}{
		{"aus: immer erlaubt, auch am Bibliotheksbuch", false, false, 5, 10, true},
		{"aus: auch mit unsinniger Spanne", true, false, 9, 7, true},
		{"an: Lernmittel 7 bis 9", true, true, 7, 9, true},
		{"an: Lernmittel 5 bis 13 (Ränder)", true, true, 5, 13, true},
		{"an: Bibliotheksbuch", false, true, 7, 9, false},
		{"an: ein Jahrgang (7 bis 7) hat keine Wirkung", true, true, 7, 7, false},
		{"an: bis unter von", true, true, 9, 7, false},
		{"an: von unter 1", true, true, 0, 9, false},
		{"an: bis über 13", true, true, 7, 14, false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			err := pruefeMehrjahresband(f.lernmittel, f.an, f.von, f.bis)
			if (err == nil) != f.ok {
				t.Errorf("lernmittel=%v an=%v %d-%d: err=%v, erwartet ok=%v", f.lernmittel, f.an, f.von, f.bis, err, f.ok)
			}
		})
	}
}
