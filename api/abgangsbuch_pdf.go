package api

import (
	"bytes"
	"fmt"

	"github.com/jung-kurt/gofpdf"

	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Der Ausdruck des Abgangsbuchs — das Blatt, das die Schule zum Stichtag abheftet.
//
// Zwei Abschnitte, einer je Topf, jeder mit eigener Stückzahl: Die Finanzen von Land und
// Schulträger werden getrennt geführt (docs/mittel_konzept.md 1.1/1.2), und wer die Summen
// aus einer gemischten Liste von Hand zieht, zieht sie jedes Halbjahr neu — und anders.
// Dieselbe Regel und dieselben Überschriften wie im Bestellbericht.

const (
	abgangSpalteDatum    = 24.0
	abgangSpalteNummer   = 30.0
	abgangSpalteTitel    = 78.0
	abgangSpalteSignatur = 20.0
	abgangSpalteGrund    = 28.0
	abgangZeilenHoehe    = 6.0
	abgangUmbruchAbY     = 250.0
)

func generateAbgangsbuchPDF(buch repository.Abgangsbuch, schule pdf.SchuleInfo) ([]byte, error) {
	p := gofpdf.New("P", "mm", "A4", "")
	p.SetMargins(20, 20, 20)
	p.SetAutoPageBreak(true, 20)
	tr := p.UnicodeTranslatorFromDescriptor("")
	p.AddPage()

	abgangsbuchKopf(p, tr, buch, schule)

	gezeigt := 0
	for _, abschnitt := range abschnitteAus(buch.Zeilen, func(z repository.AbgangsZeile) string { return z.Topf }) {
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

func abgangsbuchKopf(p *gofpdf.Fpdf, tr func(string) string, buch repository.Abgangsbuch, schule pdf.SchuleInfo) {
	p.SetFont("Arial", "B", 12)
	p.Cell(0, 8, tr(schule.Name))
	p.Ln(5)
	p.SetFont("Arial", "", 8)
	p.SetTextColor(100, 100, 100)
	p.Cell(0, 4, tr(schule.Absenderzeile()))
	p.SetTextColor(0, 0, 0)
	p.Ln(10)

	p.SetFont("Arial", "", 10)
	p.CellFormat(0, 6, tr(schule.OrtDatum(schulzeit.Jetzt().Format(dateFormatDE))), "", 1, "R", false, 0, "")
	p.Ln(4)

	p.SetFont("Arial", "B", 14)
	p.Cell(0, 10, tr("Abgangsbuch"))
	p.Ln(7)
	p.SetFont("Arial", "", 10)
	p.SetTextColor(80, 80, 80)
	p.Cell(0, 6, tr(fmt.Sprintf("Zeitraum: %s bis %s",
		buch.Von.Format(dateFormatDE), buch.Bis.Format(dateFormatDE))))
	p.SetTextColor(0, 0, 0)
	p.Ln(12)
}

// abgangsbuchAbschnitt zeichnet einen Topf und liefert die Zahl seiner Zeilen.
//
// Leere Abschnitte überspringt das Blatt: Eine Überschrift ohne Inhalt hilft niemandem, der
// den Nachweis abheftet. Auf dem Bildschirm steht sie, siehe bestandsbuch_abschnitte.go.
func abgangsbuchAbschnitt(p *gofpdf.Fpdf, tr func(string) string, abschnitt Abschnitt[repository.AbgangsZeile]) int {
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

	abgangsbuchSpaltenkoepfe(p, tr)
	for _, z := range zeilen {
		if p.GetY() > abgangUmbruchAbY {
			p.AddPage()
			abgangsbuchSpaltenkoepfe(p, tr)
		}
		p.SetFont("Arial", "", 9)
		p.CellFormat(abgangSpalteDatum, abgangZeilenHoehe, tr(z.Datum.Format(dateFormatDE)), "LB", 0, "L", false, 0, "")
		p.CellFormat(abgangSpalteNummer, abgangZeilenHoehe, tr(z.Barcode), "B", 0, "L", false, 0, "")
		p.CellFormat(abgangSpalteTitel, abgangZeilenHoehe, tr(kuerzeMitAuslassung(z.Titel, 46)), "B", 0, "L", false, 0, "")
		p.CellFormat(abgangSpalteSignatur, abgangZeilenHoehe, tr(kuerzeMitAuslassung(z.Signatur, 12)), "B", 0, "L", false, 0, "")
		p.CellFormat(abgangSpalteGrund, abgangZeilenHoehe, tr(z.GrundText), "BR", 1, "L", false, 0, "")
	}

	p.SetFont("Arial", "B", 9)
	p.SetFillColor(225, 232, 245)
	p.CellFormat(180, 7, tr(fmt.Sprintf("Summe %s: %d Exemplare  ",
		abschnitt.Titel, len(zeilen))), "1", 1, "R", true, 0, "")
	p.SetFillColor(255, 255, 255)
	p.Ln(6)
	return len(zeilen)
}

func abgangsbuchSpaltenkoepfe(p *gofpdf.Fpdf, tr func(string) string) {
	p.SetFont("Arial", "B", 9)
	p.CellFormat(abgangSpalteDatum, abgangZeilenHoehe, tr("Abgang"), "LTB", 0, "L", false, 0, "")
	p.CellFormat(abgangSpalteNummer, abgangZeilenHoehe, tr("Nummer"), "TB", 0, "L", false, 0, "")
	p.CellFormat(abgangSpalteTitel, abgangZeilenHoehe, tr("Titel"), "TB", 0, "L", false, 0, "")
	p.CellFormat(abgangSpalteSignatur, abgangZeilenHoehe, tr("Signatur"), "TB", 0, "L", false, 0, "")
	p.CellFormat(abgangSpalteGrund, abgangZeilenHoehe, tr("Grund"), "TBR", 1, "L", false, 0, "")
}

// abgangsbuchFuss nennt die Gesamtzahl — und die Abgänge OHNE bekannten Zeitpunkt.
//
// Die zweite Zeile ist die wichtigere: Was vor Migration 128 ausgesondert wurde, trägt kein
// Datum und steht in KEINER Halbjahresliste. Ein Nachweis, der das verschweigt, behauptet
// Vollständigkeit, die er nicht hat — und ein erfundenes Datum wäre schlimmer gewesen.
func abgangsbuchFuss(p *gofpdf.Fpdf, tr func(string) string, buch repository.Abgangsbuch) {
	if p.GetY() > abgangUmbruchAbY {
		p.AddPage()
	}
	p.SetFont("Arial", "B", 11)
	p.SetFillColor(220, 230, 255)
	p.CellFormat(180, 9, tr(fmt.Sprintf("Abgänge im Zeitraum: %d Exemplare  ", len(buch.Zeilen))), "1", 1, "R", true, 0, "")
	p.SetFillColor(255, 255, 255)

	if buch.OhneZeitpunkt > 0 {
		p.Ln(4)
		p.SetFont("Arial", "I", 9)
		p.SetTextColor(90, 90, 90)
		p.MultiCell(180, 5, tr(fmt.Sprintf(
			"Hinweis: %d weitere Exemplare sind ausgesondert, ohne dass ein Abgangsdatum bekannt ist. "+
				"Sie wurden vor der Einführung des Abgangsbuchs ausgebucht und können keinem Zeitraum "+
				"zugeordnet werden; sie stehen deshalb in keiner Halbjahresliste.", buch.OhneZeitpunkt)),
			"", "L", false)
		p.SetTextColor(0, 0, 0)
	}
}

// kuerzeMitAuslassung schneidet einen Text auf die Spaltenbreite und setzt ein Auslassungs-
// zeichen. Ein überlanger Titel schöbe sonst die Spalten der Zeile nach rechts, und die
// Tabelle stünde krumm.
//
// Nicht `kuerze` aus anliegen.go: Die schneidet hart ab. Auf einem Nachweis, den jemand
// unterschreibt, muss ein gekürzter Titel als gekürzt erkennbar sein — sonst liest ihn
// jemand als vollständigen Titel und findet das Buch nicht wieder.
func kuerzeMitAuslassung(s string, maxZeichen int) string {
	r := []rune(s)
	if len(r) <= maxZeichen {
		return s
	}
	return string(r[:maxZeichen-1]) + "…"
}
