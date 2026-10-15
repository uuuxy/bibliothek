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

	"bibliothek/pkg/betrag"
	"bibliothek/pkg/schulzeit"
)

// Schueler ist der Empfänger der Rechnung mit seiner Anschrift für das Fensterkuvert.
type Schueler struct {
	Vorname    string
	Nachname   string
	Strasse    string
	Hausnummer string
	PLZ        string
	Ort        string
}

// RechnungsPosition ist eine Zeile der Rechnung: eine offene Forderung.
type RechnungsPosition struct {
	Titel        string
	Barcode      string
	Ausleihdatum time.Time
	Ersatzpreis  float64
	// Land sagt, dass die Position ein Buch des Landes betrifft (Eigentum nach
	// repository.ExemplarTopfSQL). Das entscheidet den Zahlungsweg: Bücher des Landes an das
	// Land, alles andere (Bücherei, Geräte) an den Schulträger (zahlungsweg.go). Eine Rechnung
	// kann beides tragen; sie listet alle offenen Forderungen eines Schülers und sucht sie sich
	// nicht aus.
	Land bool
}

// ErzeugeRechnung erzeugt die Rechnung als PDF nach DIN 5008. zahlung nennt das Konto des Landes
// (Zahlstelle und Bankverbindung aus den Einstellungen); bar in der Bibliothek ist für ein
// Lernmittel der Weg, den die Arbeitshilfe ausdrücklich untersagt.
func ErzeugeRechnung(schueler Schueler, positionen []RechnungsPosition, schule SchuleInfo, zahlung Zahlungsangaben) ([]byte, error) {
	konfig := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(25). // linker Rand 25 mm nach DIN 5008
		WithTopMargin(20).
		WithRightMargin(20).
		Build()

	m := maroto.New(konfig)

	baueAnschrift(m, schule, schueler)
	baueKopf(m)
	baueEinleitung(m)
	bauePositionen(m, positionen)
	baueFuss(m, positionen, zahlung)

	dokument, err := m.Generate()
	if err != nil {
		return nil, err
	}

	return dokument.GetBytes(), nil
}

func baueAnschrift(m core.Maroto, schule SchuleInfo, schueler Schueler) {
	// Fensterfeld nach DIN 5008: die Absenderzeile über der Anschrift.
	m.AddRow(15,
		col.New(12).Add(
			text.New(schule.Absenderzeile(), props.Text{
				Size:  8,
				Style: fontstyle.Bold,
				Align: align.Left,
			}),
		),
	)

	// Die Anschrift. Fehlt sie, steht das im Fensterfeld: Zwei leere Zeilen sähen aus wie ein
	// Druckfehler, so ist sofort sichtbar, dass die Rechnung nicht mit der Post gehen kann
	// (dieselbe Regel wie beim Eltern-Mahnbrief, mahnbrief.go).
	zeile2 := strings.TrimSpace(schueler.Strasse + " " + schueler.Hausnummer)
	zeile3 := strings.TrimSpace(schueler.PLZ + " " + schueler.Ort)
	if zeile2 == "" && zeile3 == "" {
		zeile2 = "(keine Adresse hinterlegt)"
	}
	anschriftZeilen := []string{
		fmt.Sprintf("%s %s", schueler.Vorname, schueler.Nachname),
		zeile2,
		zeile3,
	}

	for _, zeile := range anschriftZeilen {
		m.AddRow(5,
			col.New(12).Add(
				text.New(zeile, props.Text{
					Size:  10,
					Align: align.Left,
				}),
			),
		)
	}

	// Abstand vor dem Betreff nach DIN 5008
	m.AddRow(20, col.New(12))
}

func baueKopf(m core.Maroto) {
	// Datum rechtsbündig
	m.AddRow(10,
		col.New(12).Add(
			text.New(fmt.Sprintf("Datum: %s", schulzeit.Jetzt().Format("02.01.2006")), props.Text{
				Size:  10,
				Align: align.Right,
			}),
		),
	)

	// Betreff
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

func baueEinleitung(m core.Maroto) {
	// Einleitung
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

func bauePositionen(m core.Maroto, positionen []RechnungsPosition) {
	// Kopf der Tabelle
	m.AddRow(10,
		col.New(5).Add(text.New("Titel", props.Text{Size: 10, Style: fontstyle.Bold})),
		col.New(3).Add(text.New("Barcode", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Center})),
		col.New(2).Add(text.New("Ausgeliehen", props.Text{Size: 10, Style: fontstyle.Bold})),
		col.New(2).Add(text.New("Preis", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Right})),
	)

	// Zeilen der Tabelle
	var summe float64
	for _, position := range positionen {
		m.AddRows(positionsZeilen(position)...)
		summe += position.Ersatzpreis
	}
	// Geldbeträge stehen in der Datenbank genau als NUMERIC(10,2), in Go als float64. Die Summe
	// wird hier auf Cent gerundet, damit eine Abweichung der Gleitkommazahl nie im Betrag der
	// Rechnung steht.
	summe = math.Round(summe*100) / 100

	// Summenzeile
	m.AddRow(15,
		col.New(10).Add(text.New("Summe:", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Right})),
		col.New(2).Add(text.New(betrag.Text(summe)+" EUR", props.Text{Size: 10, Style: fontstyle.Bold, Align: align.Right})),
	)
}

// baueFuss zeichnet den Zahlungsweg, je Topf einen, damit Geld des Landes und
// Geld des Schulträgers nicht in einer Summe auf einem Konto landen (zahlungsweg.go).
func baueFuss(m core.Maroto, positionen []RechnungsPosition, zahlung Zahlungsangaben) {
	var land, traeger float64
	for _, position := range positionen {
		if position.Land {
			land += position.Ersatzpreis
			continue
		}
		traeger += position.Ersatzpreis
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

// positionsZeilen liefert zwei Zeilen: den Rechnungsposten mit Barcode-Bild und darunter
// die lesbare Barcode-Nummer.
func positionsZeilen(position RechnungsPosition) []core.Row {
	return []core.Row{
		row.New(12).Add(
			col.New(5).Add(text.New(position.Titel, props.Text{Size: 10})),
			code.NewBarCol(3, position.Barcode, props.Barcode{Center: true, Percent: 90}),
			col.New(2).Add(text.New(position.Ausleihdatum.Format("02.01.2006"), props.Text{Size: 10})),
			col.New(2).Add(text.New(betrag.Text(position.Ersatzpreis)+" EUR", props.Text{Size: 10, Align: align.Right})),
		),
		row.New(5).Add(
			col.New(5),
			col.New(3).Add(text.New(position.Barcode, props.Text{Size: 8, Align: align.Center})),
			col.New(4),
		),
	}
}
