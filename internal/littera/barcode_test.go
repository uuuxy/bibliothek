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

// TestEtikettBarcode rechnet gegen echte Etiketten: vier gescannte Bücher und drei
// Druckzeichenketten aus TestBarcodeInhalt.
//
// Bis zum 28.09.2026 standen hier für 808 und 61512 „0008080039569" und „0615120039566" —
// nicht gemessen, sondern aus der Annahme gerechnet, kürzere Nummern würden links
// aufgefüllt. Ihre eigenen Druckzeichenketten zwei Tests weiter oben sagen etwas anderes.
func TestEtikettBarcode(t *testing.T) {
	faelle := []struct {
		exemplarnummer string
		erwartet       string
	}{
		{"105785", "1057850039567"}, // gescannt, Zeitreise, Ansch.-J. 2022
		{"110815", "1108150039563"}, // gescannt
		{"124117", "1241170039561"}, // gescannt am 18.08.2026 (internal/service)
		{"58968", "5896800039556"},  // gescannt am 18.08.2026, fünfstellig
		{"808", "8080000039530"},    // Druckzeichenkette 8 *pkpööp#-c.bc-*
		{"25317", "2531700039550"},  // Druckzeichenkette 2 *teajpö#-c.bb-*
		{"61512", "6151200039551"},  // Druckzeichenkette 6 *qgaspp#-c.bby*
	}
	for _, f := range faelle {
		got, ok := EtikettBarcode(f.exemplarnummer, "395")
		if !ok {
			t.Errorf("Exemplarnummer %q ergab keinen Barcode", f.exemplarnummer)
			continue
		}
		if got != f.erwartet {
			t.Errorf("Exemplarnummer %q → %q, erwartet %q", f.exemplarnummer, got, f.erwartet)
		}
		if len(got) != 13 {
			t.Errorf("EAN-13 muss 13 Stellen haben, %q hat %d", got, len(got))
		}
		if p := EAN13Pruefziffer(got[:12]); string('0'+rune(p)) != got[12:] {
			t.Errorf("%q trägt eine falsche Prüfziffer", got)
		}
	}
}

// TestEtikettBarcodeLehntUnpassendesAb: ein falscher Barcode ist schlimmer als gar
// keiner — dann steht das Buch unter einer Nummer, die es nirgends gibt.
func TestEtikettBarcodeLehntUnpassendesAb(t *testing.T) {
	faelle := []struct{ nummer, bib string }{
		{"", "395"},         // keine Exemplarnummer
		{"12345678", "395"}, // passt nicht in sieben Stellen
		{"0808", "395"},     // führende Null, rechts aufgefüllt nicht von 808 zu trennen
		{"12A456", "395"},   // keine reine Ziffernfolge
		{"105785", ""},      // keine Bibliotheksnummer
		{"105785", "12345"}, // Bibliotheksnummer zu lang
	}
	for _, f := range faelle {
		if got, ok := EtikettBarcode(f.nummer, f.bib); ok {
			t.Errorf("(%q, %q) hätte abgelehnt werden müssen, ergab %q", f.nummer, f.bib, got)
		}
	}
}

// TestEtikettZiffern liest aus echten Druckzeichenketten den EAN-13 und hält ihn gegen die
// Rechnung aus Nummer und Bibliotheksnummer: Beide beschreiben dasselbe Etikett.
func TestEtikettZiffern(t *testing.T) {
	faelle := []struct{ roh, ean string }{
		{"8 *pkpööp#-c.bc-*", "8080000039530"},
		{"8 *plpööp#-c.bc.*", "8090000039539"},
		{"8 *qöpööp#-c.bcb*", "8100000039535"},
		{"2 *teajpö#-c.bb-*", "2531700039550"},
		{"6 *qgaspp#-c.bby*", "6151200039551"},
		{"5 *pafopö#-c.bbc*", "5014900039553"}, // in Littera zweimal, einmal mit Bibliotheksnummer 0
	}
	for _, f := range faelle {
		ean, ok := EtikettZiffern(f.roh)
		if !ok || ean != f.ean {
			t.Errorf("%q → %q (%v), erwartet %q", f.roh, ean, ok, f.ean)
			continue
		}
		nummer, bib, _ := BarcodeInhalt(f.roh)
		if gerechnet, _ := EtikettBarcode(nummer, bib); gerechnet != ean {
			t.Errorf("%q: aus %s/%s gerechnet %q, das Etikett trägt %q", f.roh, nummer, bib, gerechnet, ean)
		}
	}
}

// TestEtikettZifferLehntAb: Stimmen Parität, Prüfziffer oder Zeichenvorrat nicht, ist die
// Zeichenkette kein EAN-13, und es wird gerechnet statt geraten.
func TestEtikettZifferLehntAb(t *testing.T) {
	for _, roh := range []string{
		"",
		"B-00042",
		"8 *ökpööp#-c.bc-*", // erste Ziffer passt nicht zur Parität der linken Hälfte
		"8 *pkpööp#-c.bcy*", // falsche Prüfziffer
		"8 *pkpööp#pc.bc-*", // rechte Hälfte aus der falschen Reihe
		"8 *pkpöö?#-c.bc-*", // unbekanntes Zeichen
	} {
		if ean, ok := EtikettZiffern(roh); ok {
			t.Errorf("%q hätte abgelehnt werden müssen, ergab %q", roh, ean)
		}
	}
}
