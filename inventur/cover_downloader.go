package inventur

import (
	"bibliothek/pkg/closeutil"
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "golang.org/x/image/webp"
)

// coverFetchUserAgent identifiziert uns gegenüber DNB/Google/OpenLibrary als ehrliches
// Programm. WICHTIG (verifiziert 2026-06-24): Die DNB-MVB-Cover-Schnittstelle
// (portal.dnb.de/opac/mvb/cover) liegt hinter der Bot-Schranke "Anubis". Diese triggert
// gerade auf BROWSER-ähnliche User-Agents ("Mozilla/5.0 … Chrome …"): solche Requests
// erhalten HTTP 200 mit einer HTML-Proof-of-Work-Challenge STATT des Bildes — das Cover
// schlägt dann beim Dekodieren fehl. Ein schlichter, nicht-browserartiger Programm-UA wird
// von Anubis als legitimer API-Client durchgelassen und liefert das echte Bild (bzw. ein
// sauberes 404, wenn kein Cover existiert). Daher NICHT auf einen Chrome-UA „aufrüsten".
const coverFetchUserAgent = "Inventur/1.0"

// ladeCoverBytes lädt die Bilddaten einer whitelisteten Cover-URL (SSRF-Schutz) und
// verwirft Nicht-Bild-Antworten (Bot-Schranken-HTML). Liefert nil bei jedem Fehler.
func ladeCoverBytes(ctx context.Context, client *http.Client, coverURL string) []byte {
	if coverURL == "" || coverURL == openLibraryLeeresCover {
		return nil
	}

	parsed, urlErr := url.Parse(coverURL)
	if urlErr != nil {
		log.Printf("Ungültige Cover-URL: %s", coverURL)
		return nil
	}
	switch parsed.Hostname() {
	case "covers.openlibrary.org", "portal.dnb.de", "services.dnb.de", "www.googleapis.com", "openlibrary.org", "books.google.com", "books.google.de":
		// Erlaubte Hosts
	default:
		log.Printf("SSRF Schutz: Cover-URL Hostname %s ist nicht in der Whitelist", parsed.Hostname())
		return nil
	}

	// #nosec G107 - URL wird sicher aus internen Const/Whitelist generiert
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, coverURL, nil)
	if err != nil {
		log.Printf("Fehler beim Erstellen der Request für Cover %s: %v", coverURL, err)
		return nil
	}

	// Identische Programm-Identifikation wie beim DNB-HEAD-Check, sonst blockt die
	// DNB-Bot-Protection den Download (obwohl der Verfügbarkeits-Check bestand).
	req.Header.Set("User-Agent", coverFetchUserAgent)
	req.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Cover-Download fehlgeschlagen für %s: %v", coverURL, err)
		return nil
	}
	defer closeutil.LogClose(resp.Body, "cover download")

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	// Nicht-Bild-Antworten sofort verwerfen. Bot-Schranken (z. B. DNB/Anubis) liefern bei
	// einem falschen User-Agent HTTP 200 mit einer HTML-Challenge — das ist kein Cover.
	// So fällt der Aufrufer sauber auf die nächste Quelle zurück, statt HTML zu dekodieren.
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "html") || strings.Contains(ct, "text/") || strings.Contains(ct, "json") {
		log.Printf("Cover-Download: Nicht-Bild-Antwort (%s) für %s — übersprungen", ct, coverURL)
		return nil
	}

	fileBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // max. 10 MB
	if err != nil || len(fileBytes) == 0 {
		return nil
	}
	return fileBytes
}

// speichereCoverDatei schreibt das aufbereitete Cover unter einem serverseitig
// generierten (traversal-geschützten) Pfad in uploads/ und liefert den öffentlichen
// Pfad; "" bei Fehler.
func speichereCoverDatei(finalBytes []byte, isbn, saveExt string) string {
	if err := os.MkdirAll("uploads", 0750); err != nil {
		log.Printf("Cover-Download: uploads-Verzeichnis konnte nicht angelegt werden: %v", err)
		return ""
	}
	cleanDir := filepath.Clean("uploads")
	filename := fmt.Sprintf("cover_auto_%s_%d%s", filepath.Base(isbn), time.Now().Unix(), saveExt)
	savePath := filepath.Clean(filepath.Join(cleanDir, filename))

	if !strings.HasPrefix(savePath, cleanDir+string(filepath.Separator)) {
		log.Printf("Path traversal attempt in cover downloader: %s", isbn)
		return ""
	}

	if err := os.WriteFile(savePath, finalBytes, 0600); err != nil {
		log.Printf("Fehler beim lokalen Speichern von %s: %v", savePath, err)
		return "" // kein externer Fallback: lieber leer lassen und später erneut versuchen
	}

	// Erfolg! Das Bild liegt lokal.
	return "/uploads/" + filename
}

// downloadAndSaveCoverLocally lädt ein Bild von einer externen URL herunter,
// verkleinert es falls nötig, speichert es auf dem Server im Verzeichnis "uploads/"
// und gibt den lokalen Pfad zurück. Im Fehlerfall oder wenn es sich um Platzhalter handelt,
// wird die Original-URL (oder leer) zurückgegeben.
func downloadAndSaveCoverLocally(ctx context.Context, client *http.Client, coverURL string, isbn string) string {
	fileBytes := ladeCoverBytes(ctx, client, coverURL)
	if fileBytes == nil {
		return ""
	}

	// Sicherheit: Bild direkt decodieren
	img, _, err := image.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return ""
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Sehr kleine Bilder (OpenLibrary Fallback-Platzhalter) ignorieren wir
	if width < 10 || height < 10 {
		return ""
	}

	finalBytes, saveExt, err := prepareImageForStorage(img, 600, 900, 80)
	if err != nil {
		return ""
	}

	return speichereCoverDatei(finalBytes, isbn, saveExt)
}
