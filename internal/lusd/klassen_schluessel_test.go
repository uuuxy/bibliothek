package lusd_test

import (
	"testing"

	"bibliothek/internal/lusd"
)

// Der Klassenschlüssel ist ein Fixpunkt: Ändert ein zweiter Durchgang noch etwas, hat der erste
// nur zur Hälfte normalisiert, etwa eine von zwei führenden Nullen. Ohne -fuzz laufen nur die
// Startwerte; wie gesucht wird, steht in docs/ARCHITEKTUR.md 8.14.
func FuzzKlassenSchluessel_ZweimalIstEinmal(f *testing.F) {
	for _, in := range []string{"05A", " 5a ", "5 a", "007b", "0", "00", "0a", "10 R", "\t5a"} {
		f.Add(in)
	}
	f.Fuzz(func(t *testing.T, in string) {
		einmal := lusd.KlassenSchluessel(in)
		if zweimal := lusd.KlassenSchluessel(einmal); zweimal != einmal {
			t.Errorf("lusd.KlassenSchluessel(%q) = %q, noch einmal %q", in, einmal, zweimal)
		}
	})
}
