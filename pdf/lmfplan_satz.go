package pdf

// lmfplan_satz.go — der Satzspiegel des LMF-Plans. Peter, 06.09.2026: „Das PDF muss
// zwingend alles auf einer Seite darstellen." Der Plan hängt im Lehrerzimmer und geht als
// EIN Blatt an die Schulleitung; ein zweites Blatt kostet Termine, weil niemand es sucht.
//
// Feste Zeilenhöhen konnten das nicht halten: ab rund vierzig Terminen brach maroto von
// selbst um. Der Satz rechnet deshalb vorher, was passt, und wählt den größten Schriftgrad,
// mit dem alles auf ein Blatt geht:
//
//	1. eine Spalte, solange die Schrift lesbar bleibt (Faktor ≥ 0,8 → 7,2 pt)
//	2. zwei Spalten nebeneinander, wieder von voller Größe abwärts
//
// Alles steht auf einem Raster aus EINHEITSZEILEN gleicher Höhe: nur so lassen sich zwei
// Spalten Zeile für Zeile nebeneinander setzen. Ein Baustein (Überschrift, Tabellenkopf,
// Termin) belegt so viele Einheitszeilen, wie sein Text nach dem Umbruch braucht — die
// Folgezeilen bleiben leer, der Text fließt sichtbar hinein.
//
// Gemessen wird mit gofpdf, derselben Bibliothek, die maroto unter der Haube benutzt, und
// mit derselben Zeichenumsetzung: eine Schätzung würde bei langen Vermerken danebenliegen,
// und genau die entscheiden über die letzte Zeile.

import (
	"math"
	"strings"

	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/phpdave11/gofpdf"
)

const (
	lmfGrid        = 24 // Rasterspalten INNERHALB einer Satzspalte
	lmfSeiteBreite = 210.0
	lmfSeiteHoehe  = 297.0
	lmfRandOben    = 15.0
	lmfRandUnten   = 10.0
	lmfPunktInMm   = 25.4 / 72.0 // maroto setzt die Zeilenhöhe auf genau die Schriftgröße
	lmfLuft        = 1.15        // Luft zwischen zwei Textzeilen
	lmfSchritt     = 0.02        // Raster der Verkleinerung
	lmfSicherheit  = 0.5         // mm, die wir dem Seitenende nicht abtrotzen
	lmfEngstens    = 0.6         // so weit darf eine Tabellenzeile enger gesetzt werden
	lmfZellenLuft  = 1.5         // mm Haarraum zwischen zwei Zellen einer Zeile
)

// lmfModus ist eine Satzart: wie viele Spalten, wie breit die Ränder, wie das Raster
// innerhalb einer Spalte aufgeteilt ist und ab welcher Verkleinerung wir aufgeben.
type lmfModus struct {
	spalten   int
	rand      float64
	zellen    [5]int  // Wochentag, Datum, Stunde, Klassen, Besonderheiten (Summe: lmfGrid)
	steg      float64 // mm rechts der letzten Spalte — zweispaltig der Abstand der Tabellen
	kurzerTag bool    // zweispaltig ist für „Donnerstag" kein Platz — dann „Do."
	minFaktor float64
}

// lmfModi in der Reihenfolge, in der wir sie probieren: erst die gewohnte eine Spalte,
// solange sie lesbar bleibt (Faktor 0,75 ≈ 6,8 pt), dann zwei Spalten wieder von voller
// Größe abwärts — lieber zwei Spalten in 9 pt als eine in 5 pt.
var lmfModi = []lmfModus{
	{spalten: 1, rand: 20, zellen: [5]int{4, 4, 2, 6, 8}, steg: 1, minFaktor: 0.75},
	{spalten: 2, rand: 12, zellen: [5]int{4, 5, 3, 5, 7}, steg: 4, kurzerTag: true, minFaktor: 0.15},
}

func (mo lmfModus) spaltenBreite() float64 {
	return (lmfSeiteBreite - 2*mo.rand) / float64(mo.spalten)
}

func (mo lmfModus) zellenBreite(k int) float64 {
	return mo.spaltenBreite() * float64(mo.zellen[k]) / lmfGrid
}

// lmfMasse sind alle Maße eines Versuchs, aus einem Faktor gerechnet — Schriftgrade und
// Zeilenhöhe schrumpfen gemeinsam, sonst stünde Text übereinander.
type lmfMasse struct {
	faktor  float64
	einheit float64
	dok     float64
	titel   float64
	unter   float64
	basis   float64
	fuss    float64
}

func lmfMasseFuer(f float64) lmfMasse {
	return lmfMasse{faktor: f, einheit: 5 * f, dok: 11 * f, titel: 13 * f, unter: 8 * f, basis: 9 * f, fuss: 8 * f}
}

func (m lmfMasse) kopfHoehe() float64 { return 10 * m.faktor }
func (m lmfMasse) fussHoehe() float64 { return 6 * m.faktor }

// lmfZelle ist eine Zelle einer Einheitszeile: Rasterbreite, Text, Stil.
type lmfZelle struct {
	breite int
	text   string
	stil   props.Text
}

// lmfArt unterscheidet, was ein Baustein beim Spaltenumbruch bedeutet.
type lmfArt int

const (
	lmfAbstand lmfArt = iota // Leerzeile zwischen zwei Abschnitten — oben in einer Spalte entbehrlich
	lmfKopf                  // Überschrift/Tabellenkopf — gehört mit der ersten Datenzeile zusammen
	lmfDaten                 // ein Termin
)

// lmfBaustein ist ein Stück Inhalt, das ungeteilt in eine Spalte gehört.
type lmfBaustein struct {
	zellen    []lmfZelle
	zeilen    int
	art       lmfArt
	abschnitt int
}

// lmfMesser misst Textbreiten so, wie maroto sie später umbricht: gofpdf mit derselben
// Kernschrift und derselben Umsetzung nach Latin-1.
//
// Bewusst phpdave11/gofpdf und nicht das jung-kurt/gofpdf der übrigen Schriftstücke: Es
// ist die Bibliothek, die maroto selbst benutzt. Weil Go je Modul genau eine Fassung baut,
// misst dieser Messer damit immer mit denselben Schriftmaßen, mit denen maroto setzt —
// auch nach einem Versionssprung.
type lmfMesser struct {
	pdf *gofpdf.Fpdf
	tr  func(string) string
}

func lmfNeuerMesser() *lmfMesser {
	p := gofpdf.New("P", "mm", "A4", "")
	return &lmfMesser{pdf: p, tr: p.UnicodeTranslatorFromDescriptor("")}
}

// groessteWortBreite ist die Breite des breitesten Wortes — ein einzelnes Wort bricht
// maroto NICHT um, es läuft in die Nachbarzelle. Genau das sah man an „Wochentag" über
// einer halb so breiten Spalte.
func (me *lmfMesser) groessteWortBreite(text string, groesse float64, stil fontstyle.Type) float64 {
	me.pdf.SetFont("helvetica", string(stil), groesse)
	breiteste := 0.0
	for _, wort := range strings.Split(me.tr(text), " ") {
		breiteste = math.Max(breiteste, me.pdf.GetStringWidth(wort))
	}
	return breiteste
}

// passendeGroesse verkleinert die Schrift einer Zelle so weit, dass ihr längstes Wort
// hineinpasst — höchstens bis lmfEngstens, darunter wäre die Zeile nicht mehr zu lesen.
func (me *lmfMesser) passendeGroesse(text string, groesse float64, stil fontstyle.Type, breite float64) float64 {
	wort := me.groessteWortBreite(text, groesse, stil)
	if breite <= 0 || wort <= breite {
		return groesse
	}
	return math.Max(groesse*breite/wort, groesse*lmfEngstens)
}

// zeilen liefert die Anzahl der Textzeilen nach dem Umbruch an Leerzeichen — dieselbe
// Regel wie maroto (breakline.EmptySpaceStrategy), die wir jedem Text mitgeben.
func (me *lmfMesser) zeilen(text string, groesse float64, stil fontstyle.Type, breite float64) int {
	if strings.TrimSpace(text) == "" || breite <= 0 {
		return 1
	}
	me.pdf.SetFont("helvetica", string(stil), groesse)
	zeilen, aktuell := 1, 0.0
	for _, wort := range strings.Split(me.tr(text), " ") {
		if wort == "" {
			continue
		}
		stueck := wort
		if aktuell > 0 {
			stueck = " " + wort
		}
		if b := me.pdf.GetStringWidth(stueck); aktuell+b <= breite {
			aktuell += b
			continue
		}
		zeilen++
		aktuell = me.pdf.GetStringWidth(wort)
	}
	return zeilen
}
