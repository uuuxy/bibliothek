package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/phpdave11/gofpdf"

	"bibliothek/pkg/pdfzeichen"
)

// Der Ausdruck des Abgangsbuchs — das Blatt, das die Schule zum Stichtag abheftet.
//
// Zwei Abschnitte, einer je Topf, jeder mit eigener Stückzahl: Die Finanzen von Land und
// Schulträger werden getrennt geführt (docs/mittel_konzept.md 1.1/1.2), und wer die Summen
// aus einer gemischten Liste von Hand zieht, zieht sie jedes Halbjahr neu und anders.
// Dieselbe Regel und dieselben Überschriften wie im Bestellbericht.

// Eine Zeile fasst die Exemplare eines Titels zusammen, die am selben Tag aus demselben Grund
// gingen, mit Anzahl und allen Barcodes (internal/bestandsbuch).
const (
	abgangSpalteDatum    = 22.0
	abgangSpalteTitel    = 56.0
	abgangSpalteSignatur = 20.0
	abgangSpalteGrund    = 28.0
	abgangSpalteAnzahl   = 14.0
	abgangSpalteBarcodes = 40.0
	abgangZeilenHoehe    = 6.0
	abgangUmbruchAbY     = 250.0
)

// AbgangsZeile ist eine Zeile des Blatts: die Exemplare eines Titels, die an einem Tag aus
// einem Grund gingen.
type AbgangsZeile struct {
	Datum     time.Time
	Titel     string
	Signatur  string
	GrundText string
	Anzahl    int
	// Barcodes ist der fertige Text der Tür, fortlaufende als Bereich.
	Barcodes string
}

// AbgangsAbschnitt ist ein Topf des Blatts mit seiner Überschrift und der Zahl seiner
// Exemplare.
type AbgangsAbschnitt struct {
	Titel     string
	Exemplare int
	Zeilen    []AbgangsZeile
}

// Abgangsbuch ist die Eingabe des Blatts. Gliederung, Überschriften und Zahlen kommen von der
// Tür; sie sind dieselben wie in der Liste auf dem Bildschirm.
type Abgangsbuch struct {
	Von        time.Time
	Bis        time.Time
	Abschnitte []AbgangsAbschnitt
	// Gesamt ist die Zahl der Abgänge im Zeitraum.
	Gesamt int
	// OhneZeitpunkt und AusKatalogGeloescht zählen, was in keinem Abschnitt steht; das Blatt
	// nennt beide Zahlen unter der Liste.
	OhneZeitpunkt       int
	AusKatalogGeloescht int
}

// ErzeugeAbgangsbuch setzt das Abgangsbuch eines Zeitraums als Blatt zum Abheften.
func ErzeugeAbgangsbuch(buch Abgangsbuch, schule SchuleInfo) ([]byte, error) {
	p := gofpdf.New("P", "mm", "A4", "")
	p.SetMargins(20, 20, 20)
	p.SetAutoPageBreak(true, 20)
	tr := pdfzeichen.Uebersetzer(p.UnicodeTranslatorFromDescriptor(""))
	p.AddPage()

	bestandsbuchKopf(p, tr, "Abgangsbuch", buch.Von, buch.Bis, schule)

	gezeigt := 0
	for _, abschnitt := range buch.Abschnitte {
		gezeigt += abgangsbuchAbschnitt(p, tr, abschnitt)
	}
	if gezeigt == 0 {
		p.SetFont("Arial", "I", 10)
		p.SetTextColor(120, 120, 120)
		p.Cell(0, 8, tr("In diesem Zeitraum ist kein Exemplar aus dem Bestand gegangen."))
		p.SetTextColor(0, 0, 0)
		p.Ln(10)
	}

	abgangsbuchFuss(p, tr, buch)

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// abgangsbuchAbschnitt zeichnet einen Topf und liefert die Zahl seiner Zeilen.
//
// Leere Abschnitte überspringt das Blatt: Eine Überschrift ohne Inhalt hilft niemandem, der
// den Nachweis abheftet. Auf dem Bildschirm steht der leere Topf als Feld mit einer Null,
// siehe abschnitteAus in api/katalog/bestandsbuch.go.
func abgangsbuchAbschnitt(p *gofpdf.Fpdf, tr func(string) string, abschnitt AbgangsAbschnitt) int {
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

	koepfe := func() { abgangsbuchSpaltenkoepfe(p, tr) }
	koepfe()
	for _, z := range zeilen {
		// Ein überlanger Titel schöbe die Spalten der Zeile nach rechts. Gekürzt wird mit
		// Auslassungszeichen: Auf einem Nachweis, den jemand unterschreibt, muss ein gekürzter
		// Titel als gekürzt erkennbar sein. Die Barcodes laufen um, keiner fehlt.
		p.SetFont("Arial", "", 9)
		zeichneBuchzeile(p, tr, []buchzelle{
			{breite: abgangSpalteDatum, text: z.Datum.Format(datumsformat)},
			{breite: abgangSpalteTitel, text: pdfzeichen.KuerzeAufZelle(p, tr, z.Titel, abgangSpalteTitel)},
			{breite: abgangSpalteSignatur, text: pdfzeichen.KuerzeAufZelle(p, tr, z.Signatur, abgangSpalteSignatur)},
			{breite: abgangSpalteGrund, text: z.GrundText},
			{breite: abgangSpalteAnzahl, text: fmt.Sprint(z.Anzahl)},
			{breite: abgangSpalteBarcodes, text: z.Barcodes, umbrechen: true},
		}, koepfe)
	}

	p.SetFont("Arial", "B", 9)
	p.SetFillColor(225, 232, 245)
	p.CellFormat(180, 7, tr(fmt.Sprintf("Summe %s: %s  ",
		abschnitt.Titel, exemplarText(abschnitt.Exemplare))), "1", 1, "R", true, 0, "")
	p.SetFillColor(255, 255, 255)
	p.Ln(6)
	return len(zeilen)
}

func abgangsbuchSpaltenkoepfe(p *gofpdf.Fpdf, tr func(string) string) {
	p.SetFont("Arial", "B", 9)
	p.CellFormat(abgangSpalteDatum, abgangZeilenHoehe, tr("Abgang"), "LTB", 0, "L", false, 0, "")
	p.CellFormat(abgangSpalteTitel, abgangZeilenHoehe, tr("Titel"), "TB", 0, "L", false, 0, "")
	p.CellFormat(abgangSpalteSignatur, abgangZeilenHoehe, tr("Signatur"), "TB", 0, "L", false, 0, "")
	p.CellFormat(abgangSpalteGrund, abgangZeilenHoehe, tr("Grund"), "TB", 0, "L", false, 0, "")
	p.CellFormat(abgangSpalteAnzahl, abgangZeilenHoehe, tr("Anzahl"), "TB", 0, "L", false, 0, "")
	p.CellFormat(abgangSpalteBarcodes, abgangZeilenHoehe, tr("Barcodes"), "TBR", 1, "L", false, 0, "")
}

// abgangsbuchFuss nennt die Gesamtzahl und darunter, was nicht in der Liste steht.
//
// Die beiden Hinweise sind die wichtigeren Zeilen: Was vor Migration 128 ausgesondert wurde,
// trägt kein Datum und steht in keiner Halbjahresliste; was körperlich gelöscht wurde, steht in
// gar keiner Abfrage mehr. Ein Nachweis, der das verschweigt, behauptet eine Vollständigkeit, die
// er nicht hat, und erfundene Zeilen wären schlimmer.
func abgangsbuchFuss(p *gofpdf.Fpdf, tr func(string) string, buch Abgangsbuch) {
	if p.GetY() > abgangUmbruchAbY {
		p.AddPage()
	}
	p.SetFont("Arial", "B", 11)
	p.SetFillColor(220, 230, 255)
	p.CellFormat(180, 9, tr(fmt.Sprintf("Abgänge im Zeitraum: %s  ", exemplarText(buch.Gesamt))), "1", 1, "R", true, 0, "")
	p.SetFillColor(255, 255, 255)

	if buch.OhneZeitpunkt > 0 {
		p.Ln(4)
		p.SetFont("Arial", "I", 9)
		p.SetTextColor(90, 90, 90)
		p.MultiCell(180, 5, tr(hinweisOhneZeitpunkt(buch.OhneZeitpunkt)), "", "L", false)
		p.SetTextColor(0, 0, 0)
	}

	if buch.AusKatalogGeloescht > 0 {
		p.Ln(4)
		p.SetFont("Arial", "I", 9)
		p.SetTextColor(90, 90, 90)
		p.MultiCell(180, 5, tr(hinweisAusKatalogGeloescht(buch.AusKatalogGeloescht)), "", "L", false)
		p.SetTextColor(0, 0, 0)
	}
}

// hinweisOhneZeitpunkt nennt die Exemplare, die ohne Abgangsdatum ausgesondert sind. Der Satz
// hat für ein einzelnes Exemplar eine eigene Form.
func hinweisOhneZeitpunkt(n int) string {
	if n == 1 {
		return "Hinweis: 1 weiteres Exemplar ist ausgesondert, ohne dass ein Abgangsdatum bekannt ist. " +
			"Es wurde vor der Einführung des Abgangsbuchs ausgebucht und kann keinem Zeitraum " +
			"zugeordnet werden; es steht deshalb in keiner Halbjahresliste."
	}
	return fmt.Sprintf(
		"Hinweis: %d weitere Exemplare sind ausgesondert, ohne dass ein Abgangsdatum bekannt ist. "+
			"Sie wurden vor der Einführung des Abgangsbuchs ausgebucht und können keinem Zeitraum "+
			"zugeordnet werden; sie stehen deshalb in keiner Halbjahresliste.", n)
}

// hinweisAusKatalogGeloescht nennt die Exemplare, die im Zeitraum aus dem Katalog gelöscht statt
// ausgesondert wurden, in der Form für eines oder für mehrere.
func hinweisAusKatalogGeloescht(n int) string {
	if n == 1 {
		return "Hinweis: 1 Exemplar wurde in diesem Zeitraum aus dem Katalog gelöscht, statt " +
			"ausgesondert zu werden — mit seinem Titel oder als endgültig entfernter Verlust. " +
			"Titel, Signatur und Abgangsgrund sind mit ihm gelöscht worden; es steht " +
			"deshalb in keiner Liste oben. Seine Nummer ist im Protokoll nachschlagbar."
	}
	return fmt.Sprintf(
		"Hinweis: %d Exemplare wurden in diesem Zeitraum aus dem Katalog gelöscht, statt "+
			"ausgesondert zu werden — mit ihrem Titel oder als endgültig entfernter Verlust. "+
			"Titel, Signatur und Abgangsgrund sind mit ihnen gelöscht worden; sie stehen "+
			"deshalb in keiner Liste oben. Ihre Nummern sind im Protokoll nachschlagbar.", n)
}
