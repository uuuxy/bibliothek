package littera

import (
	"testing"
	"time"
)

// TestOptionenVorgabe prüft die Vorgabewerte für einen Lauf.
func TestOptionenVorgabe(t *testing.T) {
	jetzt := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	opt := OptionenVorgabe(jetzt)

	if opt.Barcodes != BarcodeLittera {
		t.Errorf("Barcodes = %q, erwartet %q", opt.Barcodes, BarcodeLittera)
	}
	if opt.BatchGroesse != 200 {
		t.Errorf("BatchGroesse = %d, erwartet 200", opt.BatchGroesse)
	}
	if opt.LehrkraefteInaktiv != false {
		t.Errorf("LehrerInaktiv = %t, erwartet false", opt.LehrkraefteInaktiv)
	}
	if !opt.Jetzt.Equal(jetzt) {
		t.Errorf("Jetzt = %v, erwartet %v", opt.Jetzt, jetzt)
	}
}

// TestOptionenVorgabeSchuljahr: das Schuljahr endet im Sommer. Ab August zählt bereits
// das nächste Kalenderjahr — dieselbe Auslegung wie im Versetzungslauf. Läge die Grenze
// falsch, bekämen alle importierten Schüler ein um ein Jahr verschobenes Abgangsjahr.
func TestOptionenVorgabeSchuljahr(t *testing.T) {
	faelle := []struct {
		jetzt time.Time
		ende  int
	}{
		{time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC), 2026},
		{time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), 2027},
		{time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC), 2027},
	}
	for _, f := range faelle {
		if got := OptionenVorgabe(f.jetzt).SchuljahrEnde; got != f.ende {
			t.Errorf("%s → Schuljahrende %d, erwartet %d", f.jetzt.Format("2006-01-02"), got, f.ende)
		}
	}
}
