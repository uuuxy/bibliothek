package pdf

import (
	"bytes"
	"fmt"
	"time"

	"bibliothek/pkg/betrag"
	"bibliothek/pkg/pdfzeichen"
	"bibliothek/pkg/schulzeit"

	"github.com/jung-kurt/gofpdf"
)

// Bestellbericht ist die Eingabe des Berichts über die Bestellungen eines Zeitraums. Der
// Erzeuger druckt die Blöcke seiner Eingabe in ihrer Folge; welche Töpfe es gibt, wie sie
// heißen und welche Bestellung in welchen gehört, weiß die Tür.
type Bestellbericht struct {
	Titel string
	Von   time.Time
	Bis   time.Time
	// Jahresansicht stellt den Bestellungen die Übersichten nach Monat und nach Lieferant voran.
	Jahresansicht bool
	// MitPreisen entscheidet, ob das Blatt Beträge ausweist oder Mengen. Ohne gepflegte Preise
	// summierte ein Betragsbericht Nullen und läse sich wie ein Nachweis über Ausgaben von null.
	MitPreisen bool
	// Toepfe sind die Blöcke der Bestellungen, je Topf einer mit eigener Summe. Für die
	// Abrechnung zählt der Anteil eines Topfs, nicht die Summe einer gemischten Liste.
	Toepfe []BestellberichtTopf
}

// BestellberichtTopf ist ein Block des Berichts: die Bestellungen eines Topfs unter seiner
// Beschriftung.
type BestellberichtTopf struct {
	Beschriftung string
	Bestellungen []BestellberichtBestellung
}

// BestellberichtBestellung ist eine Bestellung mit ihren Positionen.
type BestellberichtBestellung struct {
	LieferantName   string
	Kundennummer    string
	Bestelldatum    time.Time
	Gesamtbetrag    float64
	AnzahlExemplare int
	Positionen      []BestellberichtPosition
}

// BestellberichtPosition ist ein bestellter Titel mit Menge und Einzelpreis.
type BestellberichtPosition struct {
	TitelName   string
	ISBN        string
	Menge       int
	Einzelpreis float64
}

// bestellungen nennt die Bestellungen aller Blöcke hintereinander, für Summen und Übersichten
// über den ganzen Bericht.
func (b Bestellbericht) bestellungen() []BestellberichtBestellung {
	var alle []BestellberichtBestellung
	for _, topf := range b.Toepfe {
		alle = append(alle, topf.Bestellungen...)
	}
	return alle
}

// berichtSpalten sind die Spaltenbreiten der Positionstabelle. Ohne Preisspalten verteilen
// sich Titel, ISBN und Menge auf dieselben 170 mm.
type berichtSpalten struct {
	Titel, ISBN, Menge float64
}

func spaltenFuerBericht(mitPreisen bool) berichtSpalten {
	if mitPreisen {
		return berichtSpalten{Titel: 83, ISBN: 33, Menge: 14}
	}
	// Breitere Titelspalte trägt mehr Text.
	return berichtSpalten{Titel: 103, ISBN: 43, Menge: 24}
}

// berichtRahmen bündelt, was für den ganzen Bericht gilt; als einzelne Argumente reichte
// jeder Helfer dieselbe Liste weiter.
type berichtRahmen struct {
	Von, Bis        time.Time
	Bestellungen    int
	GesamtBetrag    float64
	GesamtExemplare int
	MitPreisen      bool
}

// zeichneBerichtBestellKopf setzt die Kopfzeile einer einzelnen Bestellung (Datum, Lieferant,
// Kundennummer). Das Datum ist der Kalendertag der Schule, an dem bestellt wurde.
func zeichneBerichtBestellKopf(p *gofpdf.Fpdf, tr func(string) string, o BestellberichtBestellung) {
	p.SetFont("Arial", "B", 9)
	p.SetFillColor(235, 235, 245)
	kopf := fmt.Sprintf("%s  ·  %s  ·  Kd.-Nr. %s",
		o.Bestelldatum.In(schulzeit.Zone()).Format(datumsformat), o.LieferantName, o.Kundennummer)
	p.CellFormat(0, 7, tr(kopf), "LTR", 1, "L", true, 0, "")
	p.SetFillColor(255, 255, 255)
}

// zeichneBerichtSpaltenkoepfe setzt die Überschriftenzeile der Positionstabelle.
func zeichneBerichtSpaltenkoepfe(p *gofpdf.Fpdf, tr func(string) string, s berichtSpalten, mitPreisen bool) {
	p.SetFont("Arial", "B", 8)
	p.SetFillColor(245, 245, 245)
	p.CellFormat(s.Titel, 6, tr("Titel"), "1", 0, "L", true, 0, "")
	p.CellFormat(s.ISBN, 6, tr("ISBN"), "1", 0, "C", true, 0, "")
	if mitPreisen {
		p.CellFormat(s.Menge, 6, tr("Menge"), "1", 0, "C", true, 0, "")
		p.CellFormat(20, 6, tr("Einzelpr."), "1", 0, "R", true, 0, "")
		p.CellFormat(20, 6, tr("Gesamt"), "1", 1, "R", true, 0, "")
	} else {
		p.CellFormat(s.Menge, 6, tr("Menge"), "1", 1, "C", true, 0, "")
	}
	p.SetFillColor(255, 255, 255)
}

// zeichneBerichtPositionen setzt die Titelzeilen einer Bestellung, mit Seitenumbruch am Fuß.
func zeichneBerichtPositionen(p *gofpdf.Fpdf, tr func(string) string, positionen []BestellberichtPosition, s berichtSpalten, mitPreisen bool) {
	p.SetFont("Arial", "", 8)
	for _, pos := range positionen {
		if p.GetY() > 265 {
			p.AddPage()
		}
		isbn := pos.ISBN
		if isbn == "" {
			isbn = "—"
		}
		// Gekürzt wird auf die gedruckte Breite der Zelle; gofpdf druckt Überlanges über die ISBN.
		titel := pdfzeichen.KuerzeAufZelle(p, tr, pos.TitelName, s.Titel)
		p.CellFormat(s.Titel, 5, tr(titel), "1", 0, "L", false, 0, "")
		p.CellFormat(s.ISBN, 5, tr(isbn), "1", 0, "C", false, 0, "")
		if mitPreisen {
			p.CellFormat(s.Menge, 5, fmt.Sprintf("%d", pos.Menge), "1", 0, "C", false, 0, "")
			p.CellFormat(20, 5, tr(betrag.Euro(pos.Einzelpreis)), "1", 0, "R", false, 0, "")
			p.CellFormat(20, 5, tr(betrag.Euro(float64(pos.Menge)*pos.Einzelpreis)), "1", 1, "R", false, 0, "")
		} else {
			p.CellFormat(s.Menge, 5, fmt.Sprintf("%d", pos.Menge), "1", 1, "C", false, 0, "")
		}
	}
}

// zeichneBerichtBestellSumme setzt die Abschlusszeile einer einzelnen Bestellung.
func zeichneBerichtBestellSumme(p *gofpdf.Fpdf, tr func(string) string, o BestellberichtBestellung, mitPreisen bool) {
	p.SetFont("Arial", "B", 8)
	p.SetFillColor(235, 235, 245)
	if mitPreisen {
		p.CellFormat(150, 6, tr(fmt.Sprintf("Summe (%d Exemplare)", o.AnzahlExemplare)), "LBR", 0, "R", true, 0, "")
		p.CellFormat(20, 6, tr(betrag.Euro(o.Gesamtbetrag)), "1", 1, "R", true, 0, "")
	} else {
		p.CellFormat(170, 6, tr(fmt.Sprintf("Summe: %d Exemplare", o.AnzahlExemplare)), "LBR", 1, "R", true, 0, "")
	}
	p.SetFillColor(255, 255, 255)
	p.Ln(5)
}

// zeichneBerichtGesamtsumme setzt die Schlusszeile über alle Bestellungen des Zeitraums.
func zeichneBerichtGesamtsumme(p *gofpdf.Fpdf, tr func(string) string, r berichtRahmen) {
	p.Ln(4)
	p.SetFont("Arial", "B", 11)
	p.SetFillColor(220, 230, 255)
	zeitraum := fmt.Sprintf("%s – %s", r.Von.Format(datumsformat), r.Bis.Format(datumsformat))
	if r.MitPreisen {
		p.CellFormat(150, 9, tr("Gesamtbetrag "+zeitraum), "1", 0, "R", true, 0, "")
		p.CellFormat(20, 9, tr(betrag.Euro(r.GesamtBetrag)), "1", 1, "R", true, 0, "")
	} else {
		p.CellFormat(150, 9, tr("Gesamt "+zeitraum), "1", 0, "R", true, 0, "")
		p.CellFormat(20, 9, fmt.Sprintf("%d", r.GesamtExemplare), "1", 1, "R", true, 0, "")
	}
	p.SetFillColor(255, 255, 255)
}

// zeichneBerichtDetailliste setzt je Topf einen Block mit seinen Bestellungen und seiner Summe,
// darunter die Summe über alle. Die vier Abschnitte einer Bestellung sind eigene Funktionen
// und keine Closures: Eine Closure zählt bei der Komplexität als weitere Ebene.
func zeichneBerichtDetailliste(p *gofpdf.Fpdf, tr func(string) string, toepfe []BestellberichtTopf, r berichtRahmen) {
	if r.Bestellungen == 0 {
		p.SetFont("Arial", "I", 10)
		p.SetTextColor(120, 120, 120)
		p.Cell(0, 8, tr("Keine Bestellungen im gewählten Zeitraum."))
		p.SetTextColor(0, 0, 0)
		return
	}

	p.SetFont("Arial", "B", 11)
	p.Cell(0, 8, tr("Bestellungen im Detail"))
	p.Ln(8)

	spalten := spaltenFuerBericht(r.MitPreisen)
	for _, topf := range toepfe {
		// Ein Topf ohne Bestellung bekommt weder Überschrift noch Summe.
		if len(topf.Bestellungen) == 0 {
			continue
		}
		zeichneBerichtTopfKopf(p, tr, topf.Beschriftung)
		for _, o := range topf.Bestellungen {
			if p.GetY() > 240 {
				p.AddPage()
			}
			zeichneBerichtBestellKopf(p, tr, o)
			zeichneBerichtSpaltenkoepfe(p, tr, spalten, r.MitPreisen)
			zeichneBerichtPositionen(p, tr, o.Positionen, spalten, r.MitPreisen)
			zeichneBerichtBestellSumme(p, tr, o, r.MitPreisen)
		}
		zeichneBerichtTopfSumme(p, tr, topf, r.MitPreisen)
	}

	zeichneBerichtGesamtsumme(p, tr, r)
}

// berichtSumme liefert Betrag und Exemplarzahl einer Gruppe.
func berichtSumme(bestellungen []BestellberichtBestellung) (summe float64, exemplare int) {
	for _, o := range bestellungen {
		summe += o.Gesamtbetrag
		exemplare += o.AnzahlExemplare
	}
	return summe, exemplare
}

// zeichneBerichtTopfKopf setzt die Überschrift eines Topf-Blocks.
func zeichneBerichtTopfKopf(p *gofpdf.Fpdf, tr func(string) string, beschriftung string) {
	if p.GetY() > 235 {
		p.AddPage()
	}
	p.SetFont("Arial", "B", 10)
	p.SetFillColor(235, 240, 250)
	p.CellFormat(170, 7, tr(" "+beschriftung), "1", 1, "L", true, 0, "")
	p.SetFillColor(255, 255, 255)
	p.Ln(2)
}

// zeichneBerichtTopfSumme setzt die Abschlusszeile eines Topf-Blocks — die Zahl, die in die
// Abrechnung dieses Topfs geht.
func zeichneBerichtTopfSumme(p *gofpdf.Fpdf, tr func(string) string, topf BestellberichtTopf, mitPreisen bool) {
	summe, exemplare := berichtSumme(topf.Bestellungen)
	p.SetFont("Arial", "B", 9)
	p.SetFillColor(225, 232, 245)
	beschriftung := fmt.Sprintf("Summe %s (%d Bestellungen, %d Exemplare)",
		topf.Beschriftung, len(topf.Bestellungen), exemplare)
	if mitPreisen {
		p.CellFormat(150, 7, tr(beschriftung), "1", 0, "R", true, 0, "")
		p.CellFormat(20, 7, tr(betrag.Euro(summe)), "1", 1, "R", true, 0, "")
	} else {
		p.CellFormat(170, 7, tr(beschriftung), "1", 1, "R", true, 0, "")
	}
	p.SetFillColor(255, 255, 255)
	p.Ln(6)
}

// ErzeugeBestellbericht setzt den Bericht: Kopf der Schule, Zeitraum, Zusammenfassung, in
// der Jahresansicht die zwei Übersichten, dann die Bestellungen je Topf.
func ErzeugeBestellbericht(b Bestellbericht, schule SchuleInfo) ([]byte, error) {
	p := gofpdf.New("P", "mm", "A4", "")
	p.SetMargins(20, 20, 20)
	p.SetAutoPageBreak(true, 20)
	tr := pdfzeichen.Uebersetzer(p.UnicodeTranslatorFromDescriptor(""))
	p.AddPage()

	// Briefkopf
	p.SetFont("Arial", "B", 12)
	p.Cell(0, 8, tr(schule.Name))
	p.Ln(5)
	p.SetFont("Arial", "", 8)
	p.SetTextColor(100, 100, 100)
	p.Cell(0, 4, tr(schule.Absenderzeile()))
	p.SetTextColor(0, 0, 0)
	p.Ln(12)

	p.SetFont("Arial", "", 10)
	p.CellFormat(0, 6, tr(schule.ortDatum(schulzeit.Jetzt().Format(datumsformat))), "", 1, "R", false, 0, "")
	p.Ln(6)

	// Berichtstitel
	p.SetFont("Arial", "B", 14)
	p.Cell(0, 10, tr(b.Titel))
	p.Ln(7)
	p.SetFont("Arial", "", 10)
	p.SetTextColor(80, 80, 80)
	p.Cell(0, 6, tr(fmt.Sprintf("Zeitraum: %s bis %s", b.Von.Format(datumsformat), b.Bis.Format(datumsformat))))
	p.SetTextColor(0, 0, 0)
	p.Ln(12)

	// Zusammenfassung
	bestellungen := b.bestellungen()
	var gesamtBetrag float64
	var gesamtExemplare int
	for _, o := range bestellungen {
		gesamtBetrag += o.Gesamtbetrag
		gesamtExemplare += o.AnzahlExemplare
	}

	p.SetFillColor(240, 245, 255)
	p.SetFont("Arial", "B", 9)
	if b.MitPreisen {
		p.CellFormat(60, 10, tr(fmt.Sprintf("Bestellungen: %d", len(bestellungen))), "1", 0, "C", true, 0, "")
		p.CellFormat(60, 10, tr(fmt.Sprintf("Exemplare: %d", gesamtExemplare)), "1", 0, "C", true, 0, "")
		p.CellFormat(50, 10, tr("Gesamtbetrag: "+betrag.Euro(gesamtBetrag)), "1", 1, "C", true, 0, "")
	} else {
		p.CellFormat(85, 10, tr(fmt.Sprintf("Bestellungen: %d", len(bestellungen))), "1", 0, "C", true, 0, "")
		p.CellFormat(85, 10, tr(fmt.Sprintf("Exemplare: %d", gesamtExemplare)), "1", 1, "C", true, 0, "")
	}
	p.SetFillColor(255, 255, 255)
	p.Ln(10)

	// Jahresübersicht-Tabellen
	if b.Jahresansicht {
		zeichneBerichtMonate(p, tr, bestellungen, gesamtExemplare, gesamtBetrag, b.MitPreisen)
		zeichneBerichtLieferanten(p, tr, bestellungen, b.MitPreisen)
	}

	// Detailliste
	zeichneBerichtDetailliste(p, tr, b.Toepfe, berichtRahmen{
		Von: b.Von, Bis: b.Bis, Bestellungen: len(bestellungen),
		GesamtBetrag: gesamtBetrag, GesamtExemplare: gesamtExemplare,
		MitPreisen: b.MitPreisen,
	})

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
