package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// SchadensfallInfo contains all required data for the damage PDF generation.
type SchadensfallInfo struct {
	Beschreibung     string
	Betrag           float64
	ErstelltAm       time.Time
	SchuelerVorname  string
	SchuelerNachname string
	SchuelerKlasse   string
	BuchTitel        string
	ExemplarBarcode  string
}

// GenerateSchadensfallPDF generates a formal PDF notification letter ("Elternbrief")
// for a student responsible for library book damage.
func GenerateSchadensfallPDF(data SchadensfallInfo, schule SchuleInfo) ([]byte, error) {
	// Create new A4 PDF page in portrait mode
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetMargins(20, 20, 20)

	// UTF-8 to ISO-8859-1 conversion to support German umlauts (ä, ö, ü, ß) in standard PDF fonts
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	addSchadensfallHeader(pdf, schule, tr)
	addSchadensfallAddress(pdf, data, tr)
	addSchadensfallBody(pdf, data, tr)
	addSchadensfallSignatures(pdf, tr)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func addSchadensfallHeader(pdf *gofpdf.Fpdf, schule SchuleInfo, tr func(string) string) {
	// Letter Header
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 10, tr(schule.Name))
	pdf.Ln(6)
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 4, tr(schule.Absenderzeile()))
	pdf.SetTextColor(0, 0, 0)

	// Date line (Right-aligned)
	pdf.SetY(40)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 6, schule.OrtDatum(time.Now().Format(dateFormatDE)), "", 0, "R", false, 0, "")
}

func addSchadensfallAddress(pdf *gofpdf.Fpdf, data SchadensfallInfo, tr func(string) string) {
	// DIN 5008 Address Window (approx. 45mm from top)
	pdf.SetXY(20, 45)
	pdf.SetFont("Arial", "B", 9)
	pdf.Cell(0, 4, tr("An die Erziehungsberechtigten von:"))
	pdf.Ln(5)
	pdf.SetX(20)
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 6, tr(fmt.Sprintf("%s %s", data.SchuelerVorname, data.SchuelerNachname)))
	pdf.Ln(5)
	pdf.SetX(20)
	pdf.Cell(0, 6, tr("_________________________"))
	pdf.Ln(5)
	pdf.SetX(20)
	pdf.Cell(0, 6, tr("_________________________"))
	pdf.Ln(30)
}

func addSchadensfallBody(pdf *gofpdf.Fpdf, data SchadensfallInfo, tr func(string) string) {
	// Letter Subject
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, tr("Ersatzforderung für ein beschädigtes oder verlorenes Bibliotheksbuch"))
	pdf.Ln(12)

	// Introduction Body
	pdf.SetFont("Arial", "", 10)
	introText := fmt.Sprintf("Sehr geehrte Erziehungsberechtigte,\n\n"+
		"bei der Rückgabe bzw. Überprüfung des von Ihrem Kind ausgeliehenen Schulbuches "+
		"\"%s\" (Barcode: %s) wurde am %s folgende Beschädigung oder Verlust festgestellt:\n\n",
		data.BuchTitel, data.ExemplarBarcode, data.ErstelltAm.Format(dateFormatDE))
	pdf.MultiCell(0, 5, tr(introText), "", "L", false)

	// Damage Description Box
	pdf.SetFillColor(245, 245, 245)
	pdf.SetFont("Arial", "I", 10)
	pdf.CellFormat(0, 10, tr(fmt.Sprintf("   Schadensfall: %s", data.Beschreibung)), "1", 1, "L", true, 0, "")
	pdf.Ln(6)

	// Resolution guidelines and payment request
	pdf.SetFont("Arial", "", 10)
	dueTime := time.Now().AddDate(0, 0, 14).Format(dateFormatDE)
	instructions := fmt.Sprintf("Gemäß der Schulbibliotheksordnung bitten wir Sie, für den entstandenen Schaden "+
		"einen Ersatzbetrag von %.2f EUR bis spätestens zum %s zu begleichen.\n\n"+
		"Bitte bezahlen Sie den Betrag bar in der Bibliothek zu den Öffnungszeiten.\n\n"+
		"Sollten Sie Fragen zum Schadensfall haben, können Sie sich gerne zu den Öffnungszeiten "+
		"an das Bibliotheksteam wenden.\n\n"+
		"Vielen Dank für Ihr Verständnis und Ihre Kooperation.",
		data.Betrag, dueTime)
	pdf.MultiCell(0, 5, tr(instructions), "", "L", false)
	pdf.Ln(15)
}

func addSchadensfallSignatures(pdf *gofpdf.Fpdf, tr func(string) string) {
	// Signatures
	pdf.Cell(0, 6, tr("Mit freundlichen Grüßen,"))
	pdf.Ln(15)
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(0, 6, tr("Die Bibliotheksleitung"))
}
