// Package coverdatei liest lokal gespeicherte Buchcover und liefert sie in einer Form,
// die PDF-Erzeuger einbetten können.
//
// Warum ein eigenes Paket: Cover liegen als WebP unter „uploads/" (inventur/
// cover_storage.go), aber weder gofpdf noch maroto kennen WebP. Der Weg dorthin —
// Pfad prüfen, Datei innerhalb des Wurzelverzeichnisses öffnen, nach JPEG wandeln —
// stand bis zum 03.09.2026 nur im Mahnwesen (api/mahnwesen_pdf.go). Der Schulbuch-
// Export braucht denselben Weg; eine zweite Fassung wäre die Gelegenheit gewesen,
// die Pfadprüfung schwächer nachzubauen.
package coverdatei

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/imageutil"
)

// Wurzel ist das Verzeichnis aller lokal gespeicherten Bilder, relativ zum
// Arbeitsverzeichnis des Servers.
const Wurzel = "uploads"

// Pfad löst die Cover-URL eines Titels in einen lesbaren lokalen Dateipfad auf.
// Rückgabe "" heißt: kein verwendbares Cover — der Aufrufer zeichnet dann nur den Rahmen.
//
// Der Clean-und-Prefix-Test ist kein Selbstzweck: Ohne ihn würde eine Cover-URL wie
// "/uploads/../../etc/passwd" aus dem Upload-Verzeichnis ausbrechen und beliebige
// Dateien in ein PDF einbetten lassen. Die Cover-URL stammt aus der Datenbank und
// damit aus Importen — sie ist keine geprüfte Eingabe.
func Pfad(coverURL string) string {
	if !strings.HasPrefix(coverURL, "/"+Wurzel+"/") {
		return ""
	}
	pfad := filepath.Clean(strings.TrimPrefix(coverURL, "/"))
	if pfad != Wurzel && !strings.HasPrefix(pfad, Wurzel+string(filepath.Separator)) {
		return ""
	}
	info, err := os.Stat(pfad)
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
// still — genau eine Zeile ohne Bild statt einer Liste, die mit 500 endet.
func AlsJPEG(coverURL string) (bilddaten []byte, pfad string, ok bool) {
	pfad = Pfad(coverURL)
	if pfad == "" {
		return nil, "", false
	}
	wurzel, err := os.OpenRoot(Wurzel)
	if err != nil {
		return nil, "", false
	}
	defer closeutil.LogClose(wurzel, "coverdatei wurzel")

	rel, err := filepath.Rel(Wurzel, pfad)
	if err != nil {
		return nil, "", false
	}
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
