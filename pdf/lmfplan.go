package pdf

import (
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

// LmfPlanZeile ist eine Zeile des Plans, wie ihn das Kollegium kennt: Wochentag,
// Datum, Stunde, Klassen, Besonderheiten.
type LmfPlanZeile struct {
	Datum   time.Time
	Stunde  int
	Klassen string // „6F1/6F2" oder leer
	Vermerk string
}

// LmfPlanAbschnitt ist „BÜCHERRÜCKGABE" oder „BÜCHERAUSGABE" mit seinen Zeilen.
type LmfPlanAbschnitt struct {
	Titel  string
	Zeilen []LmfPlanZeile
}

var wochentage = [...]string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"}

// Wochentag liefert den deutschen Namen — für PDF und Tests dieselbe Tabelle.
func Wochentag(t time.Time) string { return wochentage[t.Weekday()] }

// GenerateLmfPlan baut den Plan in der Form der bisherigen Excel-Tabelle der Schule:
// Kopf „LMF-PLAN", je Abschnitt eine Überschrift und die Tabelle, sortiert nach
// Zeitpunkt. Ein Abschnitt ohne Zeilen wird ausgelassen.
func GenerateLmfPlan(abschnitte []LmfPlanAbschnitt, stand time.Time) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(20).
		WithTopMargin(15).
		WithRightMargin(20).
		Build()
	m := maroto.New(cfg)
	p := page.New()

	p.Add(row.New(10).Add(col.New(12).Add(
		text.New("LMF-PLAN", props.Text{Size: 11, Style: fontstyle.Bold, Align: align.Center}))))

	klein := props.Text{Size: 9}
	kopf := props.Text{Size: 9, Style: fontstyle.Bold}
	for _, a := range abschnitte {
		if len(a.Zeilen) == 0 {
			continue
		}
		p.Add(row.New(8).Add(col.New(12).Add(
			text.New(a.Titel, props.Text{Size: 13, Style: fontstyle.Bold, Align: align.Center}))))
		p.Add(row.New(6).Add(col.New(12).Add(
			text.New("Sortiert nach Zeitpunkt/Termin", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Center}))))
		p.Add(row.New(6).Add(
			col.New(2).Add(text.New("Wochentag", kopf)),
			col.New(2).Add(text.New("Datum", kopf)),
			col.New(1).Add(text.New("Stunde", kopf)),
			col.New(3).Add(text.New("Klassen", kopf)),
			col.New(4).Add(text.New("Besonderheiten", kopf)),
		))
		for _, z := range a.Zeilen {
			p.Add(row.New(5).Add(
				col.New(2).Add(text.New(Wochentag(z.Datum), klein)),
				col.New(2).Add(text.New(z.Datum.Format(dateFormatDE), klein)),
				col.New(1).Add(text.New(stundeText(z.Stunde), klein)),
				col.New(3).Add(text.New(z.Klassen, props.Text{Size: 9, Style: fontstyle.Bold})),
				col.New(4).Add(text.New(z.Vermerk, klein)),
			))
		}
		p.Add(row.New(6).Add(col.New(12)))
	}

	p.Add(row.New(6).Add(col.New(12).Add(
		text.New("Stand: "+stand.Format(dateFormatDE), props.Text{Size: 8, Style: fontstyle.Italic, Align: align.Right}))))

	m.AddPages(p)
	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return doc.GetBytes(), nil
}

func stundeText(stunde int) string {
	return itoa(stunde) + ". Std."
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [4]byte
	i := len(b)
	for n > 0 && i > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
