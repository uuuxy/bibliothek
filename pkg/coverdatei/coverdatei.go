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
	"io"

	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/coverablage"
	"bibliothek/pkg/imageutil"
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
