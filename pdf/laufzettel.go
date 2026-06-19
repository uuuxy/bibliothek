package pdf

import (
	"fmt"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/page"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// LaufzettelAusleihe represents a loan in the clearance certificate.
type LaufzettelAusleihe struct {
	Titel     string
	BarcodeID string
	Frist     string
}

// LaufzettelStudent represents a student in the clearance certificate.
type LaufzettelStudent struct {
	Vorname   string
	Nachname  string
	Klasse    string
	Ausleihen []LaufzettelAusleihe
}

// GenerateLaufzettel creates a PDF with a page per student for the library clearance (Entlassungslaufzettel).
func GenerateLaufzettel(schuelerListe []LaufzettelStudent) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(20).
		WithTopMargin(20).
		WithRightMargin(20).
		Build()

	m := maroto.New(cfg)

	for _, schueler := range schuelerListe {
		p := page.New()

		// Title / Header
		p.Add(row.New(20).Add(
			col.New(12).Add(
				text.New("Entlassungslaufzettel - Schulbibliothek", props.Text{
					Size:  16,
					Style: fontstyle.Bold,
					Align: align.Center,
				}),
			),
		))

		p.Add(row.New(15).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("Schüler/in: %s %s (Klasse %s)", schueler.Vorname, schueler.Nachname, schueler.Klasse), props.Text{
					Size:  12,
					Style: fontstyle.Bold,
					Align: align.Left,
				}),
			),
		))

		// Introductory text
		p.Add(row.New(15).Add(
			col.New(12).Add(text.New("Folgende Medien sind in der Schulbibliothek noch nicht zurückgegeben worden:", props.Text{Size: 10})),
		))

		// Table Header
		p.Add(row.New(10).Add(
			col.New(6).Add(text.New("Titel", props.Text{Size: 10, Style: fontstyle.Bold})),
			col.New(3).Add(text.New("Barcode", props.Text{Size: 10, Style: fontstyle.Bold})),
			col.New(3).Add(text.New("Fällig seit", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Right})),
		))

		// Table Rows
		if len(schueler.Ausleihen) == 0 {
			p.Add(row.New(10).Add(
				col.New(12).Add(text.New("-- Keine offenen Ausleihen --", props.Text{Size: 10, Style: fontstyle.Italic, Align: align.Center})),
			))
		} else {
			for _, buch := range schueler.Ausleihen {
				p.Add(row.New(10).Add(
					col.New(6).Add(text.New(buch.Titel, props.Text{Size: 10})),
					col.New(3).Add(text.New(buch.BarcodeID, props.Text{Size: 10})),
					col.New(3).Add(text.New(buch.Frist, props.Text{Size: 10, Align: align.Right})),
				))
			}
		}

		// Signature block
		p.Add(row.New(40).Add(col.New(12))) // Spacer

		p.Add(row.New(10).Add(
			col.New(12).Add(text.New("_________________________________________________________", props.Text{Size: 10})),
		))
		p.Add(row.New(10).Add(
			col.New(12).Add(text.New("Datum, Unterschrift Bibliotheksteam", props.Text{Size: 10, Style: fontstyle.Italic})),
		))

		m.AddPages(p)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}

	return doc.GetBytes(), nil
}
