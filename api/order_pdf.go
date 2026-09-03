package api

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"bibliothek/pdf"
	"bibliothek/pkg/csvutil"
	"bibliothek/pkg/schulzeit"

	"github.com/jung-kurt/gofpdf"
)

// OrderedItem represents a single book title ordered with its quantity.
type OrderedItem struct {
	Titel  string
	Autor  string
	ISBN   string
	Verlag string
	Menge  int
}

// BarcodeLabelDetail holds data needed to print a barcode label.
type BarcodeLabelDetail struct {
	BarcodeID string
	Titel     string
	Autor     string
	ISBN      string
	// AnschaffungsJahr ist das Jahr aus buecher_exemplare.erworben_am und steht als
	// "Ansch.J. 2016" auf dem Etikett. Leer = Zeile entfällt (etwa bei Vorab-Etiketten
	// für eine Bestellung, deren Exemplare es noch gar nicht gibt).
	AnschaffungsJahr string
	// Signatur ist buecher_titel.signatur (z. B. "LMF-Deutsch 5"). Leer = Zeile entfällt,
	// genau wie bei AnschaffungsJahr.
	Signatur string
}

// EtikettKopf trägt die schulweiten Angaben, die auf JEDEM Etikett stehen — sie kommen
// aus den Systemeinstellungen und nicht aus dem einzelnen Exemplar.
//
// Bis zum 04.08.2026 stand hier fest "Schulbibliothek" im Code. Das ist auf einem
// Buchetikett die falsche Angabe: Es soll die Schule benennen, der das Buch gehört,
// und im Verlustfall den Weg zurück zeigen.
type EtikettKopf struct {
	Schulname        string // z. B. "Philipp-Reis-Schule, Friedrichsdorf"
	Eigentumsvermerk string // z. B. "Eigentum des Landes Hessen"
}

// etikettenWeg beschreibt, WIE der Lieferant an die Aufkleber kommt. Er entscheidet über
// EINEN Satz im Anschreiben — und dieser Satz darf niemals etwas anderes behaupten als
// das, was der Lieferant tatsächlich vor sich hat.
type etikettenWeg int

const (
	// ohneEtiketten: Für diese Bestellung wurden keine Vorab-Barcodes erzeugt. Dann steht
	// im Brief auch keine Klebeanweisung.
	ohneEtiketten etikettenWeg = iota
	// bogenLiegtBei: Der fertige Etikettenbogen hängt als PDF an der Mail.
	bogenLiegtBei
	// bogenHinterLink: Die Etiketten stehen hinter dem Bestätigungs-Link, wo der
	// Hauptlieferant die Größe selbst wählt. Der Mail liegt dann bewusst KEIN Bogen bei
	// (siehe bestellAnhaenge).
	bogenHinterLink
)

// barcodebogenSatz ist die Anweisung an den Lieferanten, die Exemplare vorab zu
// bekleben. Sie darf nur im Brief stehen, wenn der Bogen der E-Mail auch beiliegt.
const barcodebogenSatz = "Bitte versehen Sie die gelieferten Exemplare vorab mit den Barcode/QR-Code-Aufklebern aus dem beigefügten Bogen.\n"

// linkbogenSatz ersetzt den Satz oben, wenn der Bogen nicht beiliegt, sondern hinter dem
// Bestätigungs-Link steht. Er nennt den Weg dorthin — ein Brief, der auf eine "beigefügte"
// Anlage verweist, die es nicht gibt, kostet die Lieferung eine Rückfrage.
const linkbogenSatz = "Bitte versehen Sie die gelieferten Exemplare vorab mit den Barcode/QR-Code-Aufklebern. Den Etikettenbogen rufen Sie über den Link in dieser E-Mail ab; dort wählen Sie zwischen kleinen und großen Etiketten.\n"

// bestellAnschreibenText baut den Fliesstext des Anschreibens.
//
// Eigene Funktion, damit der Satz mit dem "beigefügten Bogen" prüfbar ist, ohne ein
// erzeugtes PDF wieder auseinanderzunehmen: Genau dieser Satz stand vorher bedingungslos
// im Brief und verwies auf eine Anlage, die oft nicht existierte.
func bestellAnschreibenText(weg etikettenWeg) string {
	text := "Sehr geehrte Damen und Herren,\n\n" +
		"hiermit bestellen wir für unsere Schulbibliothek die nachfolgend aufgeführten Buchtitel zur Lieferung.\n"
	switch weg {
	case bogenLiegtBei:
		text += barcodebogenSatz
	case bogenHinterLink:
		text += linkbogenSatz
	case ohneEtiketten:
		// Keine Klebeanweisung — es gibt nichts zu kleben.
	}
	return text + "Die Rechnung senden Sie bitte an die oben angegebene Anschrift.\n\n" +
		"Bestellte Titel:"
}

// GenerateOrderSummaryPDF generates a PDF cover letter ("Bestellanschreiben") containing the table of ordered book titles.
//
// weg entscheidet über EINEN Satz im Anschreiben — den mit dem Etikettenbogen. Er stand
// vorher immer drin, auch wenn der Bogen gar nicht mitgeschickt wurde. Der Lieferant bekam
// damit eine Anweisung auf etwas, das der E-Mail nicht beilag: Er kann sie nur ignorieren
// oder nachfragen, beides kostet die Lieferung Zeit.
func GenerateOrderSummaryPDF(items []OrderedItem, schule pdf.SchuleInfo, weg etikettenWeg) ([]byte, error) {
	p := gofpdf.New("P", "mm", "A4", "")
	p.AddPage()
	p.SetMargins(20, 20, 20)
	tr := p.UnicodeTranslatorFromDescriptor("")

	// Letter Header / Sender info
	p.SetFont("Arial", "B", 12)
	p.Cell(0, 8, tr(schule.Name))
	p.Ln(5)
	p.SetFont("Arial", "", 8)
	p.SetTextColor(100, 100, 100)
	p.Cell(0, 4, tr(schule.Absenderzeile()))
	p.SetTextColor(0, 0, 0)
	p.Ln(15)

	// Date (Right-aligned)
	p.SetFont("Arial", "", 10)
	p.CellFormat(0, 6, schule.OrtDatum(schulzeit.Jetzt().Format("02.01.2006")), "", 0, "R", false, 0, "")
	p.Ln(10)

	// Recipient Block
	p.SetFont("Arial", "B", 9)
	p.Cell(0, 4, tr("An den Buchlieferanten"))
	p.Ln(20)

	// Subject
	p.SetFont("Arial", "B", 12)
	p.Cell(0, 8, tr("Buchbestellung für die Schulbibliothek"))
	p.Ln(10)

	// Letter Body Text
	p.SetFont("Arial", "", 10)
	p.MultiCell(0, 5, tr(bestellAnschreibenText(weg)), "", "L", false)
	p.Ln(6)

	// Table headers
	p.SetFont("Arial", "B", 9)
	p.SetFillColor(230, 230, 230)
	p.CellFormat(75, 8, tr("Buchtitel"), "1", 0, "L", true, 0, "")
	p.CellFormat(40, 8, tr("Autor"), "1", 0, "L", true, 0, "")
	p.CellFormat(35, 8, tr("ISBN"), "1", 0, "L", true, 0, "")
	p.CellFormat(20, 8, tr("Menge"), "1", 1, "C", true, 0, "")

	p.SetFont("Arial", "", 9)
	for _, item := range items {
		p.CellFormat(75, 7, tr(item.Titel), "1", 0, "L", false, 0, "")
		p.CellFormat(40, 7, tr(item.Autor), "1", 0, "L", false, 0, "")
		p.CellFormat(35, 7, tr(item.ISBN), "1", 0, "L", false, 0, "")
		p.CellFormat(20, 7, fmt.Sprintf("%d", item.Menge), "1", 1, "C", false, 0, "")
	}
	p.Ln(15)

	// Sign-off
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

// GenerateBarcodeCSV creates a CSV containing the barcode to ISBN mapping for the supplier.
func GenerateBarcodeCSV(labels []BarcodeLabelDetail) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Comma = ';' // European CSV format

	// Header
	if err := writer.Write([]string{"ISBN", "Titel", "Autor", "Barcode"}); err != nil {
		return nil, err
	}

	for _, l := range labels {
		// Schutz vor Formel-Injection (CWE-1236): Titel/Autor stammen aus Katalog-Importen.
		if err := writer.Write(csvutil.SanitizeRow([]string{l.ISBN, l.Titel, l.Autor, l.BarcodeID})); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buf.Bytes(), writer.Error()
}

// GenerateSingleLabelPDFA6 generates a single A6 PDF label for an exemplar.
func GenerateSingleLabelPDFA6(label BarcodeLabelDetail) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A6", "")
	pdf.AddPage()
	pdf.SetMargins(10, 10, 10)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, tr("Ersatz-Etikett"), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// Draw border box
	pdf.Rect(10, 25, 85, 110, "D")

	pdf.SetFont("Arial", "B", 12)
	pdf.SetXY(12, 28)
	pdf.MultiCell(81, 6, tr(label.Titel), "", "C", false)

	pdf.SetFont("Arial", "", 10)
	pdf.SetXY(12, 45)
	pdf.MultiCell(81, 5, tr(label.Autor), "", "C", false)

	// Generate 1D Code39 barcode
	barcodeImg, err := GenerateBarcodePNG(label.BarcodeID, false, 250, 70)
	if err == nil {
		imgReader := bytes.NewReader(barcodeImg)
		pdf.RegisterImageOptionsReader(label.BarcodeID, gofpdf.ImageOptions{ImageType: "PNG"}, imgReader)
		bcWidth := 70.0
		bcHeight := 18.0
		bcX := 10.0 + (85.0-bcWidth)/2.0
		bcY := 60.0
		pdf.Image(label.BarcodeID, bcX, bcY, bcWidth, bcHeight, false, "", 0, "")
	}

	pdf.SetFont("Arial", "B", 14)
	pdf.SetXY(12, 105)
	pdf.CellFormat(81, 6, tr(label.BarcodeID), "", 0, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
