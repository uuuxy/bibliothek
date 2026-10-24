package inventur

import (
	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/coverquelle"
	"bibliothek/pkg/imageutil"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
)

// coverAbruferKennung ist der User-Agent, mit dem das Programm sich bei DNB, Google und
// OpenLibrary als Programm ausweist. Die Schnittstelle der DNB für Cover
// (portal.dnb.de/opac/mvb/cover) liegt hinter der Bot-Schranke "Anubis", die gerade bei
// User-Agents anschlägt, die wie ein Browser aussehen ("Mozilla/5.0 … Chrome …"): Solche
// Anfragen bekommen HTTP 200 mit einer Aufgabe in HTML statt des Bildes, und das Cover scheitert
// beim Dekodieren. Ein schlichter Name, der kein Browser ist, kommt durch und liefert das echte
// Bild (oder ein sauberes 404, wenn es kein Cover gibt). Deshalb nicht auf einen User-Agent von
// Chrome „aufrüsten".
const coverAbruferKennung = "Inventur/1.0"

// openLibraryLeeresCover ist die URL, die OpenLibrary für eine ISBN ohne Cover liefert: ein
// Platzhalter von 1x1 Punkt, kein Bild.
const openLibraryLeeresCover = "https://covers.openlibrary.org/b/isbn/-L.jpg"

// ladeCoverBytes lädt die Bilddaten einer whitelisteten Cover-URL (SSRF-Schutz) und
// verwirft Nicht-Bild-Antworten (Bot-Schranken-HTML) und Ersatzbilder der Quelle, die
// istErsatzbild erkennt. Liefert nil bei jedem Fehler.
func ladeCoverBytes(ctx context.Context, client *http.Client, coverURL string, istErsatzbild func([]byte) bool) []byte {
	if coverURL == "" || coverURL == openLibraryLeeresCover {
		return nil
	}

	// Angefragt wird die NEU gebaute URL, nicht die Eingabe: Geprüft hat url.Parse,
	// losgeschickt hätte der HTTP-Client den rohen String — und wo sich beide Parser
	// über ein Zeichen uneinig sind, geht die Anfrage an einen anderen Host als den
	// freigegebenen (Parsing Differential, gosec G704). Siehe pkg/coverquelle.
	sichereURL, ok := sichereCoverURL(coverURL)
	if !ok {
		slog.Warn("SSRF Schutz: Cover-URL stammt nicht von einem erlaubten Host", "url", coverURL)
		return nil
	}

	// #nosec G107 - URL aus geprüften Teilen neu gebaut (Host aus der Allowlist-Konstante)
	anfrage, err := http.NewRequestWithContext(ctx, http.MethodGet, sichereURL, nil)
	if err != nil {
		slog.Warn("Fehler beim Erstellen der Request für Cover", "url", sichereURL, "fehler", err)
		return nil
	}

	// Identische Programm-Identifikation wie beim DNB-HEAD-Check, sonst blockt die
	// DNB-Bot-Protection den Download (obwohl der Verfügbarkeits-Check bestand).
	anfrage.Header.Set("User-Agent", coverAbruferKennung)
	anfrage.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,*/*;q=0.8")

	httpAntwort, err := client.Do(anfrage)
	if err != nil {
		slog.Warn("Cover-Download fehlgeschlagen", "url", coverURL, "fehler", err)
		return nil
	}
	defer closeutil.Schliesse(httpAntwort.Body, "cover download")

	if httpAntwort.StatusCode != http.StatusOK {
		return nil
	}

	// Nicht-Bild-Antworten sofort verwerfen. Bot-Schranken (z. B. DNB/Anubis) liefern bei
	// einem falschen User-Agent HTTP 200 mit einer HTML-Challenge — das ist kein Cover.
	// So fällt der Aufrufer sauber auf die nächste Quelle zurück, statt HTML zu dekodieren.
	if contentType := httpAntwort.Header.Get("Content-Type"); strings.Contains(contentType, "html") || strings.Contains(contentType, "text/") || strings.Contains(contentType, "json") {
		slog.Info("Cover-Download: Nicht-Bild-Antwort — übersprungen", "content_type", contentType, "url", coverURL)
		return nil
	}

	inhalt, err := io.ReadAll(io.LimitReader(httpAntwort.Body, 10<<20)) // max. 10 MB
	if err != nil || len(inhalt) == 0 {
		return nil
	}
	// Ein Ersatzbild der Quelle ist kein Cover; die nächste Quelle ist dran.
	if istErsatzbild(inhalt) {
		return nil
	}
	return inhalt
}

// speichereCoverDatei schreibt das aufbereitete Cover in uploads/ und liefert den
// öffentlichen Pfad; "" bei Fehler. Der Name trägt die ISBN und eine Prüfsumme des Inhalts:
// Dieselbe Abfrage legt dieselbe Datei ab statt bei jedem Mal eine weitere, und ein anderes
// Bild bekommt einen anderen Namen, sodass kein Browser das alte weiter zeigt.
func speichereCoverDatei(fertig []byte, isbn, endung string) string {
	summe := sha256.Sum256(fertig)
	dateiname := fmt.Sprintf("cover_auto_%s_%x%s", filepath.Base(isbn), summe[:6], endung)

	if uploadDateiVorhanden(dateiname) {
		return "/uploads/" + dateiname
	}
	if err := schreibeUploadDatei(dateiname, fertig); err != nil {
		slog.Error("Cover-Download: Datei nicht gespeichert", "fehler", err)
		return "" // kein externer Fallback: lieber leer lassen und später erneut versuchen
	}
	return "/uploads/" + dateiname
}

// speichereCoverLokal lädt ein Bild von einer externen URL herunter,
// verkleinert es falls nötig, speichert es auf dem Server im Verzeichnis "uploads/"
// und gibt den lokalen Pfad zurück. Im Fehlerfall oder wenn es sich um Platzhalter handelt,
// wird die Original-URL (oder leer) zurückgegeben.
func speichereCoverLokal(ctx context.Context, client *http.Client, coverURL string, isbn string) string {
	inhalt := ladeCoverBytes(ctx, client, coverURL, coverquelle.IstErsatzbild)
	if inhalt == nil {
		return ""
	}

	// Erst den Bild-Header prüfen, dann dekodieren: Die 10-MB-Grenze in ladeCoverBytes
	// begrenzt die Leitung, nicht den Speicher — ein stark komprimiertes Riesenbild ist
	// klein auf der Leitung und groß im RAM (30000×30000 ≈ 3,6 GB). PruefeBildgroesse
	// liest nur den Header und allokiert keine Pixeldaten.
	if err := imageutil.PruefeBildgroesse(inhalt); err != nil {
		slog.Info("Cover-Download: Bild abgelehnt", "url", coverURL, "fehler", err)
		return ""
	}

	bild, _, err := image.Decode(bytes.NewReader(inhalt))
	if err != nil {
		return ""
	}

	rahmen := bild.Bounds()
	breite, hoehe := rahmen.Dx(), rahmen.Dy()

	// Sehr kleine Bilder (OpenLibrary Fallback-Platzhalter) ignorieren wir
	if breite < 10 || hoehe < 10 {
		return ""
	}

	fertig, endung, err := bereiteBildVor(bild, 600, 900, 80)
	if err != nil {
		return ""
	}

	return speichereCoverDatei(fertig, isbn, endung)
}
