package schulzeit

import (
	"testing"
	"time"
)

// Die Stichtage sind der 15.3. und der 15.9. — die der Bestandskartei. Geprüft wird an den
// Rändern, denn dort entscheidet sich, ob ein Abgang vom 15.9. noch in den Nachweis gehört,
// den die Schule an diesem Tag ausdruckt.
func TestHalbjahr_Raender(t *testing.T) {
	faelle := []struct {
		tag      string
		von, bis string
	}{
		{"2026-09-17", "2026-09-16", "2027-03-15"}, // heute: Winterhalbjahr
		{"2026-09-16", "2026-09-16", "2027-03-15"}, // erster Tag
		{"2026-09-15", "2026-03-16", "2026-09-15"}, // Stichtag gehört noch zum Sommerhalbjahr
		{"2026-03-16", "2026-03-16", "2026-09-15"},
		{"2026-03-15", "2025-09-16", "2026-03-15"}, // Jahreswechsel rückwärts
		{"2026-01-07", "2025-09-16", "2026-03-15"},
		{"2026-12-31", "2026-09-16", "2027-03-15"},
	}
	for _, f := range faelle {
		tag, err := time.ParseInLocation("2006-01-02", f.tag, Zone())
		if err != nil {
			t.Fatalf("Testdatum %s: %v", f.tag, err)
		}
		von, bis := Halbjahr(tag)
		if von.Format("2006-01-02") != f.von || bis.Format("2006-01-02") != f.bis {
			t.Errorf("Halbjahr(%s) = %s bis %s, erwartet %s bis %s",
				f.tag, von.Format("2006-01-02"), bis.Format("2006-01-02"), f.von, f.bis)
		}
	}
}
