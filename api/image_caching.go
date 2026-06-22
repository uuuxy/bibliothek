package api

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
)

// ServeCoverImageHandler serves a locally cached WebP image by ISBN, or downloads and converts it from URL if missing.
// On errors (invalid host, download failure), it serves a transparent 1x1 GIF to prevent browser console spam.
func (s *Server) ServeCoverImageHandler() http.HandlerFunc {
	var transparent1x1 = []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00,
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x21,
		0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
		0x01, 0x00, 0x3b,
	}

	serveFallback := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "image/gif")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.WriteHeader(http.StatusOK)
		w.Write(transparent1x1)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		isbn := r.URL.Query().Get("isbn")
		urlStr := r.URL.Query().Get("url")

		if isbn == "" || urlStr == "" {
			serveFallback(w)
			return
		}

		// SSRF Schutz für externe URLs
		parsed, err := url.Parse(urlStr)
		if err != nil {
			serveFallback(w)
			return
		}
		switch parsed.Hostname() {
		case "covers.openlibrary.org", "portal.dnb.de", "services.dnb.de", "www.googleapis.com", "openlibrary.org", "books.google.com", "books.google.de":
			// Erlaubte Hosts
		default:
			serveFallback(w)
			return
		}

		dir := "uploads/covers"
		_ = os.MkdirAll(dir, 0755)
		cleanDir := filepath.Clean(dir)
		localPath := filepath.Clean(filepath.Join(cleanDir, isbn+".webp"))

		if !strings.HasPrefix(localPath, cleanDir+string(filepath.Separator)) {
			serveFallback(w)
			return
		}

		// Serve cached version if it exists
		if _, err := os.Stat(localPath); err == nil {
			w.Header().Set("Cache-Control", "public, max-age=31536000")
			w.Header().Set("Content-Type", "image/webp")
			http.ServeFile(w, r, localPath)
			return
		}

		// Download if missing
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, urlStr, nil)
		if err != nil {
			serveFallback(w)
			return
		}

		req.Header.Set("User-Agent", "Inventur/1.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			serveFallback(w)
			return
		}
		defer func() { _ = resp.Body.Close() }()

		// Decode original image
		img, _, err := image.Decode(resp.Body)
		if err != nil {
			serveFallback(w)
			return
		}

		// Write to local file as WebP
		out, err := os.Create(localPath)
		if err == nil {
			err = webp.Encode(out, img, &webp.Options{Lossless: false, Quality: 80})
			_ = out.Close()
			if err != nil {
				_ = os.Remove(localPath) // cleanup if encoding fails
			}
		}

		// Serve the newly converted file if it exists
		if _, err := os.Stat(localPath); err == nil {
			w.Header().Set("Cache-Control", "public, max-age=31536000")
			w.Header().Set("Content-Type", "image/webp")
			http.ServeFile(w, r, localPath)
		} else {
			serveFallback(w)
		}
	}
}
