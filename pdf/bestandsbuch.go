package pdf

import (
	"fmt"
	"math"
	"strings"
	"time"

	"bibliothek/pkg/schulzeit"

	"github.com/phpdave11/gofpdf"
)

// bestandsbuchKopf zeichnet den Briefkopf beider Bücher: Schule, Datum, Titel, Zeitraum.
func bestandsbuchKopf(p *gofpdf.Fpdf, tr func(string) string, titel string, von, bis time.Time, schule SchuleInfo) {
	p.SetFont("Arial", "B", 12)
	p.Cell(0, 8, tr(schule.Name))
	p.Ln(5)
	p.SetFont("Arial", "", 8)
	p.SetTextColor(100, 100, 100)
	p.Cell(0, 4, tr(schule.Absenderzeile()))
	p.SetTextColor(0, 0, 0)
	p.Ln(10)

	p.SetFont("Arial", "", 10)
	p.CellFormat(0, 6, tr(schule.ortDatum(schulzeit.Jetzt().Format(datumsformat))), "", 1, "R", false, 0, "")
	p.Ln(4)

	p.SetFont("Arial", "B", 14)
	p.Cell(0, 10, tr(titel))
	p.Ln(7)
	p.SetFont("Arial", "", 10)
	p.SetTextColor(80, 80, 80)
	p.Cell(0, 6, tr(fmt.Sprintf("Zeitraum: %s bis %s", von.Format(datumsformat), bis.Format(datumsformat))))
	p.SetTextColor(0, 0, 0)
	p.Ln(12)
}

// buchzelle ist eine Zelle einer Zeile von Zugangs- oder Abgangsbuch.
type buchzelle struct {
	breite float64
	text   string
	// umbrechen lässt den Text in seiner Spalte umlaufen; die übrigen Zellen sind einzeilig.
	umbrechen bool
}

// buchzeileAbstand ist der Abstand der Linien in einer umbrechenden Zelle.
const buchzeileAbstand = 4.5

// zeichneBuchzeile zeichnet eine Zeile von Zugangs- oder Abgangsbuch. Die Barcodes laufen in
// ihrer Spalte um, statt gekürzt zu werden: Der Nachweis nennt jede Nummer. Passt die Zeile nicht
// mehr auf die Seite, beginnt sie auf der nächsten unter den Spaltenköpfen; ist sie höher als eine
// ganze Seite, wird sie geteilt, und jedes Stück trägt die übrigen Zellen noch einmal.
func zeichneBuchzeile(p *gofpdf.Fpdf, tr func(string) string, zellen []buchzelle, koepfe func()) {
	linien, hoechste := buchzeileLinien(p, tr, zellen)
	_, oben, _, _ := p.GetMargins()
	aufNeuerSeite := passendeLinien(oben + abgangZeilenHoehe)
	for erste := 0; erste < hoechste; {
		rest := hoechste - erste
		frei := passendeLinien(p.GetY())
		if frei < 1 || (erste == 0 && frei < rest && rest <= aufNeuerSeite) {
			p.AddPage()
			koepfe()
			continue
		}
		n := min(frei, rest)
		zeichneBuchzeilenStueck(p, tr, zellen, linien, erste, n)
		erste += n
		if erste < hoechste {
			p.AddPage()
			koepfe()
		}
	}
}

// passendeLinien zählt die Linien einer Zeile, die ab der Höhe y noch über der Grenze für den
// Seitenumbruch Platz haben.
func passendeLinien(y float64) int {
	return 1 + int(math.Floor((abgangUmbruchAbY-y)/buchzeileAbstand))
}

// buchzeileLinien teilt jede Zelle in die Linien, die sie druckt: die umbrechende in so viele,
// wie ihre Spalte verlangt, jede andere in eine. hoechste ist die Zahl der Linien der Zeile.
func buchzeileLinien(p *gofpdf.Fpdf, tr func(string) string, zellen []buchzelle) (linien [][]string, hoechste int) {
	linien = make([][]string, len(zellen))
	hoechste = 1
	for i, z := range zellen {
		if !z.umbrechen {
			linien[i] = []string{z.text}
			continue
		}
		linien[i] = brichUm(p, tr, z.text, z.breite-2*p.GetCellMargin())
		hoechste = max(hoechste, len(linien[i]))
	}
	return linien, hoechste
}

// brichUm teilt einen Text an seinen Leerzeichen in Linien, die gedruckt höchstens breite
// Millimeter breit sind. Geschützte Leerzeichen halten einen Bereich zusammen.
func brichUm(p *gofpdf.Fpdf, tr func(string) string, text string, breite float64) []string {
	passt := func(s string) bool { return p.GetStringWidth(tr(s)) <= breite }
	var linien []string
	zeile := ""
	for _, wort := range woerter(text, passt) {
		switch {
		case zeile == "":
			zeile = wort
		case passt(zeile + " " + wort):
			zeile += " " + wort
		default:
			linien = append(linien, zeile)
			zeile = wort
		}
	}
	return append(linien, zeile)
}

// woerter teilt einen Text an seinen Leerzeichen. Ein Wort, das allein nicht in die Spalte passt,
// etwa ein Bereich aus zwei langen Barcodes, wird an seinen geschützten Leerzeichen geteilt, statt
// in die Nachbarspalte zu laufen.
func woerter(text string, passt func(string) bool) []string {
	var aus []string
	for _, wort := range strings.Split(text, " ") {
		if passt(wort) {
			aus = append(aus, wort)
			continue
		}
		aus = append(aus, strings.Split(wort, "\u00a0")...)
	}
	return aus
}

// zeichneBuchzeilenStueck zeichnet n Linien einer Zeile ab der Linie erste, umrandet links,
// rechts und unten wie die einzeiligen Zeilen. Die einzeiligen Zellen stehen in jedem Stück oben.
func zeichneBuchzeilenStueck(p *gofpdf.Fpdf, tr func(string) string, zellen []buchzelle, linien [][]string, erste, n int) {
	x, y := p.GetX(), p.GetY()
	links := x
	for i, z := range zellen {
		stueck := linien[i][:1]
		if z.umbrechen {
			stueck = linien[i][min(erste, len(linien[i])):min(erste+n, len(linien[i]))]
		}
		for k, linie := range stueck {
			p.SetXY(links, y+float64(k)*buchzeileAbstand)
			p.CellFormat(z.breite, abgangZeilenHoehe, tr(linie), "", 0, "L", false, 0, "")
		}
		links += z.breite
	}
	hoehe := float64(n-1)*buchzeileAbstand + abgangZeilenHoehe
	p.Line(x, y, x, y+hoehe)
	p.Line(links, y, links, y+hoehe)
	p.Line(x, y+hoehe, links, y+hoehe)
	p.SetXY(x, y+hoehe)
}
