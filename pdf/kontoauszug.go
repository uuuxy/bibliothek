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

// KontoauszugSchueler represents a student in the account statement.
type KontoauszugSchueler struct {
	Vorname  string
	Nachname string
	Klasse   string
}

// KontoauszugBuch represents a book in the account statement.
type KontoauszugBuch struct {
	Titel          string
	Barcode        string
	Ausleihdatum   time.Time
	Rueckgabedatum time.Time
}

// GenerateKontoauszug creates a simple PDF showing a student's currently borrowed books.
func GenerateKontoauszug(schueler KontoauszugSchueler, buecher []KontoauszugBuch) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(20).
		WithTopMargin(20).
		WithRightMargin(20).
		Build()

	m := maroto.New(cfg)

	// Heading
	m.AddRow(15,
		col.New(12).Add(
			text.New(fmt.Sprintf("Bibliotheks-Kontoauszug für %s %s (Klasse %s)", schueler.Vorname, schueler.Nachname, schueler.Klasse), props.Text{
				Size:  14,
				Style: fontstyle.Bold,
				Align: align.Center,
			}),
		),
	)

	// Date
	m.AddRow(10,
		col.New(12).Add(
			text.New(fmt.Sprintf("Stand: %s", time.Now().Format("02.01.2006")), props.Text{
				Size:  10,
				Align: align.Right,
			}),
		),
	)

	m.AddRow(10, col.New(12)) // Spacer

	// Table Header
	m.AddRow(10,
		col.New(5).Add(text.New("Titel", props.Text{Size: 10, Style: fontstyle.Bold})),
		col.New(3).Add(text.New("Barcode", props.Text{Size: 10, Style: fontstyle.Bold})),
		col.New(2).Add(text.New("Ausgeliehen", props.Text{Size: 10, Style: fontstyle.Bold})),
		col.New(2).Add(text.New("Rückgabe", props.Text{Size: 10, Style: fontstyle.Bold})),
	)

	// Table Rows
	for _, buch := range buecher {
		m.AddRow(10,
			col.New(5).Add(text.New(buch.Titel, props.Text{Size: 10})),
			col.New(3).Add(text.New(buch.Barcode, props.Text{Size: 10})),
			col.New(2).Add(text.New(buch.Ausleihdatum.Format("02.01.2006"), props.Text{Size: 10})),
			col.New(2).Add(text.New(buch.Rueckgabedatum.Format("02.01.2006"), props.Text{Size: 10})),
		)
	}

	m.AddRow(20, col.New(12)) // Spacer

	// Footer
	m.AddRow(10,
		col.New(12).Add(
			text.New("Bitte beachte die Rückgabefristen, um Mahnungen zu vermeiden.", props.Text{
				Size:  10,
				Style: fontstyle.Italic,
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
