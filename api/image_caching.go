package api

import (
	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/httpresp"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

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
		httpresp.Write(w, transparent1x1)
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
		if err := os.MkdirAll(dir, 0750); err != nil {
			serveFallback(w)
			return
		}

		root, err := os.OpenRoot(dir)
		if err != nil {
			serveFallback(w)
			return
		}
		defer closeutil.LogClose(root, "cover cache dir")

		// Sanity check to avoid unnecessary download/processing steps for obvious path traversals
		// even though root.OpenFile would safely block them later.
		if filepath.Base(isbn) != isbn {
			serveFallback(w)
			return
		}

		fileName := isbn + ".webp"

		// Serve cached version if it exists
		if _, err := root.Stat(fileName); err == nil {
			w.Header().Set("Cache-Control", "public, max-age=31536000")
			w.Header().Set("Content-Type", "image/webp")
			http.ServeFileFS(w, r, root.FS(), fileName) //nolint:gosec // Pre-existing G703
			return
		}

		// Download if missing
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, urlStr, nil) //nolint:gosec // Pre-existing G704
		if err != nil {
			serveFallback(w)
			return
		}

		req.Header.Set("User-Agent", "Inventur/1.0")

		resp, err := http.DefaultClient.Do(req) //nolint:gosec // Pre-existing G704
		if err != nil || resp.StatusCode != http.StatusOK {
			serveFallback(w)
			return
		}
		defer closeutil.LogClose(resp.Body, "cover download")

		// Decode original image
		img, _, err := image.Decode(resp.Body)
		if err != nil {
			serveFallback(w)
			return
		}

		// Write to local file as WebP
		out, err := root.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err == nil {
			err = webp.Encode(out, img, &webp.Options{Lossless: false, Quality: 80})
			// A failed Close can leave a truncated cache file, so treat it like an encode error.
			if cerr := out.Close(); cerr != nil && err == nil {
				err = cerr
			}
			if err != nil {
				if rerr := root.Remove(fileName); rerr != nil { // cleanup if encoding/close fails
					log.Printf("cover cache: cleanup of %s failed: %v", fileName, rerr) //nolint:gosec // Pre-existing G706
				}
			}
		}

		// Serve the newly converted file if it exists
		if _, err := root.Stat(fileName); err == nil {
			w.Header().Set("Cache-Control", "public, max-age=31536000")
			w.Header().Set("Content-Type", "image/webp")
			http.ServeFileFS(w, r, root.FS(), fileName) //nolint:gosec // Pre-existing G703
		} else {
			serveFallback(w)
		}
	}
}
