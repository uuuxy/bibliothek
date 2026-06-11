package pdf

import (
	"fmt"
	"time"

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

type MahnungSchueler struct {
	Vorname  string
	Nachname string
	Klasse   string
	Buecher  []MahnungBuch
}

type MahnungBuch struct {
	Titel       string
	Barcode     string
	FaelligSeit time.Time
}

// GenerateMahnliste creates a PDF with a page per student containing their overdue books.
func GenerateMahnliste(schuelerListe []MahnungSchueler) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(20).
		WithTopMargin(20).
		WithRightMargin(20).
		Build()

	m := maroto.New(cfg)

	for _, schueler := range schuelerListe {
		if len(schueler.Buecher) == 0 {
			continue
		}

		p := page.New()

		// Title / Header
		p.Add(row.New(20).Add(
			col.New(12).Add(
				text.New("Erinnerung: Rückgabe von Bibliotheksbüchern", props.Text{
					Size:  14,
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

		// Friendly reminder text
		p.Add(row.New(10).Add(
			col.New(12).Add(text.New("Hallo,", props.Text{Size: 10})),
		))
		p.Add(row.New(15).Add(
			col.New(12).Add(text.New("bitte denke daran, folgende überfällige Medien schnellstmöglich in der Bibliothek abzugeben:", props.Text{Size: 10})),
		))

		// Table Header
		p.Add(row.New(10).Add(
			col.New(6).Add(text.New("Titel", props.Text{Size: 10, Style: fontstyle.Bold})),
			col.New(3).Add(text.New("Barcode", props.Text{Size: 10, Style: fontstyle.Bold})),
			col.New(3).Add(text.New("Fällig seit", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Right})),
		))

		// Table Rows
		for _, buch := range schueler.Buecher {
			p.Add(row.New(10).Add(
				col.New(6).Add(text.New(buch.Titel, props.Text{Size: 10})),
				col.New(3).Add(text.New(buch.Barcode, props.Text{Size: 10})),
				col.New(3).Add(text.New(buch.FaelligSeit.Format("02.01.2006"), props.Text{Size: 10, Align: align.Right})),
			))
		}

		// Footer note
		p.Add(row.New(20).Add(col.New(12))) // Spacer
		p.Add(row.New(10).Add(
			col.New(12).Add(text.New("Vielen Dank, dein Bibliotheksteam", props.Text{Size: 10, Style: fontstyle.Italic})),
		))

		m.AddPages(p)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}

	return doc.GetBytes(), nil
}
