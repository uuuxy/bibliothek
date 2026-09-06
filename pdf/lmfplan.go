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
	Titel      string
	Untertitel string // der eine Satz unter der Überschrift (was in diesem Abschnitt geschieht)
	Zeilen     []LmfPlanZeile
}

// lmfSortierhinweis steht über jeder Tabelle — die Zeilen stehen in der Reihenfolge, in
// der die Termine stattfinden, nicht in der des Planers.
const lmfSortierhinweis = "Sortiert nach Zeitpunkt/Termin"

var wochentage = [...]string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"}

// Wochentag liefert den deutschen Namen — für PDF und Tests dieselbe Tabelle.
func Wochentag(t time.Time) string { return wochentage[t.Weekday()] }

// WochentagKurz ist die zweibuchstabige Abkürzung („Do."). Sie steht nur im zweispaltigen
// Satz: dort ist für den ausgeschriebenen Tag kein Platz.
func WochentagKurz(t time.Time) string {
	name := Wochentag(t)
	return string([]rune(name)[:2]) + "."
}

// GenerateLmfPlan baut den Plan in der Form der bisherigen Excel-Tabelle der Schule:
// Kopf „LMF-PLAN", je Abschnitt eine Überschrift und die Tabelle, sortiert nach
// Zeitpunkt. Ein Abschnitt ohne Zeilen wird ausgelassen.
//
// Das Ergebnis ist EIN Blatt — auch bei einem vollen Plan. Wie das erreicht wird (und
// warum feste Zeilenhöhen es nicht konnten), steht in lmfplan_satz.go.
func GenerateLmfPlan(abschnitte []LmfPlanAbschnitt, stand time.Time) ([]byte, error) {
	satz := lmfSatzWaehlen(abschnitte)
	breite := lmfGrid * satz.modus.spalten
	cfg := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(satz.modus.rand).
		WithRightMargin(satz.modus.rand).
		WithTopMargin(lmfRandOben).
		WithBottomMargin(lmfRandUnten).
		WithMaxGridSize(breite).
		Build()
	m := maroto.New(cfg)
	p := page.New()

	p.Add(row.New(satz.masse.kopfHoehe()).Add(col.New(breite).Add(
		text.New("LMF-PLAN", props.Text{Size: satz.masse.dok, Style: fontstyle.Bold, Align: align.Center}))))
	for _, r := range satz.reihen() {
		p.Add(r)
	}
	p.Add(row.New(satz.masse.fussHoehe()).Add(col.New(breite).Add(
		text.New("Stand: "+stand.Format(dateFormatDE), props.Text{Size: satz.masse.fuss, Style: fontstyle.Italic, Align: align.Right}))))

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
