package pdf

import (
	"fmt"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/code"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/page"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
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

// KontoauszugEintrag bündelt einen Schüler mit seinen Büchern (für den Stapeldruck,
// z. B. der Abgänger-Laufzettel als Kontoauszug mit Unterschriftszeile).
type KontoauszugEintrag struct {
	Schueler KontoauszugSchueler
	Buecher  []KontoauszugBuch
}

// GenerateKontoauszug erzeugt einen einzelnen Kontoauszug OHNE Unterschriftszeile —
// der reine Info-Auszug aus dem Schülerprofil.
func GenerateKontoauszug(schueler KontoauszugSchueler, buecher []KontoauszugBuch) ([]byte, error) {
	return GenerateKontoauszugBatch([]KontoauszugEintrag{{Schueler: schueler, Buecher: buecher}}, false)
}

// GenerateKontoauszugBatch erzeugt je Schüler eine Seite. mitUnterschrift hängt eine
// Freigabe-/Unterschriftszeile an — damit derselbe Kontoauszug beim Schulabgang zugleich
// als Laufzettel dient. Schulen, die es locker halten, ignorieren die Zeile einfach.
func GenerateKontoauszugBatch(eintraege []KontoauszugEintrag, mitUnterschrift bool) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(20).
		WithTopMargin(20).
		WithRightMargin(20).
		Build()

	m := maroto.New(cfg)

	for _, e := range eintraege {
		p := page.New()
		p.Add(kontoauszugSeite(e.Schueler, e.Buecher, mitUnterschrift)...)
		m.AddPages(p)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return doc.GetBytes(), nil
}

// kontoauszugSeite baut die Zeilen einer einzelnen Kontoauszug-Seite.
func kontoauszugSeite(schueler KontoauszugSchueler, buecher []KontoauszugBuch, mitUnterschrift bool) []core.Row {
	rows := []core.Row{
		row.New(15).Add(
			col.New(12).Add(text.New(
				fmt.Sprintf("Bibliotheks-Kontoauszug für %s %s (Klasse %s)", schueler.Vorname, schueler.Nachname, schueler.Klasse),
				props.Text{Size: 14, Style: fontstyle.Bold, Align: align.Center},
			)),
		),
		row.New(10).Add(
			col.New(12).Add(text.New(
				fmt.Sprintf("Stand: %s", time.Now().Format(dateFormatDE)),
				props.Text{Size: 10, Align: align.Right},
			)),
		),
		row.New(10).Add(col.New(12)), // Spacer
		row.New(10).Add(
			col.New(5).Add(text.New("Titel", props.Text{Size: 10, Style: fontstyle.Bold})),
			col.New(3).Add(text.New("Barcode", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Center})),
			col.New(2).Add(text.New("Ausgeliehen", props.Text{Size: 10, Style: fontstyle.Bold})),
			col.New(2).Add(text.New("Rückgabe", props.Text{Size: 10, Style: fontstyle.Bold})),
		),
	}

	if len(buecher) == 0 {
		rows = append(rows, row.New(10).Add(
			col.New(12).Add(text.New("-- Keine offenen Ausleihen --", props.Text{Size: 10, Style: fontstyle.Italic, Align: align.Center})),
		))
	}

	// Tabellenzeilen: Barcode-Bild zum Scannen, darunter die lesbare Nummer.
	for _, buch := range buecher {
		rows = append(rows,
			row.New(12).Add(
				col.New(5).Add(text.New(buch.Titel, props.Text{Size: 10})),
				code.NewBarCol(3, buch.Barcode, props.Barcode{Center: true, Percent: 90}),
				col.New(2).Add(text.New(buch.Ausleihdatum.Format(dateFormatDE), props.Text{Size: 10})),
				col.New(2).Add(text.New(buch.Rueckgabedatum.Format(dateFormatDE), props.Text{Size: 10})),
			),
			row.New(5).Add(
				col.New(5),
				col.New(3).Add(text.New(buch.Barcode, props.Text{Size: 8, Align: align.Center})),
				col.New(4),
			),
		)
	}

	rows = append(rows, row.New(15).Add(col.New(12))) // Spacer

	if mitUnterschrift {
		rows = append(rows,
			row.New(15).Add(col.New(12)),
			row.New(8).Add(col.New(12).Add(
				text.New("_________________________________________________________", props.Text{Size: 10}),
			)),
			row.New(8).Add(col.New(12).Add(
				text.New("Datum, Unterschrift Bibliotheksteam (Freigabe zum Schulabgang)", props.Text{Size: 9, Style: fontstyle.Italic}),
			)),
		)
	} else {
		rows = append(rows, row.New(10).Add(col.New(12).Add(
			text.New("Bitte beachte die Rückgabefristen, um Mahnungen zu vermeiden.", props.Text{Size: 10, Style: fontstyle.Italic, Align: align.Center}),
		)))
	}

	return rows
}
