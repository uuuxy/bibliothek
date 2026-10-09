package pdf

import (
	"bytes"
	"fmt"

	"bibliothek/pkg/coverdatei"
	"bibliothek/pkg/pdfzeichen"
	"bibliothek/pkg/schulzeit"
	"bibliothek/pkg/strichcode"

	"github.com/jung-kurt/gofpdf"
)

// MahnlisteMedium ist eine Zeile der Mahnliste: ein Buch über der Frist.
type MahnlisteMedium struct {
	Titel string
	Autor string
	// Barcode steht als Bild und als Nummer in der Zeile, damit das Buch bei der Rückgabe vom
	// Blatt gescannt werden kann.
	Barcode string
	// CoverURL ist die Adresse des Covers am Titel; gedruckt wird nur ein lokal gespeichertes Bild.
	CoverURL string
	// FaelligAm ist der Tag der Frist als fertiger Text.
	FaelligAm        string
	TageUeberfaellig int
}

// MahnlisteSchueler ist eine Seite der Mahnliste: ein Schüler mit seinen Büchern über der Frist.
type MahnlisteSchueler struct {
	Name   string
	Klasse string
	Medien []MahnlisteMedium
}

// Maße der Mahnliste in Millimetern. Unter mahnlisteSeitenende bricht gofpdf von sich aus um;
// mahnlisteFussHoehe ist der Platz der Fußzeile samt Abstand unter der letzten Zeile.
const (
	mahnlisteRandUnten   = 20.0
	mahnlisteSeitenende  = 297.0 - mahnlisteRandUnten
	mahnlisteZeilenHoehe = 18.0
	mahnlisteFussHoehe   = 15.0
	mahnlisteSpalteTitel = 52.0
	mahnlisteSpalteAutor = 26.0
)

// zeichneMahnMedienZeile setzt die Zeile eines Buchs: Cover, Titel, Autor, Barcode, Frist und
// die Tage über der Frist, ab 15 Tagen in Rot.
func zeichneMahnMedienZeile(pdf *gofpdf.Fpdf, tr func(string) string, med MahnlisteMedium, rowHeight float64) {
	startY := pdf.GetY()

	coverdatei.BindeEin(pdf, med.CoverURL, coverdatei.CoverPlatz{X: 18, Y: startY + 0.5, Breite: 7, Hoehe: rowHeight - 1})

	// Den Rahmen der Cover-Zelle gibt es auch ohne Bild.
	pdf.SetXY(18, startY)
	pdf.CellFormat(8, rowHeight, "", "1", 0, "", false, 0, "")

	titleCell := kuerzeAufZelle(pdf, tr, med.Titel, mahnlisteSpalteTitel)
	pdf.CellFormat(mahnlisteSpalteTitel, rowHeight, tr(titleCell), "1", 0, "L", false, 0, "")

	autorCell := kuerzeAufZelle(pdf, tr, med.Autor, mahnlisteSpalteAutor)
	pdf.CellFormat(mahnlisteSpalteAutor, rowHeight, tr(autorCell), "1", 0, "L", false, 0, "")

	bcX := pdf.GetX()
	pdf.CellFormat(40, rowHeight, "", "1", 0, "", false, 0, "")
	if med.Barcode != "" {
		if pngBytes, err := strichcode.PNG(med.Barcode, false, 300, 80); err == nil {
			imgName := "bc_" + med.Barcode
			opt := gofpdf.ImageOptions{ImageType: "PNG"}
			pdf.RegisterImageOptionsReader(imgName, opt, bytes.NewReader(pngBytes))
			pdf.ImageOptions(imgName, bcX+3, startY+2.5, 34, 8, false, opt, 0, "")
		}
		pdf.SetFont("Courier", "", 7)
		pdf.SetXY(bcX, startY+11.5)
		pdf.CellFormat(40, 4, tr(med.Barcode), "", 0, "C", false, 0, "")
		pdf.SetFont("Arial", "", 8)
	}

	// Die Nummer unter dem Strichcode hat die Position verschoben.
	pdf.SetXY(144, startY)
	pdf.CellFormat(22, rowHeight, tr(med.FaelligAm), "1", 0, "C", false, 0, "")

	if med.TageUeberfaellig > 14 {
		pdf.SetTextColor(200, 30, 30)
		pdf.SetFont("Arial", "B", 8)
	}
	pdf.CellFormat(26, rowHeight, fmt.Sprintf("%d Tage", med.TageUeberfaellig), "1", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 8)
}

// zeichneMahnSeite setzt die Seiten eines Schülers: Kopf, Name und Klasse, die Tabelle seiner
// Bücher und die Fußzeile.
func zeichneMahnSeite(pdf *gofpdf.Fpdf, tr func(string) string, sch MahnlisteSchueler) {
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 9, tr("Mahnung – Schulbibliothek"))
	pdf.Ln(7)

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(120, 120, 120)
	pdf.Cell(0, 5, tr(fmt.Sprintf("Erstellt am %s", schulzeit.Jetzt().Format(dateFormatDE))))
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(10)

	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(240, 245, 255)
	pdf.SetDrawColor(180, 195, 230)
	pdf.RoundedRect(18, pdf.GetY(), 174, 22, 3, "1234", "FD")
	pdf.SetXY(24, pdf.GetY()+4)
	pdf.Cell(60, 7, tr("Schüler/in:"))
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 7, tr(sch.Name))
	pdf.SetXY(24, pdf.GetY()+8)
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(60, 7, tr("Klasse:"))
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 7, tr(sch.Klasse))
	pdf.SetXY(18, pdf.GetY()+12)

	pdf.Ln(6)
	pdf.SetFont("Arial", "I", 9)
	pdf.SetTextColor(160, 60, 60)
	pdf.Cell(0, 5, tr(fmt.Sprintf(
		"Bitte gib die folgenden %d %s umgehend in der Schulbibliothek ab.",
		len(sch.Medien),
		pluralMedium(len(sch.Medien)),
	)))
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(8)

	zeichneMahnSpaltenkoepfe(pdf, tr)
	for _, med := range sch.Medien {
		// Eine Zeile setzt Cover, Strichcode und Nummer an feste Stellen; ein Umbruch mitten
		// in ihr verteilte sie über drei Seiten. Passt sie mit der Fußzeile nicht mehr auf
		// die Seite, beginnt sie auf der nächsten.
		if pdf.GetY()+mahnlisteZeilenHoehe+mahnlisteFussHoehe > mahnlisteSeitenende {
			pdf.AddPage()
			zeichneMahnFortsetzung(pdf, tr, sch)
		}
		zeichneMahnMedienZeile(pdf, tr, med, mahnlisteZeilenHoehe)
	}

	pdf.Ln(10)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(130, 130, 130)
	pdf.Cell(0, 5, tr("Schulbibliothek – Bei Fragen wende dich bitte an das Bibliotheksteam."))
	pdf.SetTextColor(0, 0, 0)
}

// zeichneMahnSpaltenkoepfe setzt die Köpfe der Tabelle und stellt die Schrift der Zeilen ein.
func zeichneMahnSpaltenkoepfe(pdf *gofpdf.Fpdf, tr func(string) string) {
	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(220, 225, 240)
	pdf.CellFormat(8, 8, "", "1", 0, "C", true, 0, "")
	pdf.CellFormat(mahnlisteSpalteTitel, 8, tr("Buchtitel"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(mahnlisteSpalteAutor, 8, tr("Autor"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(40, 8, tr("Barcode"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(22, 8, tr("Fällig"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(26, 8, tr("Tage überfällig"), "1", 1, "C", true, 0, "")
	pdf.SetFont("Arial", "", 8)
}

// zeichneMahnFortsetzung beginnt eine Folgeseite: Die Seiten eines Schülers werden einzeln
// ausgeteilt, deshalb nennt jede, wem sie gehört.
func zeichneMahnFortsetzung(pdf *gofpdf.Fpdf, tr func(string) string, sch MahnlisteSchueler) {
	zeile := "Fortsetzung: " + sch.Name
	if sch.Klasse != "" {
		zeile += ", " + sch.Klasse
	}
	pdf.SetFont("Arial", "B", 9)
	pdf.Cell(0, 5, tr(zeile))
	pdf.Ln(8)
	zeichneMahnSpaltenkoepfe(pdf, tr)
}

// GenerateMahnlistePDF setzt die Mahnliste auf A4, je Schüler eine Seite und bei mehr als zehn
// Büchern Folgeseiten. Ohne Schüler trägt das Blatt einen Satz, der das sagt.
func GenerateMahnlistePDF(schueler []MahnlisteSchueler) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(18, 18, 18)
	pdf.SetAutoPageBreak(true, mahnlisteRandUnten)
	tr := pdfzeichen.Uebersetzer(pdf.UnicodeTranslatorFromDescriptor(""))

	for _, sch := range schueler {
		pdf.AddPage()
		zeichneMahnSeite(pdf, tr, sch)
	}

	if len(schueler) == 0 {
		pdf.AddPage()
		pdf.SetFont("Arial", "", 12)
		pdf.SetTextColor(130, 130, 130)
		pdf.Cell(0, 10, tr("Keine überfälligen Ausleihen vorhanden."))
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// pluralMedium nennt das Wort zur Zahl der Bücher.
func pluralMedium(n int) string {
	if n == 1 {
		return "Medium"
	}
	return "Medien"
}
