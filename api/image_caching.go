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
func (s *Server) ServeCoverImageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isbn := r.URL.Query().Get("isbn")
		urlStr := r.URL.Query().Get("url")

		if isbn == "" || urlStr == "" {
			http.Error(w, "missing isbn or url", http.StatusBadRequest)
			return
		}

		// SSRF Schutz für externe URLs
		parsed, err := url.Parse(urlStr)
		if err != nil {
			http.Error(w, "invalid url", http.StatusBadRequest)
			return
		}
		switch parsed.Hostname() {
		case "covers.openlibrary.org", "portal.dnb.de", "services.dnb.de", "www.googleapis.com", "openlibrary.org", "books.google.com", "books.google.de":
			// Erlaubte Hosts
		default:
			http.Error(w, "ssrf protection: host not allowed", http.StatusForbidden)
			return
		}

		dir := "uploads/covers"
		_ = os.MkdirAll(dir, 0755)
		cleanDir := filepath.Clean(dir)
		localPath := filepath.Clean(filepath.Join(cleanDir, isbn+".webp"))

		if !strings.HasPrefix(localPath, cleanDir+string(filepath.Separator)) {
			http.Error(w, "invalid path", http.StatusBadRequest)
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
			http.Error(w, "failed to create request", http.StatusInternalServerError)
			return
		}

		req.Header.Set("User-Agent", "Inventur/1.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			http.Error(w, "failed to fetch external image", http.StatusBadGateway)
			return
		}
		defer func() { _ = resp.Body.Close() }()

		// Decode original image
		img, _, err := image.Decode(resp.Body)
		if err != nil {
			http.Error(w, "failed to decode image", http.StatusInternalServerError)
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
			http.Error(w, "failed to serve converted image", http.StatusInternalServerError)
		}
	}
}
