package pdf

import (
	"bytes"
	"fmt"

	"bibliothek/pkg/pdfzeichen"
	"bibliothek/pkg/schulzeit"

	"github.com/jung-kurt/gofpdf"
)

// EtikettenWeg sagt, wie der Lieferant an die Aufkleber kommt. Daran hängt ein Satz im
// Anschreiben, und der darf nichts anderes behaupten als das, was der Lieferant vor sich hat.
type EtikettenWeg int

const (
	// OhneEtiketten gilt, wenn es für die Bestellung keine Vorab-Barcodes gibt; im Brief steht
	// dann keine Klebeanweisung.
	OhneEtiketten EtikettenWeg = iota
	// BogenLiegtBei gilt, wenn der Etikettenbogen als PDF an der Mail hängt.
	BogenLiegtBei
	// BogenHinterLink gilt, wenn die Etiketten hinter dem Link der Mail stehen, wo der Lieferant
	// die Größe wählt; der Mail liegt dann kein Bogen bei.
	BogenHinterLink
)

// BestellPosition ist ein bestellter Titel mit seiner Menge.
type BestellPosition struct {
	Titel string
	Autor string
	ISBN  string
	Menge int
}

// Bestellanschreiben ist die Eingabe des Briefs an den Lieferanten. Betreff und Vermerk nennen
// den Topf, aus dem die Bestellung bezahlt wird; welche Texte zu welchem Topf gehören, weiß
// die Tür.
type Bestellanschreiben struct {
	Betreff    string
	Vermerk    string
	Etiketten  EtikettenWeg
	Positionen []BestellPosition
}

// barcodebogenSatz weist den Lieferanten an, die Exemplare vorab zu bekleben. Er steht nur im
// Brief, wenn der Bogen der Mail beiliegt.
const barcodebogenSatz = "Bitte versehen Sie die gelieferten Exemplare vorab mit den Barcode/QR-Code-Aufklebern aus dem beigefügten Bogen.\n"

// linkbogenSatz steht an seiner Stelle, wenn der Bogen hinter dem Link der Mail liegt, und
// nennt den Weg dorthin: Ein Verweis auf eine Anlage, die es nicht gibt, kostet eine Rückfrage.
const linkbogenSatz = "Bitte versehen Sie die gelieferten Exemplare vorab mit den Barcode/QR-Code-Aufklebern. Den Etikettenbogen rufen Sie über den Link in dieser E-Mail ab; dort wählen Sie zwischen kleinen und großen Etiketten.\n"

// bestellanschreibenText baut den Fließtext über der Tabelle. Der Vermerk des Topfs kehrt auf
// der Rechnung wieder, damit die Schule die Rechnungen den Töpfen zuordnen kann.
func bestellanschreibenText(weg EtikettenWeg, vermerk string) string {
	text := "Sehr geehrte Damen und Herren,\n\n" +
		"hiermit bestellen wir die nachfolgend aufgeführten Buchtitel zur Lieferung.\n" +
		vermerk + "\n"
	switch weg {
	case BogenLiegtBei:
		text += barcodebogenSatz
	case BogenHinterLink:
		text += linkbogenSatz
	case OhneEtiketten:
		// Es gibt nichts zu kleben.
	}
	return text + "Die Rechnung senden Sie bitte an die oben angegebene Anschrift und führen Sie darauf denselben Vermerk.\n\n" +
		"Bestellte Titel:"
}

// Maße des Anschreibens in Millimetern. Unter anschreibenSeitenende bricht gofpdf von sich aus
// um; anschreibenGrussHoehe ist der Platz von Abstand, Gruß und Unterschrift.
const (
	anschreibenRand        = 20.0
	anschreibenSeitenende  = 297.0 - anschreibenRand
	anschreibenKopfHoehe   = 8.0
	anschreibenZeilenHoehe = 7.0
	anschreibenGrussHoehe  = 15.0 + 6 + 12 + 6
	anschreibenSpalteTitel = 75.0
	anschreibenSpalteAutor = 40.0
	anschreibenSpalteISBN  = 35.0
	anschreibenSpalteMenge = 20.0
)

// anschreibenSpaltenkoepfe setzt die Köpfe der Tabelle und stellt die Schrift der Zeilen ein.
func anschreibenSpaltenkoepfe(p *gofpdf.Fpdf, tr func(string) string) {
	p.SetFont("Arial", "B", 9)
	p.SetFillColor(230, 230, 230)
	p.CellFormat(anschreibenSpalteTitel, anschreibenKopfHoehe, tr("Buchtitel"), "1", 0, "L", true, 0, "")
	p.CellFormat(anschreibenSpalteAutor, anschreibenKopfHoehe, tr("Autor"), "1", 0, "L", true, 0, "")
	p.CellFormat(anschreibenSpalteISBN, anschreibenKopfHoehe, tr("ISBN"), "1", 0, "L", true, 0, "")
	p.CellFormat(anschreibenSpalteMenge, anschreibenKopfHoehe, tr("Menge"), "1", 1, "C", true, 0, "")
	p.SetFont("Arial", "", 9)
}

// GenerateBestellanschreibenPDF setzt das Anschreiben an den Lieferanten mit der Tabelle der
// bestellten Titel.
func GenerateBestellanschreibenPDF(b Bestellanschreiben, schule SchuleInfo) ([]byte, error) {
	p := gofpdf.New("P", "mm", "A4", "")
	p.SetMargins(anschreibenRand, anschreibenRand, anschreibenRand)
	p.SetAutoPageBreak(true, anschreibenRand)
	p.AddPage()
	// Der Kopf der ersten Seite beginnt 10 mm unter der Kante, Folgeseiten am Rand.
	p.SetY(10)
	tr := pdfzeichen.Uebersetzer(p.UnicodeTranslatorFromDescriptor(""))

	// Kopf mit Name und Absenderzeile der Schule.
	p.SetFont("Arial", "B", 12)
	p.Cell(0, 8, tr(schule.Name))
	p.Ln(5)
	p.SetFont("Arial", "", 8)
	p.SetTextColor(100, 100, 100)
	p.Cell(0, 4, tr(schule.Absenderzeile()))
	p.SetTextColor(0, 0, 0)
	p.Ln(15)

	p.SetFont("Arial", "", 10)
	p.CellFormat(0, 6, tr(schule.OrtDatum(schulzeit.Jetzt().Format(dateFormatDE))), "", 0, "R", false, 0, "")
	p.Ln(10)

	p.SetFont("Arial", "B", 9)
	p.Cell(0, 4, tr("An den Buchlieferanten"))
	p.Ln(20)

	// Der Topf steht schon in der Betreffzeile, nicht erst im Fließtext.
	p.SetFont("Arial", "B", 12)
	p.Cell(0, 8, tr(b.Betreff))
	p.Ln(10)

	p.SetFont("Arial", "", 10)
	p.MultiCell(0, 5, tr(bestellanschreibenText(b.Etiketten, b.Vermerk)), "", "L", false)
	p.Ln(6)

	// Die Köpfe stehen nicht allein am Fuß einer Seite.
	if len(b.Positionen) > 0 && p.GetY()+anschreibenKopfHoehe+anschreibenZeilenHoehe > anschreibenSeitenende {
		p.AddPage()
	}
	anschreibenSpaltenkoepfe(p, tr)
	for _, item := range b.Positionen {
		// Jede Seite mit Positionen trägt die Köpfe.
		if p.GetY()+anschreibenZeilenHoehe > anschreibenSeitenende {
			p.AddPage()
			anschreibenSpaltenkoepfe(p, tr)
		}
		p.CellFormat(anschreibenSpalteTitel, anschreibenZeilenHoehe, tr(kuerzeAufZelle(p, tr, item.Titel, anschreibenSpalteTitel)), "1", 0, "L", false, 0, "")
		p.CellFormat(anschreibenSpalteAutor, anschreibenZeilenHoehe, tr(kuerzeAufZelle(p, tr, item.Autor, anschreibenSpalteAutor)), "1", 0, "L", false, 0, "")
		p.CellFormat(anschreibenSpalteISBN, anschreibenZeilenHoehe, tr(kuerzeAufZelle(p, tr, item.ISBN, anschreibenSpalteISBN)), "1", 0, "L", false, 0, "")
		p.CellFormat(anschreibenSpalteMenge, anschreibenZeilenHoehe, fmt.Sprintf("%d", item.Menge), "1", 1, "C", false, 0, "")
	}

	// Gruß und Unterschrift stehen zusammen auf einer Seite.
	if p.GetY()+anschreibenGrussHoehe > anschreibenSeitenende {
		p.AddPage()
	}
	p.Ln(15)

	p.SetFont("Arial", "", 10)
	p.Cell(0, 6, tr("Mit freundlichen Grüßen,"))
	p.Ln(12)
	p.SetFont("Arial", "B", 10)
	p.Cell(0, 6, tr("Das Bibliotheksteam"))

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
