// Package bestandsbuch legt die Zeilen von Zugangs- und Abgangsbuch fest: je Tag und Titel
// eine, mit der Anzahl und den Barcodes. Bildschirm und Blatt lesen dieselben Zeilen.
package bestandsbuch

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"

	"bibliothek/repository"
)

// So beschreibt die Arbeitshilfe das Zugangsbuch: „jede Lieferung mit Eingangsdatum,
// Titel, Anzahl, Lieferant, Inventarnummer" (docs/mittel_konzept.md 7.1). Eine Zeile je Exemplar
// stellte einen Klassensatz mit 45 Zeilen hin.
//
// Zusammengefasst wird, was in allen Spalten des Buchs gleich ist; zwei Zeilen, die sich nur im
// Barcode unterschieden, sähen sonst gleich aus. Jeder Barcode bleibt genannt, weil die
// Bestandskartei ihn braucht. Die Abfragen bleiben je Exemplar (repository.LadeZugangsbuch):
// Was zusammengehört, ist eine Regel der Darstellung, und Blatt und Bildschirm teilen sie hier.

// Zugang ist eine Zeile des Zugangsbuchs: die Exemplare eines Titels, die am selben Tag vom
// selben Lieferanten kamen.
type Zugang struct {
	Datum     time.Time `json:"datum"`
	Titel     string    `json:"titel"`
	Lieferant string    `json:"lieferant"`
	Anzahl    int       `json:"anzahl"`
	// Barcodes ist der fertige Text (barcodeText), damit Blatt und Bildschirm ihn gleich
	// schreiben.
	Barcodes string `json:"barcodes"`
	Topf     string `json:"topf"`
}

// Abgang ist eine Zeile des Abgangsbuchs: die Exemplare eines Titels, die am selben Tag aus
// demselben Grund gingen.
type Abgang struct {
	Datum     time.Time `json:"datum"`
	Titel     string    `json:"titel"`
	Signatur  string    `json:"signatur"`
	Grund     string    `json:"grund"`
	GrundText string    `json:"grund_text"`
	Anzahl    int       `json:"anzahl"`
	Barcodes  string    `json:"barcodes"`
	Topf      string    `json:"topf"`
}

// fasseZusammen gruppiert Zeilen mit gleichem Schlüssel in der Reihenfolge, in der sie zuerst
// vorkommen; die Abfragen liefern nach Tag und Titel sortiert. Zu jeder Gruppe kommen ihre erste
// Zeile und die Barcodes aller ihrer Zeilen zurück.
func fasseZusammen[Z any](zeilen []Z, schluessel, barcode func(Z) string) (erste []Z, barcodes [][]string) {
	stelle := map[string]int{}
	for _, z := range zeilen {
		k := schluessel(z)
		i, gesehen := stelle[k]
		if !gesehen {
			i = len(erste)
			stelle[k] = i
			erste = append(erste, z)
			barcodes = append(barcodes, nil)
		}
		barcodes[i] = append(barcodes[i], barcode(z))
	}
	return erste, barcodes
}

// schluessel verbindet die Spalten einer Zeile zu einem Schlüssel. Das Nullzeichen kommt in
// keinem Titel vor und trennt deshalb eindeutig.
func schluessel(teile ...string) string { return strings.Join(teile, "\x00") }

// Zugaenge fasst die Zugänge je Topf, Tag, Titel und Lieferant zusammen.
func Zugaenge(zeilen []repository.ZugangsZeile) []Zugang {
	erste, barcodes := fasseZusammen(zeilen,
		func(z repository.ZugangsZeile) string {
			return schluessel(z.Topf, z.Datum.Format(time.DateOnly), z.Titel, z.Lieferant)
		},
		func(z repository.ZugangsZeile) string { return z.Barcode })
	aus := make([]Zugang, len(erste))
	for i, z := range erste {
		aus[i] = Zugang{Datum: z.Datum, Titel: z.Titel, Lieferant: z.Lieferant,
			Anzahl: len(barcodes[i]), Barcodes: barcodeText(barcodes[i]), Topf: z.Topf}
	}
	return aus
}

// Abgaenge fasst die Abgänge je Topf, Tag, Titel, Signatur und Grund zusammen. Der Tag ist der
// Kalendertag der Schule: Die Abfrage liefert den Zeitpunkt schon in ihrer Zeitzone.
func Abgaenge(zeilen []repository.AbgangsZeile) []Abgang {
	erste, barcodes := fasseZusammen(zeilen,
		func(z repository.AbgangsZeile) string {
			return schluessel(z.Topf, z.Datum.Format(time.DateOnly), z.Titel, z.Signatur, z.Grund)
		},
		func(z repository.AbgangsZeile) string { return z.Barcode })
	aus := make([]Abgang, len(erste))
	for i, z := range erste {
		aus[i] = Abgang{Datum: z.Datum, Titel: z.Titel, Signatur: z.Signatur, Grund: z.Grund,
			GrundText: z.GrundText, Anzahl: len(barcodes[i]), Barcodes: barcodeText(barcodes[i]), Topf: z.Topf}
	}
	return aus
}

// bereichAb ist die Zahl fortlaufender Barcodes, ab der sie als „erste – letzte" stehen. Zwei
// nebeneinander sind als Bereich nicht kürzer.
const bereichAb = 3

// barcodeText schreibt die Barcodes einer Zeile sortiert, durch Komma getrennt, fortlaufende ab
// drei als „erste – letzte". Fortlaufend heißt: dieselbe Vorsilbe, gleich viele Ziffern und die
// Zahl je um eins höher, wie barcode_seq sie bei einer Lieferung vergibt. Um den Strich stehen
// geschützte Leerzeichen, damit ein Bereich beim Umbruch zusammenbleibt.
func barcodeText(barcodes []string) string {
	type nummer struct {
		roh, vorsilbe string
		zahl          uint64
		stellen       int
		zaehlbar      bool
	}
	nummern := make([]nummer, 0, len(barcodes))
	for _, b := range barcodes {
		i := len(b)
		for i > 0 && b[i-1] >= '0' && b[i-1] <= '9' {
			i--
		}
		zahl, err := strconv.ParseUint(b[i:], 10, 64)
		nummern = append(nummern, nummer{roh: b, vorsilbe: b[:i], zahl: zahl, stellen: len(b) - i, zaehlbar: err == nil})
	}
	slices.SortFunc(nummern, func(a, b nummer) int {
		return cmp.Or(strings.Compare(a.vorsilbe, b.vorsilbe), cmp.Compare(a.stellen, b.stellen),
			cmp.Compare(a.zahl, b.zahl), strings.Compare(a.roh, b.roh))
	})
	folgt := func(a, b nummer) bool {
		return a.zaehlbar && b.zaehlbar && a.vorsilbe == b.vorsilbe && a.stellen == b.stellen && b.zahl == a.zahl+1
	}
	teile := make([]string, 0, len(nummern))
	for i := 0; i < len(nummern); {
		j := i
		for j+1 < len(nummern) && folgt(nummern[j], nummern[j+1]) {
			j++
		}
		if j-i+1 >= bereichAb {
			teile = append(teile, nummern[i].roh+" – "+nummern[j].roh)
		} else {
			for _, n := range nummern[i : j+1] {
				teile = append(teile, n.roh)
			}
		}
		i = j + 1
	}
	return strings.Join(teile, ", ")
}
