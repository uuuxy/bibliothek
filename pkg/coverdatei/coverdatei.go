// Package coverdatei liest lokal gespeicherte Cover und setzt sie in ein PDF.
//
// Cover liegen als WebP unter „uploads/", und gofpdf kennt WebP nicht. Den Weg dorthin (Pfad
// prüfen, Datei innerhalb der Wurzel öffnen, nach JPEG wandeln) brauchen die Mahnliste und die
// Lernmittelliste. Stünde er zweimal im Code, hätte die zweite Fassung leicht die schwächere
// Pfadprüfung.
package coverdatei

import (
	"bytes"
	"io"
	"math"

	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/coverablage"
	"bibliothek/pkg/imageutil"

	"github.com/jung-kurt/gofpdf"
)

// Wurzel ist das Verzeichnis aller lokal gespeicherten Bilder, relativ zum Arbeitsverzeichnis
// des Servers. Pfadprüfung und Zugriff liegen in pkg/coverablage, das ohne Bildbibliothek
// auskommt.
const Wurzel = coverablage.Wurzel

// lokalerPfad liefert zur Cover-URL eines Titels den Pfad der Datei oder "", wenn es kein
// verwendbares Cover gibt.
func lokalerPfad(coverURL string) string {
	rel, pfad, ok := coverablage.Zerlege(coverURL)
	if !ok {
		return ""
	}
	wurzel, ok := coverablage.OeffneWurzel()
	if !ok {
		return ""
	}
	defer closeutil.Schliesse(wurzel, "coverdatei wurzel")

	info, err := wurzel.Stat(rel)
	if err != nil || info.IsDir() {
		return ""
	}
	return pfad
}

// alsJPEG liest das Cover und liefert es als Baseline-JPEG; ok ist false, wenn es kein
// verwendbares Cover gibt.
//
// Jeder Fehler führt zu ok=false und nicht zu einem Fehlerwert: Ein unlesbares, kaputtes oder
// übergroßes Cover soll eine Zeile ohne Bild kosten und nicht das ganze Dokument. Ein
// Verzeichnis braucht keine eigene Abweisung, weil io.ReadAll daran scheitert.
func alsJPEG(coverURL string) (bilddaten []byte, pfad string, ok bool) {
	rel, pfad, ok := coverablage.Zerlege(coverURL)
	if !ok {
		return nil, "", false
	}
	wurzel, ok := coverablage.OeffneWurzel()
	if !ok {
		return nil, "", false
	}
	defer closeutil.Schliesse(wurzel, "coverdatei wurzel")

	datei, err := wurzel.Open(rel)
	if err != nil {
		return nil, "", false
	}
	defer closeutil.Schliesse(datei, "coverdatei bild")

	roh, err := io.ReadAll(datei)
	if err != nil {
		return nil, "", false
	}
	jpg, err := imageutil.AlsJPEG(roh, imageutil.JPEGQualitaet)
	if err != nil {
		return nil, "", false
	}
	return jpg, pfad, true
}

// Flaeche nennt, wohin ein Cover auf dem Blatt kommt: die Ecke oben links und die größte
// Breite und Höhe, in der Einheit des Dokuments.
type Flaeche struct {
	X, Y, Breite, Hoehe float64
}

// BindeEin setzt das lokal gespeicherte Cover in die Fläche, im Seitenverhältnis des Bilds und
// darin mittig, und meldet, ob es eines gab. Fehler bleiben still: Ein Fehler an einem
// gofpdf-Dokument bleibt bis Output stehen, ein unlesbares Cover kostete sonst das ganze Blatt.
func BindeEin(pdf *gofpdf.Fpdf, coverURL string, flaeche Flaeche) bool {
	opt := gofpdf.ImageOptions{ImageType: "JPG"}
	name := lokalerPfad(coverURL)
	if name == "" {
		return false
	}
	// gofpdf hält ein Bild unter seinem Namen vor: Derselbe Titel in mehreren Zeilen wird nur
	// einmal gelesen und gewandelt.
	info := pdf.GetImageInfo(name)
	if info == nil {
		jpg, _, ok := alsJPEG(coverURL)
		if !ok {
			return false
		}
		info = pdf.RegisterImageOptionsReader(name, opt, bytes.NewReader(jpg))
	}
	if info == nil || info.Width() <= 0 || info.Height() <= 0 {
		return false
	}
	massstab := math.Min(flaeche.Breite/info.Width(), flaeche.Hoehe/info.Height())
	breite, hoehe := info.Width()*massstab, info.Height()*massstab
	pdf.ImageOptions(name, flaeche.X+(flaeche.Breite-breite)/2, flaeche.Y+(flaeche.Hoehe-hoehe)/2, breite, hoehe, false, opt, 0, "")
	return true
}
