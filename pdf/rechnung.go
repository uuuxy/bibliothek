package pdf

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/code"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"bibliothek/pkg/schulzeit"
)

// Schueler represents a student on an invoice.
type Schueler struct {
	Vorname    string
	Nachname   string
	Strasse    string
	Hausnummer string
	PLZ        string
	Ort        string
}

// RechnungItem represents a line item on an invoice.
type RechnungItem struct {
	Titel        string
	Barcode      string
	Ausleihdatum time.Time
	Ersatzpreis  float64
	// IstLernmittel entscheidet den Topf und damit den Zahlungsweg dieser Position:
	// Lernmittel gehen an das Land, alles andere (Bücherei, Geräte) an den Schulträger
	// (siehe zahlungsweg.go). Eine Rechnung kann beides tragen — sie listet alle offenen
	// Forderungen eines Schülers und sucht sie sich nicht aus.
	IstLernmittel bool
}

// GenerateRechnung creates a DIN 5008 compliant invoice PDF.
//
// `zahlung` nennt das Konto des Landes (Zahlstelle und Bankverbindung aus den
// Einstellungen). Bis zum 17.09.2026 stand am Fuß „bar in der Bibliothek" — für ein
// Lernmittel ist das der Weg, den die Arbeitshilfe ausdrücklich untersagt.
func GenerateRechnung(schueler Schueler, items []RechnungItem, schule SchuleInfo, zahlung Zahlungsangaben) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(25). // Left margin 25mm for DIN 5008 A/B
		WithTopMargin(20).
		WithRightMargin(20).
		Build()

	m := maroto.New(cfg)

	buildAddressBlock(m, schule, schueler)
	buildHeaderBlock(m)
	buildIntroBlock(m)
	buildItemsTableBlock(m, items)
	buildFooterBlock(m, items, zahlung)

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}

	return doc.GetBytes(), nil
}

func buildAddressBlock(m core.Maroto, schule SchuleInfo, schueler Schueler) {
	// DIN 5008 Address Window (Sender + Receiver)
	// Absender (Sender line above address)
	m.AddRow(15,
		col.New(12).Add(
			text.New(schule.Absenderzeile(), props.Text{
				Size:  8,
				Style: fontstyle.Bold,
				Align: align.Left,
			}),
		),
	)

	// Address lines. Fehlt die Anschrift, steht das AUSDRÜCKLICH im Fensterfeld —
	// zwei leere Zeilen sähen aus wie ein Druckfehler; so ist sofort sichtbar,
	// dass diese Rechnung nicht per Post gehen kann (gleiche Regel wie der
	// Eltern-Mahnbrief, api/reports_pdf.go).
	zeile2 := strings.TrimSpace(schueler.Strasse + " " + schueler.Hausnummer)
	zeile3 := strings.TrimSpace(schueler.PLZ + " " + schueler.Ort)
	if zeile2 == "" && zeile3 == "" {
		zeile2 = "(keine Adresse hinterlegt)"
	}
	addressLines := []string{
		fmt.Sprintf("%s %s", schueler.Vorname, schueler.Nachname),
		zeile2,
		zeile3,
	}

	for _, line := range addressLines {
		m.AddRow(5,
			col.New(12).Add(
				text.New(line, props.Text{
					Size:  10,
					Align: align.Left,
				}),
			),
		)
	}

	// Space before subject (DIN 5008 padding)
	m.AddRow(20, col.New(12))
}

func buildHeaderBlock(m core.Maroto) {
	// Date aligned right
	m.AddRow(10,
		col.New(12).Add(
			text.New(fmt.Sprintf("Datum: %s", schulzeit.Jetzt().Format("02.01.2006")), props.Text{
				Size:  10,
				Align: align.Right,
			}),
		),
	)

	// Subject
	m.AddRow(15,
		col.New(12).Add(
			text.New("Ersatzforderung für verlorene Medien", props.Text{
				Size:  12,
				Style: fontstyle.Bold,
				Align: align.Left,
			}),
		),
	)
}

func buildIntroBlock(m core.Maroto) {
	// Introductory Text
	m.AddRow(10,
		col.New(12).Add(
			text.New("Sehr geehrte Erziehungsberechtigte,", props.Text{Size: 10}),
		),
	)
	m.AddRow(15,
		col.New(12).Add(
			text.New("bitte überweisen Sie die Ersatzforderung für folgende Medien:", props.Text{Size: 10}),
		),
	)
}

func buildItemsTableBlock(m core.Maroto, items []RechnungItem) {
	// Table Header
	m.AddRow(10,
		col.New(5).Add(text.New("Titel", props.Text{Size: 10, Style: fontstyle.Bold})),
		col.New(3).Add(text.New("Barcode", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Center})),
		col.New(2).Add(text.New("Ausgeliehen", props.Text{Size: 10, Style: fontstyle.Bold})),
		col.New(2).Add(text.New("Preis", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Right})),
	)

	// Table Rows
	var total float64
	for _, item := range items {
		m.AddRows(generateItemRow(item)...)
		total += item.Ersatzpreis
	}
	// Geldbeträge liegen in der DB exakt als NUMERIC(10,2); in Go werden sie als
	// float64 geführt. Bei der einzigen Akkumulation hier die Summe explizit auf
	// Cent runden, damit theoretische Float-Drift nie in den Rechnungsbetrag leckt.
	total = math.Round(total*100) / 100

	// Total Row
	m.AddRow(15,
		col.New(10).Add(text.New("Summe:", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Right})),
		col.New(2).Add(text.New(fmt.Sprintf("%.2f EUR", total), props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Right})),
	)
}

// buildFooterBlock zeichnet den Zahlungsweg — je Topf einen, damit Geld des Landes und
// Geld des Schulträgers nicht in einer Summe auf einem Konto landen (zahlungsweg.go).
func buildFooterBlock(m core.Maroto, items []RechnungItem, zahlung Zahlungsangaben) {
	var land, traeger float64
	for _, item := range items {
		if item.IstLernmittel {
			land += item.Ersatzpreis
			continue
		}
		traeger += item.Ersatzpreis
	}
	land = math.Round(land*100) / 100
	traeger = math.Round(traeger*100) / 100

	m.AddRow(20, col.New(12))
	for _, block := range ZahlungswegBloecke(land, traeger, zahlung) {
		if block.Ueberschrift != "" {
			m.AddRow(6, col.New(12).Add(
				text.New(block.Ueberschrift, props.Text{Size: 9, Style: fontstyle.Bold}),
			))
		}
		for _, zeile := range block.Zeilen {
			m.AddRow(5, col.New(12).Add(text.New(zeile, props.Text{Size: 9})))
		}
		m.AddRow(6, col.New(12))
	}
}

// generateItemRow liefert zwei Zeilen: den Rechnungsposten mit Barcode-Bild und darunter
// die lesbare Barcode-Nummer.
func generateItemRow(item RechnungItem) []core.Row {
	return []core.Row{
		row.New(12).Add(
			col.New(5).Add(text.New(item.Titel, props.Text{Size: 10})),
			code.NewBarCol(3, item.Barcode, props.Barcode{Center: true, Percent: 90}),
			col.New(2).Add(text.New(item.Ausleihdatum.Format("02.01.2006"), props.Text{Size: 10})),
			col.New(2).Add(text.New(fmt.Sprintf("%.2f EUR", item.Ersatzpreis), props.Text{Size: 10, Align: align.Right})),
		),
		row.New(5).Add(
			col.New(5),
			col.New(3).Add(text.New(item.Barcode, props.Text{Size: 8, Align: align.Center})),
			col.New(4),
		),
	}
}
