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
// Wofür das heute noch gut ist: NICHT um zu wissen, was im Regal klebt — die Etiketten
// der Schule sind inzwischen durchweg EAN-13 (siehe EtikettBarcode), die alte
// Druckzeichenkette beschreibt eine abgelöste Generation. Es ist eine PRÜFUNG der Quelle:
// Die EAN-13 wird aus `Exemplarnummer` gerechnet, also muss diese Spalte stimmen. Die
// Druckzeichenkette trägt dieselbe Nummer unabhängig kodiert und dient als Gegenprobe —
// am Altbestand stimmen 61.520 von 61.520 überein (siehe pruefeEtiketten).
//
// Ohne die Längenangabe an vorletzter Stelle wäre die Nummer übrigens nicht eindeutig
// rekonstruierbar: 81 und 810 ergäben beide `100000`.
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

// Das Etikett am Buch trägt eine EAN-13, nicht die nackte Exemplarnummer.
//
// Gemessen an zwei echten Büchern der Schule:
//
//	Exemplar-Nr. 105785  →  1057850039567
//	Exemplar-Nr. 110815  →  1108150039563
//
// Aufbau (beide Male identisch, Prüfziffer jeweils verifiziert):
//
//	1 0 5 7 8 5   0   0 3 9 5   6   7
//	└─ 105785 ─┘  ↑   └─ 0395 ┘ ↑   ↑
//	Exemplarnr.   │   Bibl.-Nr. │   EAN-13-Prüfziffer
//	6-stellig     └── konstant ─┘
//
// Warum das hier steht und nicht „nimm Exemplar.Exemplarnummer": Der Import schrieb
// zuerst die nackte Nummer nach buecher_exemplare.barcode_id. Kein einziges der 61.520
// Exemplare wäre damit am Scanner auffindbar gewesen — und aufgefallen wäre es erst an
// der Theke. Dieselbe Falle wie beim Schülerausweis, wo unter dem Strichcode „[0395] 37"
// steht, der Scanner aber „B97601826457" liefert.
//
// Die Stellen 7 und 12 sind aus zwei Proben als konstant abgeleitet, nicht aus einer
// Littera-Dokumentation. Liefert ein drittes Buch etwas anderes, gehören sie
// parametrisiert — der Aufbau steht deshalb hier an einer Stelle und nicht verstreut.
const (
	etikettFuellerVorne  = "0" // Stelle 7, zwischen Exemplarnummer und Bibliotheksnummer
	etikettFuellerHinten = "6" // Stelle 12, vor der Prüfziffer
	etikettExemplarLen   = 6
	etikettBibLen        = 4
)

// EtikettBarcode baut den Wert, den ein Scanner vom Buchetikett liest.
//
// ok ist false, wenn die Nummern nicht in das Muster passen (zu lang oder nicht
// numerisch). Dann darf niemand raten: Ein falscher Barcode ist schlimmer als gar keiner,
// weil das Buch dann unter einer Nummer steht, die es nirgends gibt.
func EtikettBarcode(exemplarnummer, bibliotheksnummer string) (string, bool) {
	nr, nrOK := zifferngefuellt(exemplarnummer, etikettExemplarLen)
	bib, bibOK := zifferngefuellt(bibliotheksnummer, etikettBibLen)
	if !nrOK || !bibOK {
		return "", false
	}
	rumpf := nr + etikettFuellerVorne + bib + etikettFuellerHinten
	return rumpf + string('0'+rune(EAN13Pruefziffer(rumpf))), true
}

// EAN13Pruefziffer rechnet die Prüfziffer über die ersten zwölf Stellen.
func EAN13Pruefziffer(zwoelf string) int {
	summe := 0
	for i, r := range zwoelf {
		gewicht := 1
		if i%2 == 1 {
			gewicht = 3
		}
		summe += int(r-'0') * gewicht
	}
	return (10 - summe%10) % 10
}

// zifferngefuellt prüft auf reine Ziffern und füllt links mit Nullen auf.
func zifferngefuellt(wert string, laenge int) (string, bool) {
	wert = strings.TrimSpace(wert)
	if wert == "" || len(wert) > laenge {
		return "", false
	}
	for _, r := range wert {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return strings.Repeat("0", laenge-len(wert)) + wert, true
}
