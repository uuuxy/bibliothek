package api

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

// Schüler-Etiketten (Betreiber-Entscheidung 24.08.2026).
//
// Ersetzt den A4-Bogen des Ausweis-Designers. Acht Plastikkarten-Abbilder auf einem
// A4-Blatt zum Ausschneiden waren als Notbehelf gedacht und im Alltag nutzlos; ein
// Klebeetikett mit Name, Klasse und Barcode ist das, was tatsächlich gebraucht wird.
//
// Bewusst dieselben Bogenformate und dasselbe Raster wie die Buch-Etiketten
// (zeichneRaster in label_pdf.go): Es ist dieselbe Sorte Klebebogen aus demselben
// Schrank, und eine eigene Rastermathematik hätte sich früher oder später um eine
// halbe Zeile von der anderen entfernt.

// SchuelerEtikett ist eine Zeile des Bogens. Die Felder kommen aus der Datenbank, nicht
// aus dem Request — der Aufrufer schickt nur IDs. Sonst könnte jeder mit view_students
// beliebige Namen auf einen Bogen mit echten Barcodes drucken.
type SchuelerEtikett struct {
	BarcodeID string
	Vorname   string
	Nachname  string
	Klasse    string
}

// cp1252Ersatz bildet Buchstaben ab, die der PDF-Zeichensatz nicht kennt.
//
// gofpdf zeichnet über UnicodeTranslatorFromDescriptor("") in cp1252. Alles darüber
// hinaus wird zu einem Punkt: „Ayşe" kam als „Ay.e" aus dem Drucker. Ein Etikett mit
// entstelltem Namen ist schlimmer als eines ohne Häkchen unter dem s — deshalb wird
// vorher ersetzt statt hinterher verstümmelt.
//
// Abgedeckt sind die Buchstaben, die an einer hessischen Schule tatsächlich vorkommen:
// türkisch, polnisch, rumänisch, tschechisch/kroatisch. Was cp1252 kennt (ä, ö, ü, ß,
// é, à, ç, ñ …), steht hier bewusst NICHT — das druckt richtig.
var cp1252Ersatz = strings.NewReplacer(
	"ş", "s", "Ş", "S", "ğ", "g", "Ğ", "G", "ı", "i", "İ", "I",
	"ć", "c", "Ć", "C", "č", "c", "Č", "C", "ł", "l", "Ł", "L",
	"ń", "n", "Ń", "N", "ś", "s", "Ś", "S", "ż", "z", "Ż", "Z", "ź", "z", "Ź", "Z",
	"ą", "a", "Ą", "A", "ę", "e", "Ę", "E", "ő", "o", "Ő", "O", "ű", "u", "Ű", "U",
	"ș", "s", "Ș", "S", "ț", "t", "Ț", "T", "ř", "r", "Ř", "R", "ě", "e", "Ě", "E",
	"š", "s", "Š", "S", "ž", "z", "Ž", "Z", "đ", "d", "Đ", "D",
)

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

// MusterSchuelerEtikett ist der Inhalt des Testdrucks. Dieselbe Zeichenfunktion wie der
// echte Bogen — ein eigener Muster-Renderer wäre genau die Sorte zweiter Wahrheit, an der
// im Ausweis-Designer schon einmal Leinwand und Druck auseinandergelaufen sind.
var MusterSchuelerEtikett = SchuelerEtikett{
	BarcodeID: "DEMO-S-001",
	Vorname:   "Max",
	Nachname:  "Mustermann",
	Klasse:    "8G2",
}

// GenerateSchuelerEtikettenPDF erzeugt den Klebebogen.
//
// startPosition ist 1-basiert und überspringt bereits abgezogene Etiketten eines
// angebrochenen Bogens — dieselbe Bedeutung wie bei den Buch-Etiketten.
func GenerateSchuelerEtikettenPDF(formatID string, startPosition int, etiketten []SchuelerEtikett) (*gofpdf.Fpdf, error) {
	format, _ := GetLabelFormat(formatID)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(format.MarginLeft, format.MarginTop, format.MarginLeft)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	zeichneRaster(pdf, format, startPosition, len(etiketten), func(i int, pos labelPos) {
		zeichneSchuelerEtikett(pdf, tr, format, etiketten[i], pos)
	})

	return pdf, nil
}

// zeichneSchuelerEtikett setzt EIN Etikett: Name, Klasse, Barcode, Nummer.
//
// Name und Klasse stehen links am Rand, der Barcode mittig darunter — so, wie der
// Entwurf abgenommen wurde. Die Maße hängen an der Etikettenhöhe: Auf dem kleinen
// Format (Standard 52, 21,2 mm) ist für eine eigene Klassenzeile kein Platz, dort
// rückt die Klasse hinter den Namen.
func zeichneSchuelerEtikett(pdf *gofpdf.Fpdf, tr func(string) string, format LabelFormat, e SchuelerEtikett, pos labelPos) {
	gross := format.LabelHeight >= 30
	linkerRand := pos.X + 3
	textbreite := format.LabelWidth - 6

	// EIN Ort für die Zeichenersetzung: Jede Zeichenkette, die auf das Papier geht,
	// läuft hier durch. Ein zweiter Aufrufpfad, der sie vergisst, druckt wieder Punkte.
	druck := func(text string) string { return tr(cp1252Ersatz.Replace(text)) }

	klasse := strings.TrimSpace(e.Klasse)
	y := pos.Y + 3.0

	if gross {
		pdf.SetFont("Arial", "B", 10)
		pdf.SetXY(linkerRand, y)
		pdf.CellFormat(textbreite, 4.5, druck(kuerzeAufZeichen(e.name(), 26)), "", 0, "L", false, 0, "")
		y += 5.0

		// Leere Klasse lässt die Zeile weg, statt "Klasse " ins Nichts zu schreiben.
		// Vorkommen: Handanlage ohne Klassenangabe, Abgänger.
		if klasse != "" {
			pdf.SetFont("Arial", "", 8)
			pdf.SetXY(linkerRand, y)
			pdf.CellFormat(textbreite, 3.5, druck("Klasse "+kuerzeAufZeichen(klasse, 14)), "", 0, "L", false, 0, "")
		}
		y += 4.5
	} else {
		zeile := kuerzeAufZeichen(e.name(), 22)
		if klasse != "" {
			zeile = kuerzeAufZeichen(e.name(), 16) + "  " + kuerzeAufZeichen(klasse, 6)
		}
		pdf.SetFont("Arial", "B", 8)
		pdf.SetXY(linkerRand, y)
		pdf.CellFormat(textbreite, 3.5, druck(zeile), "", 0, "L", false, 0, "")
		y += 4.0
	}

	bcBreite, bcHoehe := 40.0, 10.0
	if !gross {
		bcBreite, bcHoehe = 33.0, 6.5
	}
	if bcBreite > format.LabelWidth-6 {
		bcBreite = format.LabelWidth - 6
	}

	// Ein fehlgeschlagener Barcode darf den ganzen Bogen nicht kippen: Der Rest des
	// Etiketts (Name, Klasse, ablesbare Nummer) ist auch ohne Bild brauchbar.
	if bild, err := GenerateBarcodePNG(e.BarcodeID, false, 250, 70); err == nil {
		opt := gofpdf.ImageOptions{ImageType: "PNG"}
		bildname := fmt.Sprintf("schueler_%s", e.BarcodeID)
		pdf.RegisterImageOptionsReader(bildname, opt, bytes.NewReader(bild))
		pdf.ImageOptions(bildname, pos.X+(format.LabelWidth-bcBreite)/2, y, bcBreite, bcHoehe, false, opt, 0, "")
	}
	y += bcHoehe + 1.0

	// Die Nummer unter dem Barcode ist DIESELBE, die der Barcode trägt — sie steht dort,
	// damit sie ohne Scanner ablesbar ist. Courier, weil O und 0 sich unterscheiden müssen.
	pdf.SetFont("Courier", "B", 9)
	if !gross {
		pdf.SetFont("Courier", "B", 7)
	}
	pdf.SetXY(pos.X, y)
	pdf.CellFormat(format.LabelWidth, 3.5, druck(e.BarcodeID), "", 0, "C", false, 0, "")
}
