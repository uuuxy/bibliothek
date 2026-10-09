// Package coverdatei liest lokal gespeicherte Buchcover und liefert sie in einer Form,
// die PDF-Erzeuger einbetten können.
//
// Warum ein eigenes Paket: Cover liegen als WebP unter „uploads/" (inventur/
// cover_storage.go), aber weder gofpdf noch maroto kennen WebP. Den Weg dorthin —
// Pfad prüfen, Datei innerhalb des Wurzelverzeichnisses öffnen, nach JPEG wandeln —
// brauchen die Mahnliste (pdf/mahnliste.go) und der Schulbuch-Export
// (inventur/lernmittel_pdf.go); eine zweite Fassung wäre die Gelegenheit, die
// Pfadprüfung schwächer nachzubauen.
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

// Wurzel ist das Verzeichnis aller lokal gespeicherten Bilder, relativ zum
// Arbeitsverzeichnis des Servers. Pfadprüfung und Zugriff auf das Verzeichnis liegen in
// pkg/coverablage, das ohne Bildbibliothek auskommt.
const Wurzel = coverablage.Wurzel

// Pfad löst die Cover-URL eines Titels in einen lesbaren lokalen Dateipfad auf.
// Rückgabe "" heißt: kein verwendbares Cover — der Aufrufer zeichnet dann nur den Rahmen.
func Pfad(coverURL string) string {
	rel, pfad, ok := coverablage.Zerlege(coverURL)
	if !ok {
		return ""
	}
	wurzel, ok := coverablage.OeffneWurzel()
	if !ok {
		return ""
	}
	defer closeutil.LogClose(wurzel, "coverdatei wurzel")

	info, err := wurzel.Stat(rel)
	if err != nil || info.IsDir() {
		return ""
	}
	return pfad
}

// AlsJPEG liest das Cover und gibt es als Baseline-JPEG zurück; ok ist false, wenn es
// kein verwendbares Cover gibt.
//
// Jeder Fehler führt zu ok=false statt zu einem Fehlerwert: Ein unlesbares, defektes
// oder überdimensioniertes Cover darf nie das ganze Dokument kosten. Es fehlt dann
// still — genau eine Zeile ohne Bild statt einer Liste, die mit 500 endet. Ein
// Verzeichnis braucht keine eigene Abweisung: io.ReadAll scheitert daran von selbst.
func AlsJPEG(coverURL string) (bilddaten []byte, pfad string, ok bool) {
	rel, pfad, ok := coverablage.Zerlege(coverURL)
	if !ok {
		return nil, "", false
	}
	wurzel, ok := coverablage.OeffneWurzel()
	if !ok {
		return nil, "", false
	}
	defer closeutil.LogClose(wurzel, "coverdatei wurzel")

	f, err := wurzel.Open(rel)
	if err != nil {
		return nil, "", false
	}
	defer closeutil.LogClose(f, "coverdatei bild")

	roh, err := io.ReadAll(f)
	if err != nil {
		return nil, "", false
	}
	jpg, err := imageutil.ConvertToJPEG(roh, imageutil.DefaultJPEGQuality)
	if err != nil {
		return nil, "", false
	}
	return jpg, pfad, true
}

// CoverPlatz nennt, wohin ein Cover auf dem Blatt kommt: die Ecke oben links und die größte
// Breite und Höhe, in der Einheit des Dokuments.
type CoverPlatz struct {
	X, Y, Breite, Hoehe float64
}

// BindeEin setzt das lokal gespeicherte Cover in den Platz, im Seitenverhältnis des Bilds und
// darin mittig, und meldet, ob es eines gab. Fehler bleiben still: Ein Fehler an einem
// gofpdf-Dokument bleibt bis Output stehen, ein unlesbares Cover kostete sonst das ganze Blatt.
func BindeEin(doc *gofpdf.Fpdf, coverURL string, platz CoverPlatz) bool {
	opt := gofpdf.ImageOptions{ImageType: "JPG"}
	name := Pfad(coverURL)
	if name == "" {
		return false
	}
	// gofpdf hält ein Bild unter seinem Namen vor: Derselbe Titel in mehreren Zeilen wird nur
	// einmal gelesen und gewandelt.
	info := doc.GetImageInfo(name)
	if info == nil {
		jpg, _, ok := AlsJPEG(coverURL)
		if !ok {
			return false
		}
		info = doc.RegisterImageOptionsReader(name, opt, bytes.NewReader(jpg))
	}
	if info == nil || info.Width() <= 0 || info.Height() <= 0 {
		return false
	}
	massstab := math.Min(platz.Breite/info.Width(), platz.Hoehe/info.Height())
	breite, hoehe := info.Width()*massstab, info.Height()*massstab
	doc.ImageOptions(name, platz.X+(platz.Breite-breite)/2, platz.Y+(platz.Hoehe-hoehe)/2, breite, hoehe, false, opt, 0, "")
	return true
}
