package inventur

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	defer resp.Body.Close()

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
	filename := fmt.Sprintf("cover_auto_%s_%d%s", isbn, time.Now().Unix(), saveExt)
	savePath := filepath.Join("uploads", filename)

	if err := os.WriteFile(savePath, finalBytes, 0644); err != nil {
		log.Printf("Fehler beim lokalen Speichern von %s: %v", savePath, err)
		return coverURL // Fallback auf Remote URL
	}

	// Erfolg! Das Bild liegt lokal.
	return "/uploads/" + filename
}
