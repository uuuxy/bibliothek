package auskunft

import "testing"

// Die Auskunft nach Art. 15 nennt über der Ausleihhistorie die Zahl der Einträge; bei einem
// einzigen stand „(1 Einträge)" auf dem Blatt.
func TestAusleihUeberschrift_EinzahlUndMehrzahl(t *testing.T) {
	for n, soll := range map[int]string{
		0: "3. Ausleihhistorie (0 Einträge)",
		1: "3. Ausleihhistorie (1 Eintrag)",
		4: "3. Ausleihhistorie (4 Einträge)",
	} {
		if ist := ausleihUeberschrift(n); ist != soll {
			t.Errorf("ausleihUeberschrift(%d) = %q, erwartet %q", n, ist, soll)
		}
	}
}
