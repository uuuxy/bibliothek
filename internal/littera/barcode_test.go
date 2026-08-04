package littera

import (
	"testing"
	"time"
)

// TestBarcodeInhalt entschlüsselt echte Etikettenzeichenketten aus dem Altbestand.
//
// Der interessante Fall ist die 810: Ohne die Längenangabe an vorletzter Stelle wären
// „81" und „810" beide `100000` — die Nummer wäre aus dem Etikett nicht eindeutig
// rekonstruierbar, und der Import müsste raten, was auf dem Buch klebt.
func TestBarcodeInhalt(t *testing.T) {
	faelle := []struct {
		roh        string
		nummer     string
		bibliothek string
	}{
		{"8 *pkpööp#-c.bc-*", "808", "0395"},
		{"8 *plpööp#-c.bc.*", "809", "0395"},
		{"8 *qöpööp#-c.bcb*", "810", "0395"}, // endet auf 0 – nur über die Länge lesbar
		{"2 *teajpö#-c.bb-*", "25317", "0395"},
		{"6 *qgaspp#-c.bby*", "61512", "0395"},
	}
	for _, f := range faelle {
		nummer, bib, ok := BarcodeInhalt(f.roh)
		if !ok {
			t.Errorf("%q wurde nicht erkannt", f.roh)
			continue
		}
		if nummer != f.nummer || bib != f.bibliothek {
			t.Errorf("%q → %q/%q, erwartet %q/%q", f.roh, nummer, bib, f.nummer, f.bibliothek)
		}
	}
}

// TestBarcodeInhaltLehntUnbekanntesAb: bei einer Zeichenkette, die dem Muster nicht folgt,
// darf niemand raten, was auf dem Buch klebt.
func TestBarcodeInhaltLehntUnbekanntesAb(t *testing.T) {
	for _, roh := range []string{"", "B-00042", "8 *pkpööp*", "8 *pkpöö?#-c.bc-*", "*pkpööp#-c.bc-*"} {
		if _, _, ok := BarcodeInhalt(roh); ok {
			t.Errorf("%q hätte abgelehnt werden müssen", roh)
		}
	}
}

// TestGeburtsdatumAus sichert die Jahrhundertgrenze ab. Go legt sie bei „06" fest auf 69;
// für Ausleihdaten stimmt das, für Geburtsdaten nicht.
func TestGeburtsdatumAus(t *testing.T) {
	jetzt := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	faelle := []struct {
		roh  string
		jahr int
	}{
		{"05/03/95", 1995},          // Schülerjahrgang, von Go richtig gelesen
		{"03/17/63", 1963},          // Lehrkraft – Go läse 2063
		{"12/31/68", 1968},          // genau an der Grenze
		{"01/15/05", 2005},          // Schülerjahrgang nach 2000, muss so bleiben
		{"06/01/26 00:00:00", 2026}, // schon vergangen – bleibt unangetastet
		{"12/01/26 00:00:00", 1926}, // noch nicht eingetreten – muss zurückgeholt werden
	}
	for _, f := range faelle {
		got, ok := GeburtsdatumAus(f.roh, jetzt)
		if !ok {
			t.Errorf("%q wurde nicht gelesen", f.roh)
			continue
		}
		if got.Year() != f.jahr {
			t.Errorf("%q → %d, erwartet %d", f.roh, got.Year(), f.jahr)
		}
		if got.After(jetzt) {
			t.Errorf("%q ergibt ein Geburtsdatum in der Zukunft: %s", f.roh, got)
		}
	}

	if _, ok := GeburtsdatumAus("", jetzt); ok {
		t.Error("ein leeres Geburtsdatum darf nicht als gültig durchgehen")
	}
}

// TestStandardOptionenSchuljahr: das Schuljahr endet im Sommer. Ab August zählt bereits
// das nächste Kalenderjahr — dieselbe Auslegung wie im Versetzungslauf. Läge die Grenze
// falsch, bekämen alle importierten Schüler ein um ein Jahr verschobenes Abgangsjahr.
func TestStandardOptionenSchuljahr(t *testing.T) {
	faelle := []struct {
		jetzt time.Time
		ende  int
	}{
		{time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC), 2026},
		{time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), 2027},
		{time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC), 2027},
	}
	for _, f := range faelle {
		if got := StandardOptionen(f.jetzt).SchuljahrEnde; got != f.ende {
			t.Errorf("%s → Schuljahrende %d, erwartet %d", f.jetzt.Format("2006-01-02"), got, f.ende)
		}
	}
}
