package pdf

// Die zwei Übersichten der Jahresansicht eines Bestellberichts: nach Monat und nach Lieferant.

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"bibliothek/pkg/betrag"
	"bibliothek/pkg/schulzeit"

	"github.com/phpdave11/gofpdf"
)

var berichtMonatsnamen = [...]string{"Januar", "Februar", "März", "April", "Mai", "Juni",
	"Juli", "August", "September", "Oktober", "November", "Dezember"}

type berichtMonthStat struct {
	count, exemplare int
	betrag           betrag.Cent
}

type lieferantenSumme struct {
	count     int
	exemplare int
	betrag    betrag.Cent
}

// zeichneBerichtMonate setzt die Tabelle „Übersicht nach Monat". Ohne Preise entfällt die
// Betragsspalte, und die übrigen Spalten füllen dieselbe Breite; sonst endete die Tabelle
// mitten auf der Seite.
func zeichneBerichtMonate(p *gofpdf.Fpdf, tr func(string) string, bestellungen []BestellberichtBestellung, gesamtExemplare int, gesamtBetrag betrag.Cent, mitPreisen bool) {
	jeMonat := map[time.Month]*berichtMonthStat{}
	for _, o := range bestellungen {
		// Der Monat der Schule: Eine Bestellung aus der ersten Stunde eines Monats zählt zu ihm.
		m := o.Bestelldatum.In(schulzeit.Zone()).Month()
		if jeMonat[m] == nil {
			jeMonat[m] = &berichtMonthStat{}
		}
		jeMonat[m].count++
		jeMonat[m].exemplare += o.AnzahlExemplare
		jeMonat[m].betrag += o.Gesamtbetrag
	}

	p.SetFont("Arial", "B", 11)
	p.Cell(0, 8, tr("Übersicht nach Monat"))
	p.Ln(8)
	p.SetFont("Arial", "B", 9)
	p.SetFillColor(220, 220, 220)
	bMonat, bAnzahl, bExemplare := 55.0, 40.0, 40.0
	if !mitPreisen {
		bMonat, bAnzahl, bExemplare = 70.0, 50.0, 50.0 // Summe bleibt 170
	}
	p.CellFormat(bMonat, 7, tr("Monat"), "1", 0, "L", true, 0, "")
	p.CellFormat(bAnzahl, 7, tr("Bestellungen"), "1", 0, "C", true, 0, "")
	if mitPreisen {
		p.CellFormat(bExemplare, 7, tr("Exemplare"), "1", 0, "C", true, 0, "")
		p.CellFormat(35, 7, tr("Betrag"), "1", 1, "R", true, 0, "")
	} else {
		p.CellFormat(bExemplare, 7, tr("Exemplare"), "1", 1, "C", true, 0, "")
	}
	p.SetFont("Arial", "", 9)
	p.SetFillColor(255, 255, 255)
	for m := time.January; m <= time.December; m++ {
		if werte, ok := jeMonat[m]; ok {
			p.CellFormat(bMonat, 6, tr(berichtMonatsnamen[m-1]), "1", 0, "L", false, 0, "")
			p.CellFormat(bAnzahl, 6, fmt.Sprintf("%d", werte.count), "1", 0, "C", false, 0, "")
			if mitPreisen {
				p.CellFormat(bExemplare, 6, fmt.Sprintf("%d", werte.exemplare), "1", 0, "C", false, 0, "")
				p.CellFormat(35, 6, tr(werte.betrag.Euro()), "1", 1, "R", false, 0, "")
			} else {
				p.CellFormat(bExemplare, 6, fmt.Sprintf("%d", werte.exemplare), "1", 1, "C", false, 0, "")
			}
		}
	}
	p.SetFont("Arial", "B", 9)
	p.SetFillColor(240, 240, 240)
	p.CellFormat(bMonat, 7, tr("Gesamt"), "1", 0, "L", true, 0, "")
	p.CellFormat(bAnzahl, 7, fmt.Sprintf("%d", len(bestellungen)), "1", 0, "C", true, 0, "")
	if mitPreisen {
		p.CellFormat(bExemplare, 7, fmt.Sprintf("%d", gesamtExemplare), "1", 0, "C", true, 0, "")
		p.CellFormat(35, 7, tr(gesamtBetrag.Euro()), "1", 1, "R", true, 0, "")
	} else {
		p.CellFormat(bExemplare, 7, fmt.Sprintf("%d", gesamtExemplare), "1", 1, "C", true, 0, "")
	}
	p.SetFillColor(255, 255, 255)
	p.Ln(10)
}

// zeichneBerichtLieferanten setzt die Aufteilung nach Lieferant. Ohne Preise heißt sie
// nicht „Ausgaben", und statt des Betrags stehen die Exemplare: Wie viel die Schule wo
// bezogen hat, bleibt auch ohne Geld eine Auskunft.
func zeichneBerichtLieferanten(p *gofpdf.Fpdf, tr func(string) string, bestellungen []BestellberichtBestellung, mitPreisen bool) {
	jeLieferant := map[string]*lieferantenSumme{}
	for _, o := range bestellungen {
		if jeLieferant[o.LieferantName] == nil {
			jeLieferant[o.LieferantName] = &lieferantenSumme{}
		}
		jeLieferant[o.LieferantName].count++
		jeLieferant[o.LieferantName].exemplare += o.AnzahlExemplare
		jeLieferant[o.LieferantName].betrag += o.Gesamtbetrag
	}
	ueberschrift := "Ausgaben nach Lieferant"
	if !mitPreisen {
		ueberschrift = "Bestellungen nach Lieferant"
	}
	p.SetFont("Arial", "B", 11)
	p.Cell(0, 8, tr(ueberschrift))
	p.Ln(8)
	p.SetFont("Arial", "B", 9)
	p.SetFillColor(220, 220, 220)
	p.CellFormat(85, 7, tr("Lieferant"), "1", 0, "L", true, 0, "")
	p.CellFormat(40, 7, tr("Bestellungen"), "1", 0, "C", true, 0, "")
	if mitPreisen {
		p.CellFormat(45, 7, tr("Betrag"), "1", 1, "R", true, 0, "")
	} else {
		p.CellFormat(45, 7, tr("Exemplare"), "1", 1, "C", true, 0, "")
	}
	p.SetFont("Arial", "", 9)
	p.SetFillColor(255, 255, 255)
	// Eine Map hat keine Reihenfolge: Ohne die sortierten Namen stünden die Lieferanten bei
	// jedem Druck anders untereinander.
	namen := make([]string, 0, len(jeLieferant))
	for name := range jeLieferant {
		namen = append(namen, name)
	}
	slices.SortFunc(namen, func(a, b string) int {
		if c := cmp.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})
	for _, name := range namen {
		werte := jeLieferant[name]
		p.CellFormat(85, 6, tr(name), "1", 0, "L", false, 0, "")
		p.CellFormat(40, 6, fmt.Sprintf("%d", werte.count), "1", 0, "C", false, 0, "")
		if mitPreisen {
			p.CellFormat(45, 6, tr(werte.betrag.Euro()), "1", 1, "R", false, 0, "")
		} else {
			p.CellFormat(45, 6, fmt.Sprintf("%d", werte.exemplare), "1", 1, "C", false, 0, "")
		}
	}
	p.Ln(10)
}
