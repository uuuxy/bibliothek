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

// zerlegeCoverURL prüft die Cover-URL und liefert den Dateipfad zweimal: einmal relativ
// zur Wurzel (so verlangt es os.Root) und einmal vollständig, weil die PDF-Erzeuger ihn
// als Namen ihres Bildspeichers führen. ok=false heißt: nicht verwendbar.
//
// Die Herkunftsprüfung ganz oben ist KEIN Schutz — das ist os.Root beim Zugriff (siehe
// oeffneWurzel), und nur der ließ sich am Rückbau rot zeigen. Sie steht hier als
// Vorfilter: Die allermeisten Titel haben gar kein Cover (gemessen am 17.09.2026: 3168
// von 3233 mit leerem cover_url), und für die soll kein Verzeichnis geöffnet werden.
// Eine zweite Textprüfung auf „beginnt mit uploads/" stand hier bis zum 17.09.2026;
// sie war gegenüber os.Root wirkungslos und ist ersatzlos entfallen.
func zerlegeCoverURL(coverURL string) (rel, pfad string, ok bool) {
	if !strings.HasPrefix(coverURL, "/"+Wurzel+"/") {
		return "", "", false
	}
	pfad = filepath.Clean(strings.TrimPrefix(coverURL, "/"))
	rel, err := filepath.Rel(Wurzel, pfad)
	if err != nil {
		return "", "", false
	}
	return rel, pfad, true
}

// oeffneWurzel öffnet das Upload-Verzeichnis als os.Root. Erst diese Klammer hält auch den
// Fall auf, den die Textprüfung oben nicht sehen kann: eine Verknüpfung INNERHALB von
// uploads, die nach draußen zeigt. os.Stat würde ihr folgen, root.Stat verweigert sie.
func oeffneWurzel() (*os.Root, bool) {
	wurzel, err := os.OpenRoot(Wurzel)
	if err != nil {
		return nil, false
	}
	return wurzel, true
}

// Pfad löst die Cover-URL eines Titels in einen lesbaren lokalen Dateipfad auf.
// Rückgabe "" heißt: kein verwendbares Cover — der Aufrufer zeichnet dann nur den Rahmen.
func Pfad(coverURL string) string {
	rel, pfad, ok := zerlegeCoverURL(coverURL)
	if !ok {
		return ""
	}
	wurzel, ok := oeffneWurzel()
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
	rel, pfad, ok := zerlegeCoverURL(coverURL)
	if !ok {
		return nil, "", false
	}
	wurzel, ok := oeffneWurzel()
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
