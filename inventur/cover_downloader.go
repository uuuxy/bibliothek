package inventur

import (
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

// downloadAndSaveCoverLocally lädt ein Bild von einer externen URL herunter,
// verkleinert es falls nötig, speichert es auf dem Server im Verzeichnis "uploads/"
// und gibt den lokalen Pfad zurück. Im Fehlerfall oder wenn es sich um Platzhalter handelt,
// wird die Original-URL (oder leer) zurückgegeben.
func downloadAndSaveCoverLocally(ctx context.Context, client *http.Client, coverURL string, isbn string) string {
	if coverURL == "" || coverURL == "https://covers.openlibrary.org/b/isbn/-L.jpg" {
		return ""
	}

	parsed, urlErr := url.Parse(coverURL)
	if urlErr != nil {
		log.Printf("Ungültige Cover-URL: %s", coverURL)
		return ""
	}
	switch parsed.Hostname() {
	case "covers.openlibrary.org", "portal.dnb.de", "services.dnb.de", "www.googleapis.com", "openlibrary.org", "books.google.com", "books.google.de":
		// Erlaubte Hosts
	default:
		log.Printf("SSRF Schutz: Cover-URL Hostname %s ist nicht in der Whitelist", parsed.Hostname())
		return ""
	}

	// #nosec G107 - URL wird sicher aus internen Const/Whitelist generiert
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, coverURL, nil)
	if err != nil {
		log.Printf("Fehler beim Erstellen der Request für Cover %s: %v", coverURL, err)
		return coverURL
	}

	// Viele APIs (z.B. Google, DNB) blocken reine Skripte ohne legitimen User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Inventur/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Cover-Download fehlgeschlagen für %s: %v", coverURL, err)
		return coverURL
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return coverURL
	}

	fileBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // max. 10 MB
	if err != nil || len(fileBytes) == 0 {
		return coverURL
	}

	// Sicherheit: Bild direkt decodieren
	img, format, err := image.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return coverURL
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Sehr kleine Bilder (OpenLibrary Fallback-Platzhalter) ignorieren wir
	if width < 10 || height < 10 {
		return ""
	}

	finalBytes, saveExt, err := prepareImageForStorage(fileBytes, img, format, 600, 900, 82)
	if err != nil {
		return coverURL
	}

	_ = os.MkdirAll("uploads", 0750)
	cleanDir := filepath.Clean("uploads")
	filename := fmt.Sprintf("cover_auto_%s_%d%s", filepath.Base(isbn), time.Now().Unix(), saveExt)
	savePath := filepath.Clean(filepath.Join(cleanDir, filename))

	if !strings.HasPrefix(savePath, cleanDir+string(filepath.Separator)) {
		log.Printf("Path traversal attempt in cover downloader: %s", isbn)
		return coverURL
	}

	if err := os.WriteFile(savePath, finalBytes, 0600); err != nil {
		log.Printf("Fehler beim lokalen Speichern von %s: %v", savePath, err)
		return coverURL // Fallback auf Remote URL
	}

	// Erfolg! Das Bild liegt lokal.
	return "/uploads/" + filename
}
