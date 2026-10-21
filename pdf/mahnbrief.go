package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"bibliothek/pkg/pdfzeichen"
	"bibliothek/pkg/schulzeit"
	"bibliothek/pkg/strichcode"

	"github.com/phpdave11/gofpdf"
)

// MahnbriefVorlage ist, was auf allen Briefen eines Drucks gleich steht. Betreff und Text
// kennen die Platzhalter {{.Vorname}}, {{.Nachname}} und {{.Frist}}, der Text dazu
// {{.BuchListe}}.
type MahnbriefVorlage struct {
	Betreff  string
	Text     string
	Absender string
}

// MahnbriefBuch ist ein Buch auf dem Mahnbrief.
type MahnbriefBuch struct {
	Titel            string
	Barcode          string
	AusgeliehenAm    time.Time
	Frist            time.Time
	TageUeberfaellig int
}

// MahnbriefEmpfaenger ist ein Schüler mit seinen Büchern über der Frist und der Anschrift für
// das Fensterkuvert.
type MahnbriefEmpfaenger struct {
	Vorname    string
	Nachname   string
	Strasse    string
	Hausnummer string
	PLZ        string
	Ort        string
	Buecher    []MahnbriefBuch
}

// Maße des Mahnbriefs in Millimetern. Folgeseiten beginnen mahnbriefRand unter der Kante, und
// unter mahnbriefSeitenende bricht gofpdf von sich aus um.
const (
	mahnbriefRand        = 20.0
	mahnbriefSeitenende  = 297.0 - mahnbriefRand
	mahnbriefKopfHoehe   = 7.0
	mahnbriefZeilenHoehe = 15.0
	mahnbriefSpalteTitel = 75.0
)

// mahnbriefAnschrift baut das Fensterfeld. Fehlt die Anschrift, steht das im Feld: Eine leere
// Zeile sähe aus wie ein Druckfehler, so ist zu sehen, welcher Brief über das Kind oder die
// Klassenleitung geht.
func mahnbriefAnschrift(e MahnbriefEmpfaenger) []string {
	name := fmt.Sprintf("Eltern von %s %s", e.Vorname, e.Nachname)
	strasse := strings.TrimSpace(e.Strasse + " " + e.Hausnummer)
	ort := strings.TrimSpace(e.PLZ + " " + e.Ort)
	if strasse == "" && ort == "" {
		strasse = "(keine Adresse hinterlegt)"
	}
	return []string{name, strasse, ort}
}

// zeichneMahnbriefKoepfe setzt die Köpfe der Tabelle und stellt die Schrift der Zeilen ein.
func zeichneMahnbriefKoepfe(pdf *gofpdf.Fpdf, tr func(string) string) {
	pdf.SetFont("Arial", "B", 10)
	pdf.SetX(20)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(mahnbriefSpalteTitel, mahnbriefKopfHoehe, tr("Titel"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, mahnbriefKopfHoehe, tr("Barcode"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, mahnbriefKopfHoehe, tr("Ausgeliehen"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(30, mahnbriefKopfHoehe, tr("Tage überfällig"), "1", 1, "R", true, 0, "")
	pdf.SetFont("Arial", "", 10)
}

// zeichneMahnbriefBuecher setzt die Tabelle der Bücher über der Frist. Der Barcode steht als
// Bild und als Nummer da, damit das Buch bei der Rückgabe vom Brief gescannt werden kann.
func zeichneMahnbriefBuecher(pdf *gofpdf.Fpdf, tr func(string) string, buecher []MahnbriefBuch) {
	// Die Köpfe stehen nicht allein am Fuß einer Seite.
	if len(buecher) > 0 && pdf.GetY()+mahnbriefKopfHoehe+mahnbriefZeilenHoehe > mahnbriefSeitenende {
		pdf.AddPage()
	}
	zeichneMahnbriefKoepfe(pdf, tr)

	const zeilenHoehe = mahnbriefZeilenHoehe
	for _, b := range buecher {
		// Eine Zeile setzt Strichcode und Nummer an feste Stellen; ein Umbruch mitten in ihr
		// verteilte sie über drei Seiten. Passt sie nicht mehr auf die Seite, beginnt sie auf
		// der nächsten, unter den Köpfen.
		if pdf.GetY()+zeilenHoehe > mahnbriefSeitenende {
			pdf.AddPage()
			zeichneMahnbriefKoepfe(pdf, tr)
		}
		obenY := pdf.GetY()
		pdf.SetX(20)
		pdf.CellFormat(mahnbriefSpalteTitel, zeilenHoehe, tr(pdfzeichen.KuerzeAufZelle(pdf, tr, b.Titel, mahnbriefSpalteTitel)), "1", 0, "L", false, 0, "")

		strichX := pdf.GetX()
		pdf.CellFormat(35, zeilenHoehe, "", "1", 0, "", false, 0, "")
		if b.Barcode != "" {
			if png, err := strichcode.PNG(b.Barcode, false, 300, 80); err == nil {
				bildName := "bc_eltern_" + b.Barcode
				optionen := gofpdf.ImageOptions{ImageType: "PNG"}
				pdf.RegisterImageOptionsReader(bildName, optionen, bytes.NewReader(png))
				pdf.ImageOptions(bildName, strichX+2.5, obenY+2, 30, 8, false, optionen, 0, "")
			}
			pdf.SetFont("Courier", "", 7)
			pdf.SetXY(strichX, obenY+10.5)
			pdf.CellFormat(35, 4, tr(b.Barcode), "", 0, "C", false, 0, "")
			pdf.SetFont("Arial", "", 10)
		}

		pdf.SetXY(strichX+35, obenY)
		pdf.CellFormat(30, zeilenHoehe, b.AusgeliehenAm.In(schulzeit.Zone()).Format(datumsformat), "1", 0, "L", false, 0, "")
		pdf.CellFormat(30, zeilenHoehe, fmt.Sprintf("%d", b.TageUeberfaellig), "1", 1, "R", false, 0, "")
	}
}

// platzhalterBuchListe steht in der Vorlage für die Tabelle der gemahnten Bücher.
const platzhalterBuchListe = "{{.BuchListe}}"

// zeichneMahnbrief setzt den Brief eines Schülers nach DIN 5008 (Form A, Fensterkuvert) auf die
// eben begonnene Seite: Fensterfeld, Betreff, Text und die Tabelle seiner Bücher über der Frist.
func zeichneMahnbrief(pdf *gofpdf.Fpdf, tr func(string) string, e MahnbriefEmpfaenger, v MahnbriefVorlage) {
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
	pdf.Cell(40, 5, "Datum: "+schulzeit.Jetzt().Format(datumsformat))

	// {{.Frist}} ist die älteste Rückgabefrist der gemahnten Bücher; je Buch steht die
	// eigene Überschreitung in der Tabelle.
	aeltesteFrist := schulzeit.Jetzt()
	for _, b := range e.Buecher {
		if b.Frist.Before(aeltesteFrist) {
			aeltesteFrist = b.Frist
		}
	}
	ersetzer := strings.NewReplacer(
		"{{.Vorname}}", e.Vorname,
		"{{.Nachname}}", e.Nachname,
		"{{.Frist}}", aeltesteFrist.In(schulzeit.Zone()).Format(datumsformat),
	)

	// In der Betreffzeile hat die Tabelle keinen Platz; ein dort eingetragenes
	// {{.BuchListe}} fällt weg, statt wörtlich im Brief zu stehen.
	pdf.SetFont("Arial", "B", 12)
	pdf.SetXY(20, 100)
	pdf.Cell(0, 5, tr(strings.TrimSpace(strings.ReplaceAll(ersetzer.Replace(v.Betreff), platzhalterBuchListe, ""))))

	// Die Tabelle ersetzt das erste {{.BuchListe}}; der Text danach wird gedruckt, weitere
	// Vorkommen fallen weg.
	teile := strings.SplitN(ersetzer.Replace(v.Text), platzhalterBuchListe, 2)
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

// ErzeugeMahnbriefe setzt je Schüler einen Brief in ein PDF. Ein Brief, der nicht auf eine
// Seite passt, läuft auf Folgeseiten weiter.
func ErzeugeMahnbriefe(briefe []MahnbriefEmpfaenger, v MahnbriefVorlage) ([]byte, error) {
	dokument := gofpdf.New("P", "mm", "A4", "")
	dokument.SetTopMargin(mahnbriefRand)
	dokument.SetAutoPageBreak(true, mahnbriefRand)
	tr := pdfzeichen.Uebersetzer(dokument.UnicodeTranslatorFromDescriptor(""))

	// Ein Druck trägt viele Briefe hintereinander: Jede Folgeseite nennt über dem Rand, zu
	// wessen Brief sie gehört, ob die Tabelle sie füllt oder der Text.
	folgeseiteFuer := ""
	dokument.SetHeaderFunc(func() {
		if folgeseiteFuer == "" {
			return
		}
		dokument.SetFont("Arial", "B", 9)
		dokument.SetXY(20, 12)
		dokument.Cell(0, 5, tr(folgeseiteFuer))
		dokument.SetY(mahnbriefRand)
	})
	for _, e := range briefe {
		folgeseiteFuer = ""
		dokument.AddPage()
		folgeseiteFuer = "Fortsetzung: " + strings.TrimSpace(e.Vorname+" "+e.Nachname)
		zeichneMahnbrief(dokument, tr, e, v)
	}
	var buf bytes.Buffer
	if err := dokument.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
