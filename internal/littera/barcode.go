package littera

import (
	"regexp"
	"strings"
)

// Litteras Barcodefeld ist keine Nummer, sondern die ZEICHENKETTE FÜR DEN DRUCK: Sie
// wird in einer Barcode-Schrift gesetzt, in der jedes Zeichen das Balkenmuster einer
// Ziffer trägt. Ein Etikett sieht so aus:
//
//	8 *pkpööp#-c.bc-*
//	│  │      │    ││
//	│  │      │    │└─ Prüfzeichen
//	│  │      │    └── Länge der Exemplarnummer (hier 3)
//	│  │      └─────── Bibliotheksnummer, vierstellig (hier 0395)
//	│  └────────────── Rest der Exemplarnummer, rechts mit Nullen auf 6 Stellen aufgefüllt
//	└───────────────── erste Ziffer der Exemplarnummer, außerhalb der Start-/Stoppzeichen
//
// Die Ziffern sind auf zwei Tastaturreihen abgebildet, `qwertzuiop` und `asdfghjklö`
// stehen beide für 1–9 und 0; die vierte Reihe `yxcvbnm,.-` trägt Bibliotheksnummer,
// Länge und Prüfzeichen. Zwei Alphabete für dieselben Ziffern, damit gleiche Ziffern
// nebeneinander unterscheidbare Balken ergeben.
//
// Warum das hier steht statt „nimm einfach die Spalte Exemplarnummer": Ohne die
// Längenangabe an vorletzter Stelle wäre die Nummer aus dem Etikett nicht eindeutig
// rekonstruierbar — 81 und 810 ergäben beide `100000`. Diese Funktion belegt, dass die
// Spalte `Exemplarnummer` wirklich das ist, was auf dem Buch klebt: gegen den Altbestand
// stimmen 61.520 von 61.520 überein (siehe TestBarcodeInhaltTrifftExemplarnummer).
var barcodeMuster = regexp.MustCompile(`^(\d) \*(.{6})#(.{4})(.)(.)\*$`)

// barcodeZiffern bildet die Druckzeichen auf ihre Ziffer ab.
var barcodeZiffern = func() map[rune]int {
	m := map[rune]int{}
	for _, reihe := range []string{"qwertzuiop", "asdfghjklö", "yxcvbnm,.-"} {
		for i, r := range reihe {
			m[r] = (i + 1) % 10 // die zehnte Taste jeder Reihe ist die 0
		}
	}
	return m
}()

// BarcodeInhalt entschlüsselt die Druckzeichenkette eines Littera-Etiketts.
//
// Zurück kommen die Exemplarnummer und die Bibliotheksnummer, so wie sie auf dem
// Etikett stehen. ok ist false, wenn die Zeichenkette dem Muster nicht folgt — dann darf
// niemand raten, was auf dem Buch klebt.
func BarcodeInhalt(roh string) (exemplarnummer, bibliotheksnummer string, ok bool) {
	treffer := barcodeMuster.FindStringSubmatch(strings.TrimSpace(roh))
	if treffer == nil {
		return "", "", false
	}
	zahl, zahlOK := entschluessele(treffer[2])
	bib, bibOK := entschluessele(treffer[3])
	laenge, laengeOK := entschluesseleZiffer(treffer[4])
	if !zahlOK || !bibOK || !laengeOK || laenge < 1 || laenge > len(zahl)+1 {
		return "", "", false
	}
	// treffer[1] ist die erste Ziffer, treffer[2] trägt den Rest — deshalb laenge-1.
	return treffer[1] + zahl[:laenge-1], bib, true
}

func entschluessele(s string) (string, bool) {
	var b strings.Builder
	for _, r := range s {
		z, bekannt := barcodeZiffern[r]
		if !bekannt {
			return "", false
		}
		b.WriteByte(byte('0' + z))
	}
	return b.String(), true
}

func entschluesseleZiffer(s string) (int, bool) {
	for _, r := range s {
		z, bekannt := barcodeZiffern[r]
		return z, bekannt
	}
	return 0, false
}
