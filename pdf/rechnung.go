package pdf

import (
	"fmt"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
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
}

// GenerateRechnung creates a DIN 5008 compliant invoice PDF.
func GenerateRechnung(schueler Schueler, items []RechnungItem) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(25). // Left margin 25mm for DIN 5008 A/B
		WithTopMargin(20).
		WithRightMargin(20).
		Build()

	m := maroto.New(cfg)

	// DIN 5008 Address Window (Sender + Receiver)
	// Absender (Sender line above address)
	m.AddRow(15,
		col.New(12).Add(
			text.New("Schulbibliothek Musterstadt, Musterweg 1, 12345 Musterstadt", props.Text{
				Size:  8,
				Style: fontstyle.Bold,
				Align: align.Left,
			}),
		),
	)

	// Address lines
	addressLines := []string{
		fmt.Sprintf("%s %s", schueler.Vorname, schueler.Nachname),
		fmt.Sprintf("%s %s", schueler.Strasse, schueler.Hausnummer),
		fmt.Sprintf("%s %s", schueler.PLZ, schueler.Ort),
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

	// Date aligned right
	m.AddRow(10,
		col.New(12).Add(
			text.New(fmt.Sprintf("Datum: %s", time.Now().Format("02.01.2006")), props.Text{
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

	// Table Header
	m.AddRow(10,
		col.New(5).Add(text.New("Titel", props.Text{Size: 10, Style: fontstyle.Bold})),
		col.New(3).Add(text.New("Barcode", props.Text{Size: 10, Style: fontstyle.Bold})),
		col.New(2).Add(text.New("Ausgeliehen", props.Text{Size: 10, Style: fontstyle.Bold})),
		col.New(2).Add(text.New("Preis", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Right})),
	)

	// Table Rows
	var total float64
	for _, item := range items {
		m.AddRow(10,
			col.New(5).Add(text.New(item.Titel, props.Text{Size: 10})),
			col.New(3).Add(text.New(item.Barcode, props.Text{Size: 10})),
			col.New(2).Add(text.New(item.Ausleihdatum.Format("02.01.2006"), props.Text{Size: 10})),
			col.New(2).Add(text.New(fmt.Sprintf("%.2f EUR", item.Ersatzpreis), props.Text{Size: 10, Align: align.Right})),
		)
		total += item.Ersatzpreis
	}

	// Total Row
	m.AddRow(15,
		col.New(10).Add(text.New("Summe:", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Right})),
		col.New(2).Add(text.New(fmt.Sprintf("%.2f EUR", total), props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Right})),
	)

	// Footer with Bank details
	m.AddRow(40, col.New(12)) // push footer down somewhat
	m.AddRow(10,
		col.New(12).Add(
			text.New("Bankverbindung: Schulbibliothek Musterstadt | IBAN: DE12 3456 7890 1234 5678 90 | BIC: MUSTERDE", props.Text{
				Size:  8,
				Align: align.Center,
			}),
		),
	)

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}

	return doc.GetBytes(), nil
}
