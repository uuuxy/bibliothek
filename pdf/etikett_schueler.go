package pdf

import (
	"bibliothek/pkg/pdfzeichen"
	"bibliothek/pkg/strichcode"
	"bytes"
	"fmt"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

// Etiketten für Schüler: ein Klebeetikett mit Name, Klasse und Barcode.
//
// Sie nehmen dieselben Bögen und dasselbe Raster wie die Etiketten der Bücher (zeichneRaster
// in etikett_buch.go): Es ist dieselbe Sorte Klebebogen aus demselben Schrank, und eine
// eigene Rechnung des Rasters liefe früher oder später um eine halbe Zeile auseinander.

// SchuelerEtikett ist ein Etikett des Bogens. Die Felder kommen aus der Datenbank, nicht aus der
// Anfrage, die nur Kennungen schickt; sonst könnte jeder mit view_students beliebige Namen auf
// einen Bogen mit echten Barcodes drucken.
type SchuelerEtikett struct {
	BarcodeID string
	Vorname   string
	Nachname  string
	Klasse    string
}

// name liefert "Nachname, Vorname" — die Form, in der die Theke sucht und sortiert.
func (e SchuelerEtikett) name() string {
	nach := strings.TrimSpace(e.Nachname)
	vor := strings.TrimSpace(e.Vorname)
	switch {
	case nach != "" && vor != "":
		return nach + ", " + vor
	case nach != "":
		return nach
	default:
		return vor
	}
}

// MusterSchuelerEtikett ist der Inhalt des Probedrucks. Er geht durch dieselbe Zeichenfunktion
// wie der echte Bogen; ein eigener Weg für das Muster wäre eine zweite Wahrheit, an der Bild und
// Druck auseinanderlaufen.
var MusterSchuelerEtikett = SchuelerEtikett{
	BarcodeID: "DEMO-S-001",
	Vorname:   "Max",
	Nachname:  "Mustermann",
	Klasse:    "8G2",
}

// ErzeugeSchuelerEtiketten erzeugt den Klebebogen. startPosition zählt ab 1 und überspringt die
// schon abgezogenen Etiketten eines angebrochenen Bogens, wie bei den Etiketten der Bücher.
func ErzeugeSchuelerEtiketten(formatID string, startPosition int, etiketten []SchuelerEtikett) (*gofpdf.Fpdf, error) {
	format, _ := FindeEtikettFormat(formatID)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(format.RandLinks, format.RandOben, format.RandLinks)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()
	tr := pdfzeichen.Uebersetzer(pdf.UnicodeTranslatorFromDescriptor(""))

	zeichneRaster(pdf, format, startPosition, len(etiketten), func(i int, pos etikettPosition) {
		zeichneSchuelerEtikett(pdf, tr, format, etiketten[i], pos)
	})

	return pdf, nil
}

// zeichneSchuelerEtikett setzt ein Etikett: Name, Klasse, Barcode, Nummer.
//
// Name und Klasse stehen links am Rand, der Barcode mittig darunter, wie der Entwurf
// abgenommen wurde. Die Maße hängen an der Höhe des Etiketts: Auf dem kleinen Format
// (Standard 52, 21,2 mm) ist für eine eigene Zeile der Klasse kein Platz, dort rückt sie hinter
// den Namen.
func zeichneSchuelerEtikett(pdf *gofpdf.Fpdf, tr func(string) string, format EtikettFormat, e SchuelerEtikett, pos etikettPosition) {
	gross := format.Hoehe >= 30
	linkerRand := pos.X + 3
	textbreite := format.Breite - 6

	// tr trägt die Zeichenersetzung schon (pdfzeichen); ohne sie käme „Ayşe" als „Ay.e" aus
	// dem Drucker.
	druck := tr
	kuerze := func(text string, breite float64) string {
		return pdfzeichen.KuerzeAufBreite(pdf, druck, text, breite)
	}

	klasse := strings.TrimSpace(e.Klasse)
	y := pos.Y + 3.0

	if gross {
		pdf.SetFont("Arial", "B", 10)
		pdf.SetXY(linkerRand, y)
		pdf.CellFormat(textbreite, 4.5, druck(kuerze(e.name(), textbreite)), "", 0, "L", false, 0, "")
		y += 5.0

		// Ohne Klasse (von Hand angelegt, Abgänger) fällt die Zeile weg, statt „Klasse " ins
		// Leere zu schreiben.
		if klasse != "" {
			pdf.SetFont("Arial", "", 8)
			pdf.SetXY(linkerRand, y)
			pdf.CellFormat(textbreite, 3.5, druck(kuerze("Klasse "+klasse, textbreite)), "", 0, "L", false, 0, "")
		}
		y += 4.5
	} else {
		// Kleines Format: keine eigene Klassenzeile, die Klasse rückt hinter den Namen.
		// Gekürzt wird der Name: Die Klasse ist kurz, und eine halbe Klasse ist wertlos.
		pdf.SetFont("Arial", "B", 8)
		zeile := e.name()
		if klasse != "" {
			anhang := "  " + klasse
			zeile = kuerze(e.name(), textbreite-pdf.GetStringWidth(druck(anhang))) + anhang
		}
		pdf.SetXY(linkerRand, y)
		pdf.CellFormat(textbreite, 3.5, druck(kuerze(zeile, textbreite)), "", 0, "L", false, 0, "")
		y += 4.0
	}

	bcBreite, bcHoehe := 40.0, 10.0
	if !gross {
		bcBreite, bcHoehe = 33.0, 6.5
	}
	if bcBreite > format.Breite-6 {
		bcBreite = format.Breite - 6
	}

	// Scheitert der Barcode, bleibt der Bogen: Der Rest des Etiketts (Name, Klasse, ablesbare
	// Nummer) ist auch ohne Bild brauchbar.
	if bild, err := strichcode.PNG(e.BarcodeID, false, 250, 70); err == nil {
		optionen := gofpdf.ImageOptions{ImageType: "PNG"}
		bildname := fmt.Sprintf("schueler_%s", e.BarcodeID)
		pdf.RegisterImageOptionsReader(bildname, optionen, bytes.NewReader(bild))
		pdf.ImageOptions(bildname, pos.X+(format.Breite-bcBreite)/2, y, bcBreite, bcHoehe, false, optionen, 0, "")
	}
	y += bcHoehe + 1.0

	// Die Nummer unter dem Barcode ist dieselbe, die er trägt; sie steht dort, damit sie ohne
	// Scanner ablesbar ist. Courier, weil sich O und 0 unterscheiden müssen.
	pdf.SetFont("Courier", "B", 9)
	if !gross {
		pdf.SetFont("Courier", "B", 7)
	}
	pdf.SetXY(pos.X, y)
	pdf.CellFormat(format.Breite, 3.5, druck(e.BarcodeID), "", 0, "C", false, 0, "")
}
