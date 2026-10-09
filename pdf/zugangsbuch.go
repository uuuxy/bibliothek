package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"

	"bibliothek/pkg/pdfzeichen"
)

// Der Ausdruck des Zugangsbuchs. Aufbau wie beim Abgangsbuch (abgangsbuch.go): ein
// Abschnitt je Topf, jeder mit eigener Stückzahl, leere werden übersprungen.
//
// Die Spalten sind die der Arbeitshilfe: Eingangsdatum, Inventarnummer, Titel, Lieferant
// (docs/mittel_konzept.md 7.1). „Anzahl" ist die Stückzahl des Abschnitts — jede Zeile ist
// EIN Exemplar mit seiner eigenen Nummer, denn genau die braucht die Bestandskartei.

const (
	zugangSpalteDatum     = 24.0
	zugangSpalteNummer    = 30.0
	zugangSpalteTitel     = 74.0
	zugangSpalteLieferant = 52.0
)

// ZugangsZeile ist ein Zugang, wie er auf dem Blatt steht.
type ZugangsZeile struct {
	Datum     time.Time
	Barcode   string
	Titel     string
	Lieferant string
}

// ZugangsAbschnitt ist ein Topf des Blatts mit seiner Überschrift.
type ZugangsAbschnitt struct {
	Titel  string
	Zeilen []ZugangsZeile
}

// Zugangsbuch ist die Eingabe des Blatts. Gliederung, Überschriften und Zahlen kommen von der
// Tür; sie sind dieselben wie in der Liste auf dem Bildschirm.
type Zugangsbuch struct {
	Von        time.Time
	Bis        time.Time
	Abschnitte []ZugangsAbschnitt
	// Gesamt ist die Zahl der Zugänge im Zeitraum.
	Gesamt int
	// OhneZuordnung sagt, dass es Zugänge ohne hinterlegte Bestellung gibt. Das Blatt erklärt
	// dann unter der Liste, was ihr Abschnitt bedeutet.
	OhneZuordnung bool
}

// GenerateZugangsbuchPDF setzt das Zugangsbuch eines Zeitraums als Blatt zum Abheften.
func GenerateZugangsbuchPDF(buch Zugangsbuch, schule SchuleInfo) ([]byte, error) {
	p := gofpdf.New("P", "mm", "A4", "")
	p.SetMargins(20, 20, 20)
	p.SetAutoPageBreak(true, 20)
	tr := pdfzeichen.Uebersetzer(p.UnicodeTranslatorFromDescriptor(""))
	p.AddPage()

	bestandsbuchKopf(p, tr, "Zugangsbuch", buch.Von, buch.Bis, schule)

	gezeigt := 0
	for _, abschnitt := range buch.Abschnitte {
		gezeigt += zugangsbuchAbschnitt(p, tr, abschnitt)
	}
	if gezeigt == 0 {
		p.SetFont("Arial", "I", 10)
		p.SetTextColor(120, 120, 120)
		p.Cell(0, 8, tr("In diesem Zeitraum ist kein Exemplar in den Bestand gekommen."))
		p.SetTextColor(0, 0, 0)
		p.Ln(10)
	}

	zugangsbuchFuss(p, tr, buch.Gesamt, buch.OhneZuordnung)

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func zugangsbuchAbschnitt(p *gofpdf.Fpdf, tr func(string) string, abschnitt ZugangsAbschnitt) int {
	zeilen := abschnitt.Zeilen
	if len(zeilen) == 0 {
		return 0
	}
	if p.GetY() > abgangUmbruchAbY-20 {
		p.AddPage()
	}
	p.SetFont("Arial", "B", 10)
	p.SetFillColor(235, 240, 250)
	p.CellFormat(180, 7, tr(" "+abschnitt.Titel), "1", 1, "L", true, 0, "")
	p.SetFillColor(255, 255, 255)
	p.Ln(2)

	zugangsbuchSpaltenkoepfe(p, tr)
	for _, z := range zeilen {
		if p.GetY() > abgangUmbruchAbY {
			p.AddPage()
			zugangsbuchSpaltenkoepfe(p, tr)
		}
		p.SetFont("Arial", "", 9)
		p.CellFormat(zugangSpalteDatum, abgangZeilenHoehe, tr(z.Datum.Format(dateFormatDE)), "LB", 0, "L", false, 0, "")
		p.CellFormat(zugangSpalteNummer, abgangZeilenHoehe, tr(z.Barcode), "B", 0, "L", false, 0, "")
		p.CellFormat(zugangSpalteTitel, abgangZeilenHoehe, tr(pdfzeichen.KuerzeAufZelle(p, tr, z.Titel, zugangSpalteTitel)), "B", 0, "L", false, 0, "")
		p.CellFormat(zugangSpalteLieferant, abgangZeilenHoehe, tr(pdfzeichen.KuerzeAufZelle(p, tr, z.Lieferant, zugangSpalteLieferant)), "BR", 1, "L", false, 0, "")
	}

	p.SetFont("Arial", "B", 9)
	p.SetFillColor(225, 232, 245)
	p.CellFormat(180, 7, tr(fmt.Sprintf("Summe %s: %d Exemplare  ", abschnitt.Titel, len(zeilen))), "1", 1, "R", true, 0, "")
	p.SetFillColor(255, 255, 255)
	p.Ln(6)
	return len(zeilen)
}

func zugangsbuchSpaltenkoepfe(p *gofpdf.Fpdf, tr func(string) string) {
	p.SetFont("Arial", "B", 9)
	p.CellFormat(zugangSpalteDatum, abgangZeilenHoehe, tr("Zugang"), "LTB", 0, "L", false, 0, "")
	p.CellFormat(zugangSpalteNummer, abgangZeilenHoehe, tr("Nummer"), "TB", 0, "L", false, 0, "")
	p.CellFormat(zugangSpalteTitel, abgangZeilenHoehe, tr("Titel"), "TB", 0, "L", false, 0, "")
	p.CellFormat(zugangSpalteLieferant, abgangZeilenHoehe, tr("Lieferant"), "TBR", 1, "L", false, 0, "")
}

// zugangsbuchFuss nennt die Gesamtzahl — und erklärt den Abschnitt „ohne Zuordnung", wenn es
// ihn gibt.
//
// Diese Erklärung gehört auf das Blatt, weil sie eine Einschränkung des Nachweises ist: Ein
// Exemplar ohne hinterlegte Bestellung trägt als Zugangsdatum den Tag, an dem es im Programm
// entstanden ist — bei einer Bestandskorrektur ist das nicht zwingend der Tag, an dem das
// Buch in die Schule kam.
func zugangsbuchFuss(p *gofpdf.Fpdf, tr func(string) string, gesamt int, ohneBestellung bool) {
	if p.GetY() > abgangUmbruchAbY {
		p.AddPage()
	}
	p.SetFont("Arial", "B", 11)
	p.SetFillColor(220, 230, 255)
	p.CellFormat(180, 9, tr(fmt.Sprintf("Zugänge im Zeitraum: %d Exemplare  ", gesamt)), "1", 1, "R", true, 0, "")
	p.SetFillColor(255, 255, 255)

	if ohneBestellung {
		p.Ln(4)
		p.SetFont("Arial", "I", 9)
		p.SetTextColor(90, 90, 90)
		p.MultiCell(180, 5, tr("Hinweis: Zu den Exemplaren im Abschnitt „ohne Zuordnung\" ist keine Bestellung "+
			"hinterlegt — Altbestand, Handanlage oder Bestandskorrektur. Aus welchen Mitteln sie bezahlt "+
			"wurden, ist deshalb nicht belegt, und ihr Zugangsdatum ist der Tag, an dem sie im Programm "+
			"angelegt wurden."), "", "L", false)
		p.SetTextColor(0, 0, 0)
	}
}
