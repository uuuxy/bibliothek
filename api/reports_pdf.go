package api

import (
	"bibliothek/pdf"
	"bibliothek/pkg/pdfzeichen"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
	"bytes"
	"context"

	"fmt"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

// mahnbriefVorlage ist, was auf allen Briefen eines Drucks gleich steht.
type mahnbriefVorlage struct {
	Betreff  string
	Text     string
	Absender string
}

// loadMahnungTemplate lädt die Eltern-Mahnvorlage aus der Datenbank; ist keine
// konfiguriert ODER die gespeicherte leer, wird die Standardvorlage verwendet.
// Der Leer-Fall ist real: Der Vorlagen-Editor lässt ” durch (Spalten NOT NULL,
// aber ” erlaubt) — bis zum 01.09.2026 erzeugte das Mahnbriefe ohne Betreff
// und ohne Anschreiben. Dieselbe Prüfung hatte die Bestell-Schwester
// (loadBestellTemplate) von Anfang an.
func (s *Server) loadMahnungTemplate(ctx context.Context) (betreff, textBody string) {
	err := s.DB.Pool.QueryRow(ctx, "SELECT betreff, text_body FROM mail_vorlagen WHERE typ = 'MAHNUNG_ELTERN'").Scan(&betreff, &textBody)
	if err != nil || strings.TrimSpace(betreff) == "" || strings.TrimSpace(textBody) == "" {
		betreff = "Mahnung: Überfällige Bücher"
		textBody = "Sehr geehrte Eltern von {{.Vorname}} {{.Nachname}},\n\nbitte geben Sie folgende Bücher umgehend in die Bibliothek zurück:\n\n{{.BuchListe}}\n\nVielen Dank."
	}
	return betreff, textBody
}

// ladeMahnbriefVorlage liest Betreff und Text aus der Vorlage und die Absenderzeile aus den
// Angaben zur Schule.
func (s *Server) ladeMahnbriefVorlage(ctx context.Context) mahnbriefVorlage {
	betreff, text := s.loadMahnungTemplate(ctx)
	settings, _ := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx) //nolint:errcheck
	schule := pdf.SchuleInfo{
		Name:    settings.SchuleName,
		Strasse: settings.SchuleStrasse,
		PLZ:     settings.SchulePLZ,
		Ort:     settings.SchuleOrt,
	}
	return mahnbriefVorlage{Betreff: betreff, Text: text, Absender: schule.Absenderzeile()}
}

// mahnbriefAnschrift baut das Fensterfeld. Fehlt die Anschrift, steht das im Feld: Eine leere
// Zeile sähe aus wie ein Druckfehler, so ist zu sehen, welcher Brief über das Kind oder die
// Klassenleitung geht.
func mahnbriefAnschrift(e repository.MahnbriefEmpfaenger) []string {
	name := fmt.Sprintf("Eltern von %s %s", e.Vorname, e.Nachname)
	strasse := strings.TrimSpace(e.Strasse + " " + e.Hausnummer)
	ort := strings.TrimSpace(e.PLZ + " " + e.Ort)
	if strasse == "" && ort == "" {
		strasse = "(keine Adresse hinterlegt)"
	}
	return []string{name, strasse, ort}
}

// zeichneMahnbriefBuecher setzt die Tabelle der Bücher über der Frist. Der Barcode steht als
// Bild und als Nummer da, damit das Buch bei der Rückgabe vom Brief gescannt werden kann.
func zeichneMahnbriefBuecher(pdf *gofpdf.Fpdf, tr func(string) string, buecher []repository.MahnbriefBuch) {
	pdf.SetFont("Arial", "B", 10)
	pdf.SetX(20)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(75, 7, tr("Titel"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 7, tr("Barcode"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 7, tr("Ausgeliehen"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(30, 7, tr("Tage überfällig"), "1", 1, "R", true, 0, "")

	pdf.SetFont("Arial", "", 10)
	const rowH = 15.0
	for _, b := range buecher {
		startY := pdf.GetY()
		pdf.SetX(20)
		pdf.CellFormat(75, rowH, tr(kuerzeAufZeichen(b.Titel, 38)), "1", 0, "L", false, 0, "")

		bcX := pdf.GetX()
		pdf.CellFormat(35, rowH, "", "1", 0, "", false, 0, "")
		if b.Barcode != "" {
			if pngBytes, err := GenerateBarcodePNG(b.Barcode, false, 300, 80); err == nil {
				imgName := "bc_eltern_" + b.Barcode
				opt := gofpdf.ImageOptions{ImageType: "PNG"}
				pdf.RegisterImageOptionsReader(imgName, opt, bytes.NewReader(pngBytes))
				pdf.ImageOptions(imgName, bcX+2.5, startY+2, 30, 8, false, opt, 0, "")
			}
			pdf.SetFont("Courier", "", 7)
			pdf.SetXY(bcX, startY+10.5)
			pdf.CellFormat(35, 4, tr(b.Barcode), "", 0, "C", false, 0, "")
			pdf.SetFont("Arial", "", 10)
		}

		pdf.SetXY(bcX+35, startY)
		pdf.CellFormat(30, rowH, b.AusgeliehenAm.In(schulzeit.Zone()).Format(dateFormatDE), "1", 0, "L", false, 0, "")
		pdf.CellFormat(30, rowH, fmt.Sprintf("%d", b.TageUeberfaellig), "1", 1, "R", false, 0, "")
	}
}

// platzhalterBuchListe steht in der Vorlage für die Tabelle der gemahnten Bücher.
const platzhalterBuchListe = "{{.BuchListe}}"

// zeichneMahnbrief setzt eine Seite nach DIN 5008 (Form A, Fensterkuvert) für einen Schüler:
// Fensterfeld, Betreff, Text und die Tabelle seiner Bücher über der Frist.
func zeichneMahnbrief(pdf *gofpdf.Fpdf, tr func(string) string, e repository.MahnbriefEmpfaenger, v mahnbriefVorlage) {
	pdf.AddPage()

	// Falzmarken und Lochmarke.
	pdf.SetLineWidth(0.2)
	pdf.SetDrawColor(150, 150, 150)
	pdf.Line(0, 105, 4, 105)
	pdf.Line(0, 148.5, 6, 148.5)
	pdf.Line(0, 210, 4, 210)
	pdf.SetDrawColor(0, 0, 0)

	// Fensterfeld: Absenderzeile klein darüber, Anschrift ab 52 mm.
	pdf.SetFont("Arial", "U", 7)
	pdf.SetXY(20, 45)
	pdf.Cell(85, 5, tr(v.Absender))
	pdf.SetFont("Arial", "", 11)
	pdf.SetXY(20, 52)
	for _, zeile := range mahnbriefAnschrift(e) {
		pdf.SetX(20)
		pdf.CellFormat(85, 5, tr(zeile), "", 1, "L", false, 0, "")
	}

	pdf.SetXY(150, 85)
	pdf.Cell(40, 5, "Datum: "+schulzeit.Jetzt().Format(dateFormatDE))

	// {{.Frist}} ist die älteste Rückgabefrist der gemahnten Bücher; je Buch steht die
	// eigene Überschreitung in der Tabelle.
	aeltesteFrist := schulzeit.Jetzt()
	for _, b := range e.Buecher {
		if b.Frist.Before(aeltesteFrist) {
			aeltesteFrist = b.Frist
		}
	}
	replacer := strings.NewReplacer(
		"{{.Vorname}}", e.Vorname,
		"{{.Nachname}}", e.Nachname,
		"{{.Frist}}", aeltesteFrist.In(schulzeit.Zone()).Format(dateFormatDE),
	)

	// In der Betreffzeile hat die Tabelle keinen Platz; ein dort eingetragenes
	// {{.BuchListe}} fällt weg, statt wörtlich im Brief zu stehen.
	pdf.SetFont("Arial", "B", 12)
	pdf.SetXY(20, 100)
	pdf.Cell(0, 5, tr(strings.TrimSpace(strings.ReplaceAll(replacer.Replace(v.Betreff), platzhalterBuchListe, ""))))

	// Die Tabelle ersetzt das erste {{.BuchListe}}; der Text danach wird gedruckt, weitere
	// Vorkommen fallen weg.
	teile := strings.SplitN(replacer.Replace(v.Text), platzhalterBuchListe, 2)
	pdf.SetFont("Arial", "", 11)
	pdf.SetXY(20, 115)
	pdf.MultiCell(170, 6, tr(teile[0]), "", "L", false)
	pdf.Ln(5)

	zeichneMahnbriefBuecher(pdf, tr, e.Buecher)

	if len(teile) > 1 {
		pdf.Ln(5)
		pdf.SetX(20)
		pdf.SetFont("Arial", "", 11)
		pdf.MultiCell(170, 6, tr(strings.TrimSpace(strings.ReplaceAll(teile[1], platzhalterBuchListe, ""))), "", "L", false)
	}
}

// erzeugeMahnbriefe setzt je Schüler einen Brief in ein PDF.
func erzeugeMahnbriefe(briefe []repository.MahnbriefEmpfaenger, v mahnbriefVorlage) ([]byte, error) {
	doc := gofpdf.New("P", "mm", "A4", "")
	tr := pdfzeichen.Uebersetzer(doc.UnicodeTranslatorFromDescriptor(""))
	for _, e := range briefe {
		zeichneMahnbrief(doc, tr, e, v)
	}
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
