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

// LMFPlanZeile ist eine Zeile des Plans, wie ihn das Kollegium kennt: Wochentag,
// Datum, Stunde, Klassen, Besonderheiten.
type LMFPlanZeile struct {
	Datum   time.Time
	Stunde  int
	Klassen string // „6F1/6F2" oder leer
	Vermerk string
}

// LMFPlanAbschnitt ist „BÜCHERRÜCKGABE" oder „BÜCHERAUSGABE" mit seinen Zeilen.
type LMFPlanAbschnitt struct {
	Titel      string
	Untertitel string // der eine Satz unter der Überschrift (was in diesem Abschnitt geschieht)
	Zeilen     []LMFPlanZeile
}

// lmfSortierhinweis steht über jeder Tabelle: Die Zeilen stehen in der Reihenfolge, in
// der die Termine stattfinden, nicht in der des Planers.
const lmfSortierhinweis = "Sortiert nach Zeitpunkt/Termin"

var wochentage = [...]string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"}

// wochentag liefert den deutschen Namen; PDF und Tests nehmen dieselbe Tabelle.
func wochentag(t time.Time) string { return wochentage[t.Weekday()] }

// wochentagKurz ist die zweibuchstabige Abkürzung („Do."). Sie steht nur im zweispaltigen
// Satz: dort ist für den ausgeschriebenen Tag kein Platz.
func wochentagKurz(t time.Time) string {
	name := wochentag(t)
	return string([]rune(name)[:2]) + "."
}

// ErzeugeLMFPlan baut den Plan in der Form, die das Kollegium aus der Excel-Tabelle der Schule
// kennt: Kopf „LMF-PLAN", je Abschnitt eine Überschrift und die Tabelle, sortiert nach
// Zeitpunkt. Ein Abschnitt ohne Zeilen wird ausgelassen.
//
// Das Ergebnis ist ein Blatt, auch bei einem vollen Plan. Wie das erreicht wird (und
// warum feste Zeilenhöhen es nicht konnten), steht in lmfplan_satz.go.
func ErzeugeLMFPlan(abschnitte []LMFPlanAbschnitt, stand time.Time) ([]byte, error) {
	satz := lmfSatzWaehlen(abschnitte)
	breite := lmfGrid * satz.modus.spalten
	konfig := config.NewBuilder().
		WithPageSize("A4").
		WithLeftMargin(satz.modus.rand).
		WithRightMargin(satz.modus.rand).
		WithTopMargin(lmfRandOben).
		WithBottomMargin(lmfRandUnten).
		WithMaxGridSize(breite).
		Build()
	m := maroto.New(konfig)
	p := page.New()

	p.Add(row.New(satz.masse.kopfHoehe()).Add(col.New(breite).Add(
		text.New("LMF-PLAN", props.Text{Size: satz.masse.dok, Style: fontstyle.Bold, Align: align.Center}))))
	for _, r := range satz.reihen() {
		p.Add(r)
	}
	p.Add(row.New(satz.masse.fussHoehe()).Add(col.New(breite).Add(
		text.New("Stand: "+stand.Format(datumsformat), props.Text{Size: satz.masse.fuss, Style: fontstyle.Italic, Align: align.Right}))))

	m.AddPages(p)
	dokument, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return dokument.GetBytes(), nil
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
