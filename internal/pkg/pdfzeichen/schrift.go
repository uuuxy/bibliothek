package pdfzeichen

import (
	"embed"
	"fmt"

	"github.com/phpdave11/gofpdf"
)

// Die Familien der Ausdrucke. Google hat Arimo, Tinos und Cousine als Ersatz für Arial, Times und
// Courier gezeichnet: Jedes Zeichen ist genauso breit, ein Ausdruck behält also seinen Aufbau.
// Anders als die Standardschriften des PDF kennen sie jeden lateinischen Buchstaben (ş, ł, ǧ, ễ),
// dazu Griechisch und Kyrillisch.
const (
	Schrift      = "Arimo"   // für Arial
	SchriftSerif = "Tinos"   // für Times
	SchriftFest  = "Cousine" // für Courier: Nummern unter dem Strichcode
)

// schriftDateien hält die Schriftdateien im Programm; ihre Lizenz (SIL Open Font License 1.1)
// liegt in schriften/ daneben.
//
//go:embed schriften/*.ttf
var schriftDateien embed.FS

// Schnitt ist eine Schriftdatei mit Familie und Stil ("" normal, "B" fett, "I" kursiv), so wie
// gofpdf und maroto sie anmelden.
type Schnitt struct {
	Familie string
	Stil    string
	Daten   []byte
}

// Schnitte liest jeden Schnitt, den ein Ausdruck setzt. Ein Stil, der hier fehlt, bricht das
// Dokument mit „undefined font“ ab.
func Schnitte() ([]Schnitt, error) {
	var schnitte []Schnitt
	for _, s := range []struct{ familie, stil, datei string }{
		{Schrift, "", "Arimo-Regular.ttf"}, {Schrift, "B", "Arimo-Bold.ttf"}, {Schrift, "I", "Arimo-Italic.ttf"},
		{SchriftSerif, "", "Tinos-Regular.ttf"}, {SchriftSerif, "B", "Tinos-Bold.ttf"},
		{SchriftFest, "", "Cousine-Regular.ttf"}, {SchriftFest, "B", "Cousine-Bold.ttf"},
	} {
		daten, err := schriftDateien.ReadFile("schriften/" + s.datei)
		if err != nil {
			return nil, fmt.Errorf("schrift %s: %w", s.datei, err)
		}
		schnitte = append(schnitte, Schnitt{Familie: s.familie, Stil: s.stil, Daten: daten})
	}
	return schnitte, nil
}

// NewDokument legt ein A4-Dokument im Hochformat an, das die Schriften der Ausdrucke kennt.
// Jedes Dokument mit gofpdf beginnt hier; pruefung/pdfzeichen_ratsche_test.go hält das. Fehlt
// eine Schrift, meldet das Dokument den Fehler bei der Ausgabe, wie jeden anderen.
func NewDokument() *gofpdf.Fpdf {
	p := gofpdf.New("P", "mm", "A4", "")
	schnitte, err := Schnitte()
	if err != nil {
		p.SetError(err)
		return p
	}
	for _, s := range schnitte {
		p.AddUTF8FontFromBytes(s.Familie, s.Stil, s.Daten)
	}
	return p
}
