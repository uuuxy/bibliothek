package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"

	"github.com/jung-kurt/gofpdf"
)

// Die Abschnitte eines Bestandsnachweises — EIN Ort für Reihenfolge und Überschriften.
//
// Zugangs- und Abgangsbuch zeigen ihre Zeilen getrennt nach Topf: Die Finanzen von Land und
// Schulträger werden getrennt geführt (docs/mittel_konzept.md 1.1/1.2). Beide Bücher gibt es
// zweimal — auf dem Bildschirm und als Blatt.
//
// Genau daran ist das Abgangsbuch beim ersten Wurf auseinandergelaufen: Das Blatt schrieb
// „Lernmittelfreiheit (Land)" (mittelBeschriftung), die Oberfläche „Lernmittel (Land)" — zwei
// Wörter für denselben Topf auf demselben Nachweis. Seit dem 17.09.2026 baut deshalb der
// SERVER die Abschnitte samt Überschrift, und der Ausdruck und der Bildschirm zeigen dieselbe
// Liste.

// Abschnitt ist ein Topf-Block eines Bestandsnachweises.
type Abschnitt[T any] struct {
	// Topf ist der gespeicherte Wert ('land', 'schultraeger', '' = ohne Zuordnung).
	Topf string `json:"topf"`
	// Titel ist die Überschrift, wie ein Mensch sie liest.
	Titel  string `json:"titel"`
	Zeilen []T    `json:"zeilen"`
}

// abschnitteAus gruppiert Zeilen nach Topf — in der Reihenfolge, die in diesem Programm
// überall gilt (Lernmittel zuerst, dann Schülerbücherei, zuletzt ohne Zuordnung).
//
// Die beiden echten Töpfe stehen IMMER da, auch leer: Auf dem Bildschirm sagt „kein Zugang in
// diesem Zeitraum" etwas, ein fehlender Abschnitt sähe nach einem Filterfehler aus. Der
// Ausdruck überspringt die leeren — ein Blatt zum Abheften soll keine Überschriften ohne
// Inhalt tragen.
//
// „Ohne Zuordnung" erscheint nur, wenn es solche Zeilen gibt. Beim Abgangsbuch kann es sie
// nicht geben (jeder Titel ist entweder Lernmittel oder nicht), beim Zugangsbuch schon: Ein
// Exemplar, das ohne Bestellung entstanden ist, trägt keinen Beleg darüber, aus welchem Geld
// es bezahlt wurde — und geraten wird das nicht.
func abschnitteAus[T any](zeilen []T, topfVon func(T) string) []Abschnitt[T] {
	nach := map[string][]T{}
	for _, z := range zeilen {
		topf := topfVon(z)
		nach[topf] = append(nach[topf], z)
	}
	aus := make([]Abschnitt[T], 0, len(mittelReihenfolge))
	for _, topf := range mittelReihenfolge {
		if topf == "" && len(nach[topf]) == 0 {
			continue
		}
		zeilenDesTopfs := nach[topf]
		if zeilenDesTopfs == nil {
			zeilenDesTopfs = []T{}
		}
		aus = append(aus, Abschnitt[T]{Topf: topf, Titel: mittelBeschriftung(topf), Zeilen: zeilenDesTopfs})
	}
	return aus
}

// bestandsbuchZeitraum liest von/bis aus der Anfrage; fehlt eines, gilt das laufende
// Schulhalbjahr (Stichtage 15.3./15.9., siehe schulzeit.Halbjahr).
//
// EINE Stelle für beide Bücher und für beide Türen jedes Buchs (Bildschirm und Blatt):
// Zwei Auslegungen von „laufendes Halbjahr" ergäben einen Ausdruck, der einen anderen
// Zeitraum abdeckt als die Liste, aus der er entstand.
func bestandsbuchZeitraum(r *http.Request) (von, bis time.Time, err error) {
	von, bis = schulzeit.Halbjahr(schulzeit.Jetzt())
	lies := func(schluessel string, ziel *time.Time) error {
		roh := r.URL.Query().Get(schluessel)
		if roh == "" {
			return nil
		}
		t, fehler := time.ParseInLocation(dateFormatISO, roh, schulzeit.Zone())
		if fehler != nil {
			return fmt.Errorf("%s muss ein Datum sein (JJJJ-MM-TT)", schluessel)
		}
		*ziel = t
		return nil
	}
	if err = lies("von", &von); err != nil {
		return von, bis, err
	}
	if err = lies("bis", &bis); err != nil {
		return von, bis, err
	}
	if bis.Before(von) {
		return von, bis, errors.New("das Ende des Zeitraums liegt vor seinem Anfang")
	}
	return von, bis, nil
}

// bestandsbuchKopf zeichnet den Briefkopf beider Bücher: Schule, Datum, Titel, Zeitraum.
func bestandsbuchKopf(p *gofpdf.Fpdf, tr func(string) string, titel string, von, bis time.Time, schule pdf.SchuleInfo) {
	p.SetFont("Arial", "B", 12)
	p.Cell(0, 8, tr(schule.Name))
	p.Ln(5)
	p.SetFont("Arial", "", 8)
	p.SetTextColor(100, 100, 100)
	p.Cell(0, 4, tr(schule.Absenderzeile()))
	p.SetTextColor(0, 0, 0)
	p.Ln(10)

	p.SetFont("Arial", "", 10)
	p.CellFormat(0, 6, tr(schule.OrtDatum(schulzeit.Jetzt().Format(dateFormatDE))), "", 1, "R", false, 0, "")
	p.Ln(4)

	p.SetFont("Arial", "B", 14)
	p.Cell(0, 10, tr(titel))
	p.Ln(7)
	p.SetFont("Arial", "", 10)
	p.SetTextColor(80, 80, 80)
	p.Cell(0, 6, tr(fmt.Sprintf("Zeitraum: %s bis %s", von.Format(dateFormatDE), bis.Format(dateFormatDE))))
	p.SetTextColor(0, 0, 0)
	p.Ln(12)
}
