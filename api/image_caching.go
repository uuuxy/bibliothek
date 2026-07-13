package api

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/httpresp"

	"github.com/chai2010/webp"
)

// coverFallbackGIF ist ein transparentes 1x1-GIF, das bei Fehlern ausgeliefert
// wird, um Browser-Konsolen-Spam zu vermeiden.
var coverFallbackGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00,
	0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x21,
	0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
	0x01, 0x00, 0x3b,
}

func serveCoverFallback(w http.ResponseWriter) {
	w.Header().Set(headerContentType, "image/gif")
	w.Header().Set(headerCacheControl, "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	httpresp.Write(w, coverFallbackGIF)
}

func serveCachedCover(w http.ResponseWriter, r *http.Request, root *os.Root, fileName string) {
	w.Header().Set(headerCacheControl, "public, max-age=31536000")
	w.Header().Set(headerContentType, "image/webp")
	http.ServeFileFS(w, r, root.FS(), fileName)
}

// istErlaubterCoverHost prüft die Cover-URL gegen die Host-Allowlist (SSRF-Schutz).
func istErlaubterCoverHost(urlStr string) bool {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	switch parsed.Hostname() {
	case "covers.openlibrary.org", "portal.dnb.de", "services.dnb.de", "www.googleapis.com", "openlibrary.org", "books.google.com", "books.google.de":
		return true
	default:
		return false
	}
}

// holeUndKonvertiereCover lädt das Cover herunter und speichert es als WebP im
// Cache-Verzeichnis. Bei Encode-/Close-Fehler wird die evtl. angefangene Datei
// wieder entfernt.
func holeUndKonvertiereCover(ctx context.Context, root *os.Root, urlStr, fileName string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Inventur/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer closeutil.LogClose(resp.Body, "cover download")
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cover download: unerwarteter Status %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return err
	}

	out, err := root.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	err = webp.Encode(out, img, &webp.Options{Lossless: false, Quality: 80})
	// A failed Close can leave a truncated cache file, so treat it like an encode error.
	if cerr := out.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		if rerr := root.Remove(fileName); rerr != nil { // cleanup if encoding/close fails
			log.Printf("cover cache: cleanup of %s failed: %v", fileName, rerr)
		}
		return err
	}
	return nil
}

// ServeCoverImageHandler serves a locally cached WebP image by ISBN, or downloads and converts it from URL if missing.
// On errors (invalid host, download failure), it serves a transparent 1x1 GIF to prevent browser console spam.
func (s *Server) ServeCoverImageHandler() http.HandlerFunc {
	return s.serveCoverImage
}

// serveCoverImage liefert ein lokal gecachtes WebP-Cover zur ISBN aus oder lädt und
// konvertiert es bei Bedarf. Bei jedem Fehler (ungültiger Host, Download-Fehler) wird
// ein transparentes 1x1-GIF ausgeliefert, um Browser-Konsolen-Spam zu vermeiden.
func (s *Server) serveCoverImage(w http.ResponseWriter, r *http.Request) {
	isbn := r.URL.Query().Get("isbn")
	urlStr := r.URL.Query().Get("url")

	if isbn == "" || urlStr == "" {
		serveCoverFallback(w)
		return
	}

	// SSRF-Schutz für externe URLs
	if !istErlaubterCoverHost(urlStr) {
		serveCoverFallback(w)
		return
	}

	dir := "uploads/covers"
	if err := os.MkdirAll(dir, 0750); err != nil {
		serveCoverFallback(w)
		return
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		serveCoverFallback(w)
		return
	}
	defer closeutil.LogClose(root, "cover cache dir")

	// Sanity check to avoid unnecessary download/processing steps for obvious path traversals
	// even though root.OpenFile would safely block them later.
	if filepath.Base(isbn) != isbn {
		serveCoverFallback(w)
		return
	}

	fileName := isbn + ".webp"

	// Serve cached version if it exists
	if _, err := root.Stat(fileName); err == nil {
		serveCachedCover(w, r, root, fileName)
		return
	}

	// Download & convert if missing
	if err := holeUndKonvertiereCover(r.Context(), root, urlStr, fileName); err != nil {
		serveCoverFallback(w)
		return
	}

	// Serve the newly converted file if it exists
	if _, err := root.Stat(fileName); err == nil {
		serveCachedCover(w, r, root, fileName)
	} else {
		serveCoverFallback(w)
	}
}
