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
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"bibliothek/pkg/schulzeit"
)

// KontoauszugSchueler ist der Schüler, dessen Konto der Auszug zeigt.
type KontoauszugSchueler struct {
	Vorname  string
	Nachname string
	Klasse   string
}

// KontoauszugBuch ist ein ausgeliehenes Buch auf dem Auszug.
type KontoauszugBuch struct {
	Titel          string
	Barcode        string
	Ausleihdatum   time.Time
	Rueckgabedatum time.Time
}

// KontoauszugEintrag bündelt einen Schüler mit seinen Büchern für den Druck vieler Auszüge,
// etwa den Laufzettel der Abgänger mit Unterschriftszeile.
type KontoauszugEintrag struct {
	Schueler KontoauszugSchueler
	Buecher  []KontoauszugBuch
}

// ErzeugeKontoauszug erzeugt einen Kontoauszug ohne Unterschriftszeile, den Auszug zum Lesen
// aus der Leserakte.
func ErzeugeKontoauszug(schueler KontoauszugSchueler, buecher []KontoauszugBuch) ([]byte, error) {
	return ErzeugeKontoauszuege([]KontoauszugEintrag{{Schueler: schueler, Buecher: buecher}}, false)
}

// ErzeugeKontoauszuege erzeugt je Schüler eine Seite. mitUnterschrift hängt eine Zeile für
// Freigabe und Unterschrift an; so dient derselbe Auszug beim Schulabgang als Laufzettel, und
// wer die Unterschrift nicht braucht, lässt die Zeile leer.
func ErzeugeKontoauszuege(eintraege []KontoauszugEintrag, mitUnterschrift bool) ([]byte, error) {
	konfig := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(20).
		WithTopMargin(20).
		WithRightMargin(20).
		Build()

	m := maroto.New(konfig)

	for _, e := range eintraege {
		p := page.New()
		p.Add(kontoauszugSeite(e.Schueler, e.Buecher, mitUnterschrift)...)
		m.AddPages(p)
	}

	dokument, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return dokument.GetBytes(), nil
}

// kontoauszugSeite baut die Zeilen einer Seite des Auszugs.
func kontoauszugSeite(schueler KontoauszugSchueler, buecher []KontoauszugBuch, mitUnterschrift bool) []core.Row {
	zeilen := []core.Row{
		row.New(15).Add(
			col.New(12).Add(text.New(
				fmt.Sprintf("Bibliotheks-Kontoauszug für %s %s (Klasse %s)", schueler.Vorname, schueler.Nachname, schueler.Klasse),
				props.Text{Size: 14, Style: fontstyle.Bold, Align: align.Center},
			)),
		),
		row.New(10).Add(
			col.New(12).Add(text.New(
				fmt.Sprintf("Stand: %s", schulzeit.Jetzt().Format(datumsformat)),
				props.Text{Size: 10, Align: align.Right},
			)),
		),
		row.New(10).Add(col.New(12)), // Abstand
		row.New(10).Add(
			col.New(5).Add(text.New("Titel", props.Text{Size: 10, Style: fontstyle.Bold})),
			col.New(3).Add(text.New("Barcode", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Center})),
			col.New(2).Add(text.New("Ausgeliehen", props.Text{Size: 10, Style: fontstyle.Bold})),
			col.New(2).Add(text.New("Rückgabe", props.Text{Size: 10, Style: fontstyle.Bold})),
		),
	}

	if len(buecher) == 0 {
		zeilen = append(zeilen, row.New(10).Add(
			col.New(12).Add(text.New("-- Keine offenen Ausleihen --", props.Text{Size: 10, Style: fontstyle.Italic, Align: align.Center})),
		))
	}

	// Nur die lesbare Nummer, kein Strichcode: Der Auszug ist zum Lesen da, nicht zum Scannen
	// an der Theke, und ein Bild je Zeile füllte das Blatt ohne Nutzen.
	for _, buch := range buecher {
		zeilen = append(zeilen,
			row.New(8).Add(
				col.New(5).Add(text.New(buch.Titel, props.Text{Size: 10})),
				col.New(3).Add(text.New(buch.Barcode, props.Text{Size: 10, Align: align.Center})),
				col.New(2).Add(text.New(buch.Ausleihdatum.Format(datumsformat), props.Text{Size: 10})),
				col.New(2).Add(text.New(buch.Rueckgabedatum.Format(datumsformat), props.Text{Size: 10})),
			),
		)
	}

	zeilen = append(zeilen, row.New(15).Add(col.New(12))) // Abstand

	if mitUnterschrift {
		zeilen = append(zeilen,
			row.New(15).Add(col.New(12)),
			row.New(8).Add(col.New(12).Add(
				text.New("_________________________________________________________", props.Text{Size: 10}),
			)),
			row.New(8).Add(col.New(12).Add(
				text.New("Datum, Unterschrift Bibliotheksteam (Freigabe zum Schulabgang)", props.Text{Size: 9, Style: fontstyle.Italic}),
			)),
		)
	} else {
		zeilen = append(zeilen, row.New(10).Add(col.New(12).Add(
			text.New("Bitte beachte die Rückgabefristen, um Mahnungen zu vermeiden.", props.Text{Size: 10, Style: fontstyle.Italic, Align: align.Center}),
		)))
	}

	return zeilen
}
