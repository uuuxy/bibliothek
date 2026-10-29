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

// Raster des Auszugs: A4 mit 20 mm Rand, die Tabelle in vier Spalten aus 24 Teilen. Rückgabe
// braucht für ein Datum mit „überfällig" fett 34,5 mm, Barcode für eine 13-stellige Nummer aus
// Littera 25,5 mm und Luft zum Datum daneben (kontoauszug_test.go).
const (
	kontoauszugRand              = 20
	kontoauszugRaster            = 24
	kontoauszugSpalteTitel       = 10
	kontoauszugSpalteBarcode     = 5
	kontoauszugSpalteAusgeliehen = 4
	kontoauszugSpalteRueckgabe   = 5
)

// ErzeugeKontoauszug erzeugt einen Kontoauszug ohne Unterschriftszeile, den Auszug zum Lesen
// aus der Leserakte.
func ErzeugeKontoauszug(schueler KontoauszugSchueler, buecher []KontoauszugBuch, schule SchuleInfo) ([]byte, error) {
	return ErzeugeKontoauszuege([]KontoauszugEintrag{{Schueler: schueler, Buecher: buecher}}, schule, false)
}

// ErzeugeKontoauszuege erzeugt je Schüler eine Seite. mitUnterschrift hängt eine Zeile für
// Freigabe und Unterschrift an; so dient derselbe Auszug beim Schulabgang als Laufzettel, und
// wer die Unterschrift nicht braucht, lässt die Zeile leer.
func ErzeugeKontoauszuege(eintraege []KontoauszugEintrag, schule SchuleInfo, mitUnterschrift bool) ([]byte, error) {
	konfig := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(kontoauszugRand).
		WithTopMargin(kontoauszugRand).
		WithRightMargin(kontoauszugRand).
		WithMaxGridSize(kontoauszugRaster).
		Build()

	m := maroto.New(konfig)

	// Ein Zeitpunkt für das ganze Dokument: Der Stand oben und das „überfällig" in der Liste
	// sagen dasselbe.
	jetzt := schulzeit.Jetzt()
	for _, e := range eintraege {
		p := page.New()
		p.Add(kontoauszugSeite(e.Schueler, e.Buecher, schule, jetzt, mitUnterschrift)...)
		m.AddPages(p)
	}

	dokument, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return dokument.GetBytes(), nil
}

// kontoauszugSeite baut die Zeilen einer Seite des Auszugs.
func kontoauszugSeite(schueler KontoauszugSchueler, buecher []KontoauszugBuch, schule SchuleInfo, jetzt time.Time, mitUnterschrift bool) []core.Row {
	// Der Kopf der Schule wie bei Bestandsbuch und Bestellbericht (bestandsbuchKopf): Name fett,
	// darunter die Anschrift klein und grau.
	zeilen := []core.Row{
		row.New(6).Add(
			col.New().Add(text.New(schule.Name, props.Text{Size: 12, Style: fontstyle.Bold})),
		),
		row.New(10).Add(
			col.New().Add(text.New(schule.Absenderzeile(), props.Text{
				Size:  8,
				Color: &props.Color{Red: 100, Green: 100, Blue: 100},
			})),
		),
		row.New(15).Add(
			col.New().Add(text.New(
				fmt.Sprintf("Bibliotheks-Kontoauszug für %s %s (Klasse %s)", schueler.Vorname, schueler.Nachname, schueler.Klasse),
				props.Text{Size: 14, Style: fontstyle.Bold, Align: align.Center},
			)),
		),
		row.New(10).Add(
			col.New().Add(text.New(
				fmt.Sprintf("Stand: %s", jetzt.In(schulzeit.Zone()).Format(datumsformat)),
				props.Text{Size: 10, Align: align.Right},
			)),
		),
		row.New(10).Add(col.New()), // Abstand
		row.New(10).Add(
			col.New(kontoauszugSpalteTitel).Add(text.New("Titel", props.Text{Size: 10, Style: fontstyle.Bold})),
			col.New(kontoauszugSpalteBarcode).Add(text.New("Barcode", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Center})),
			col.New(kontoauszugSpalteAusgeliehen).Add(text.New("Ausgeliehen", props.Text{Size: 10, Style: fontstyle.Bold})),
			col.New(kontoauszugSpalteRueckgabe).Add(text.New("Rückgabe", props.Text{Size: 10, Style: fontstyle.Bold})),
		),
	}

	if len(buecher) == 0 {
		zeilen = append(zeilen, row.New(10).Add(
			col.New().Add(text.New("-- Keine offenen Ausleihen --", props.Text{Size: 10, Style: fontstyle.Italic, Align: align.Center})),
		))
	}

	// Nur die lesbare Nummer, kein Strichcode: Der Auszug ist zum Lesen da, nicht zum Scannen
	// an der Theke, und ein Bild je Zeile füllte das Blatt ohne Nutzen.
	ueberfaellig := 0
	for _, buch := range buecher {
		// Überfällig wie in der Leserakte (utils/ueberfaellig.js), ohne deren Ausnahme für die
		// Dauerleihe: Die hat nur, wer kein Schüler ist. Wort und fette Schrift statt Farbe, weil
		// die Schule schwarz-weiß druckt.
		rueckgabe := buch.Rueckgabedatum.In(schulzeit.Zone()).Format(datumsformat)
		rueckgabeText := props.Text{Size: 10}
		if buch.Rueckgabedatum.Before(jetzt) {
			ueberfaellig++
			rueckgabe += " überfällig"
			rueckgabeText.Style = fontstyle.Bold
		}
		zeilen = append(zeilen,
			row.New(8).Add(
				col.New(kontoauszugSpalteTitel).Add(text.New(buch.Titel, props.Text{Size: 10})),
				col.New(kontoauszugSpalteBarcode).Add(text.New(buch.Barcode, props.Text{Size: 10, Align: align.Center})),
				col.New(kontoauszugSpalteAusgeliehen).Add(text.New(buch.Ausleihdatum.In(schulzeit.Zone()).Format(datumsformat), props.Text{Size: 10})),
				col.New(kontoauszugSpalteRueckgabe).Add(text.New(rueckgabe, rueckgabeText)),
			),
		)
	}

	zeilen = append(zeilen, row.New(15).Add(col.New())) // Abstand

	if mitUnterschrift {
		zeilen = append(zeilen,
			row.New(15).Add(col.New()),
			row.New(8).Add(col.New().Add(
				text.New("_________________________________________________________", props.Text{Size: 10}),
			)),
			row.New(8).Add(col.New().Add(
				text.New("Datum, Unterschrift Bibliotheksteam (Freigabe zum Schulabgang)", props.Text{Size: 9, Style: fontstyle.Italic}),
			)),
		)
	} else {
		zeilen = append(zeilen, row.New(10).Add(col.New().Add(
			text.New(kontoauszugHinweis(ueberfaellig), props.Text{Size: 10, Style: fontstyle.Italic, Align: align.Center}),
		)))
	}

	return zeilen
}

// kontoauszugHinweis ist der Satz unter der Liste. Ist etwas überfällig, bittet er um die
// Rückgabe, statt vor Mahnungen zu warnen, die schon fällig sind.
func kontoauszugHinweis(ueberfaellig int) string {
	switch ueberfaellig {
	case 0:
		return "Bitte beachte die Rückgabefristen, um Mahnungen zu vermeiden."
	case 1:
		return "Ein Buch ist überfällig. Bitte gib es so bald wie möglich zurück."
	default:
		return fmt.Sprintf("%d Bücher sind überfällig. Bitte gib sie so bald wie möglich zurück.", ueberfaellig)
	}
}
