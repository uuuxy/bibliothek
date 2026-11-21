package pdf

import (
	"fmt"
	"strings"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/page"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"bibliothek/internal/pkg/schulzeit"
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
	// Dauerleihe hat keine Frist und wird nie überfällig.
	Dauerleihe bool
	// IstGeraet heißt: Die Zeile ist ein verliehenes Gerät; Titel trägt den Modellnamen.
	IstGeraet bool
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
	bauer, err := marotoKonfig()
	if err != nil {
		return nil, err
	}
	konfig := bauer.
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
			col.New().Add(marotoText(schule.Name, props.Text{Size: 12, Style: fontstyle.Bold})),
		),
		row.New(10).Add(
			col.New().Add(marotoText(schule.Absenderzeile(), props.Text{
				Size:  8,
				Color: &props.Color{Red: 100, Green: 100, Blue: 100},
			})),
		),
		row.New(15).Add(
			col.New().Add(marotoText(
				kontoauszugUeberschrift(schueler),
				props.Text{Size: 14, Style: fontstyle.Bold, Align: align.Center},
			)),
		),
		row.New(10).Add(
			col.New().Add(marotoText(
				fmt.Sprintf("Stand: %s", jetzt.In(schulzeit.Zone()).Format(datumsformat)),
				props.Text{Size: 10, Align: align.Right},
			)),
		),
		row.New(10).Add(col.New()), // Abstand
		row.New(10).Add(
			col.New(kontoauszugSpalteTitel).Add(marotoText("Titel", props.Text{Size: 10, Style: fontstyle.Bold})),
			col.New(kontoauszugSpalteBarcode).Add(marotoText("Barcode", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Center})),
			col.New(kontoauszugSpalteAusgeliehen).Add(marotoText("Ausgeliehen", props.Text{Size: 10, Style: fontstyle.Bold})),
			col.New(kontoauszugSpalteRueckgabe).Add(marotoText("Rückgabe", props.Text{Size: 10, Style: fontstyle.Bold})),
		),
	}

	if len(buecher) == 0 {
		zeilen = append(zeilen, row.New(10).Add(
			col.New().Add(marotoText("-- Keine offenen Ausleihen --", props.Text{Size: 10, Style: fontstyle.Italic, Align: align.Center})),
		))
	}

	// Nur die lesbare Nummer, kein Strichcode: Der Auszug ist zum Lesen da, nicht zum Scannen
	// an der Theke, und ein Bild je Zeile füllte das Blatt ohne Nutzen.
	ueberfaelligeBuecher, ueberfaelligeGeraete, mitFrist := 0, 0, 0
	for _, buch := range buecher {
		// Überfällig wie in der Leserakte (utils/ueberfaellig.js): Eine Dauerleihe hat keine Frist.
		// Wort und fette Schrift statt Farbe, weil die Schule schwarz-weiß druckt.
		rueckgabe := buch.Rueckgabedatum.In(schulzeit.Zone()).Format(datumsformat)
		rueckgabeText := props.Text{Size: 10}
		if buch.Dauerleihe {
			rueckgabe = "ohne Frist"
		} else {
			mitFrist++
		}
		if !buch.Dauerleihe && buch.Rueckgabedatum.Before(jetzt) {
			if buch.IstGeraet {
				ueberfaelligeGeraete++
			} else {
				ueberfaelligeBuecher++
			}
			rueckgabe += " überfällig"
			rueckgabeText.Style = fontstyle.Bold
		}
		zeilen = append(zeilen,
			row.New(8).Add(
				col.New(kontoauszugSpalteTitel).Add(marotoText(buch.Titel, props.Text{Size: 10})),
				col.New(kontoauszugSpalteBarcode).Add(marotoText(buch.Barcode, props.Text{Size: 10, Align: align.Center})),
				col.New(kontoauszugSpalteAusgeliehen).Add(marotoText(buch.Ausleihdatum.In(schulzeit.Zone()).Format(datumsformat), props.Text{Size: 10})),
				col.New(kontoauszugSpalteRueckgabe).Add(marotoText(rueckgabe, rueckgabeText)),
			),
		)
	}

	zeilen = append(zeilen, row.New(15).Add(col.New())) // Abstand

	if mitUnterschrift {
		zeilen = append(zeilen,
			row.New(15).Add(col.New()),
			row.New(8).Add(col.New().Add(
				marotoText("_________________________________________________________", props.Text{Size: 10}),
			)),
			row.New(8).Add(col.New().Add(
				marotoText("Freigabe zum Schulabgang: Datum und Unterschrift des Bibliotheksteams", props.Text{Size: 9, Style: fontstyle.Italic}),
			)),
		)
	} else if mitFrist > 0 {
		// Der Satz spricht von Fristen; Dauerleihen haben keine.
		zeilen = append(zeilen, row.New(10).Add(col.New().Add(
			marotoText(kontoauszugHinweis(ueberfaelligeBuecher, ueberfaelligeGeraete), props.Text{Size: 10, Style: fontstyle.Italic, Align: align.Center}),
		)))
	}

	return zeilen
}

// kontoauszugUeberschrift nennt den Leser und, wenn er eine hat, die Klasse.
func kontoauszugUeberschrift(leser KontoauszugSchueler) string {
	if leser.Klasse == "" {
		return fmt.Sprintf("Bibliotheks-Kontoauszug für %s %s", leser.Vorname, leser.Nachname)
	}
	return fmt.Sprintf("Bibliotheks-Kontoauszug für %s %s, Klasse %s", leser.Vorname, leser.Nachname, leser.Klasse)
}

// kontoauszugHinweis ist der Satz unter der Liste. Ist etwas überfällig, bittet er um die
// Rückgabe, statt vor Mahnungen zu warnen, die schon fällig sind. Bücher und Geräte nennt er
// getrennt: Ein Tablet ist kein Buch, und wer nur Bücher hat, liest denselben Satz wie bisher.
func kontoauszugHinweis(buecher, geraete int) string {
	switch {
	case buecher == 0 && geraete == 0:
		return "Bitte beachte die Rückgabefristen, um Mahnungen zu vermeiden."
	case buecher == 1 && geraete == 0:
		return "Ein Buch ist überfällig. Bitte gib es so bald wie möglich zurück."
	case buecher == 0 && geraete == 1:
		return "Ein Gerät ist überfällig. Bitte gib es so bald wie möglich zurück."
	}
	var teile []string
	if buecher > 0 {
		teile = append(teile, anzahlText(buecher, "Buch", "Bücher"))
	}
	if geraete > 0 {
		teile = append(teile, anzahlText(geraete, "Gerät", "Geräte"))
	}
	return strings.Join(teile, " und ") + " sind überfällig. Bitte gib sie so bald wie möglich zurück."
}
