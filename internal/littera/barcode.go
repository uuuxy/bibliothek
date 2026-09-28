package littera

import (
	"regexp"
	"strconv"
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
// Die Zeichenkette IST der EAN-13 des Etiketts, gesetzt in einer EAN-13-Schrift: Die erste
// Ziffer steht vor dem Startzeichen, die linke Hälfte wählt je Ziffer zwischen den Reihen
// `qwertz…` und `asdf…` (die EAN-Parität, aus der ein Scanner die erste Ziffer liest), die
// rechte Hälfte steht in der Reihe `yxcv…`, das letzte Zeichen ist die EAN-Prüfziffer.
// Nachgemessen am 28.09.2026 an allen 61.520 Exemplaren der Sicherung von 2010: Parität und
// Prüfziffer stimmen bei allen (siehe EtikettZiffern). Bis dahin stand hier, die Etiketten
// seien „inzwischen durchweg EAN-13" und die Zeichenkette beschreibe eine abgelöste
// Generation — es ist dieselbe Generation.
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

// eanParitaet ist die Parität der linken Hälfte je erster Ziffer (EAN-13, A = `qwertz…`,
// B = `asdf…`). Die erste Ziffer steht nicht als Balken im Code, ein Scanner liest sie aus
// diesem Muster.
var eanParitaet = [10]string{
	"AAAAAA", "AABABB", "AABBAB", "AABBBA", "ABAABB",
	"ABBAAB", "ABBBAA", "ABABAB", "ABABBA", "ABBABA",
}

// EtikettZiffern liest aus der Druckzeichenkette die dreizehn Ziffern, die ein Scanner vom
// Etikett liefert. ok nur, wenn Parität und Prüfziffer stimmen — sonst ist die Zeichenkette
// kein EAN-13, und niemand darf raten.
func EtikettZiffern(roh string) (string, bool) {
	treffer := barcodeMuster.FindStringSubmatch(strings.TrimSpace(roh))
	if treffer == nil {
		return "", false
	}
	var paritaet strings.Builder
	for _, r := range treffer[2] {
		switch {
		case strings.ContainsRune("qwertzuiop", r):
			paritaet.WriteByte('A')
		case strings.ContainsRune("asdfghjklö", r):
			paritaet.WriteByte('B')
		default:
			return "", false
		}
	}
	if paritaet.String() != eanParitaet[treffer[1][0]-'0'] {
		return "", false
	}
	rechts := treffer[3] + treffer[4] + treffer[5]
	for _, r := range rechts {
		if !strings.ContainsRune("yxcvbnm,.-", r) {
			return "", false
		}
	}
	ziffern, ok := entschluessele(treffer[2] + rechts)
	if !ok {
		return "", false
	}
	ean := treffer[1] + ziffern
	if EAN13Pruefziffer(ean[:12]) != int(ean[12]-'0') {
		return "", false
	}
	return ean, true
}

// Das Etikett am Buch trägt eine EAN-13, nicht die nackte Exemplarnummer. Aufbau:
//
//	5 8 9 6 8 0 0   0 3 9 5   5   6
//	└ 58968 ┘ └┘    └ 0395 ┘  ↑   ↑
//	Exemplarnr.,    Bibl.-Nr. │   EAN-13-Prüfziffer
//	rechts mit Nullen         └── Stellenzahl der Exemplarnummer
//	auf 7 Stellen
//
// Belegt an vier gescannten Büchern (105785 → 1057850039567, 110815 → 1108150039563,
// 58968 → 5896800039556, 124117 → 1241170039561) und an allen 61.520 Druckzeichenketten
// der Sicherung von 2010 (61.518 gleich; die zwei anderen tragen in der Spalte die
// Bibliotheksnummer 0, auf dem Etikett 0395).
//
// Bis zum 28.09.2026 füllte EtikettBarcode LINKS auf sechs Stellen auf und setzte an Stelle
// 12 fest eine 6 — abgeleitet aus zwei 6-stelligen Nummern, bei denen beide Regeln dasselbe
// ergeben. Für jede kürzere Nummer entstand ein Barcode, den kein Etikett trägt; in der
// Sicherung von 2010 hat keine Nummer mehr als fünf Stellen, getroffen hätte es also jedes
// ihrer 61.520 Exemplare. Der Scanner rechnete schon richtig zurück (dekodiereLitteraEtikett
// in internal/service); dass Übernahme und Scanner übereinstimmen, prüft
// internal/service/littera_etikett_uebernahme_test.go.
//
// Aufdruck und Scanwert sind verschiedene Werte — dieselbe Falle wie beim Schülerausweis,
// wo unter dem Strichcode „[0395] 37" steht, der Scanner aber „B97601826457" liefert.
const (
	etikettNummerLen = 7 // erste Ziffer und linke Hälfte des EAN-13
	etikettBibLen    = 4
)

// EtikettBarcode baut den Wert, den ein Scanner vom Buchetikett liest.
//
// ok ist false, wenn die Nummern nicht in das Muster passen (zu lang oder nicht
// numerisch). Dann darf niemand raten: Ein falscher Barcode ist schlimmer als gar keiner,
// weil das Buch dann unter einer Nummer steht, die es nirgends gibt.
//
// Eine führende Null lehnt sie ab: Rechts aufgefüllt wäre „0808" nicht von „808" zu
// unterscheiden, und Littera vergibt solche Nummern nicht.
func EtikettBarcode(exemplarnummer, bibliotheksnummer string) (string, bool) {
	nr := strings.TrimSpace(exemplarnummer)
	if _, ok := zifferngefuellt(nr, etikettNummerLen); !ok || nr[0] == '0' {
		return "", false
	}
	bib, bibOK := zifferngefuellt(bibliotheksnummer, etikettBibLen)
	if !bibOK {
		return "", false
	}
	rumpf := nr + strings.Repeat("0", etikettNummerLen-len(nr)) + bib + strconv.Itoa(len(nr))
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
