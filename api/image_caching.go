package api

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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

// erlaubteCoverHosts ist die Host-Allowlist für Cover-Downloads (SSRF-Schutz).
var erlaubteCoverHosts = []string{
	"covers.openlibrary.org",
	"portal.dnb.de",
	"services.dnb.de",
	"www.googleapis.com",
	"openlibrary.org",
	"books.google.com",
	"books.google.de",
}

// erlaubterCoverHost bildet einen Hostnamen auf die kanonische Allowlist-Konstante
// ab; "" heißt: nicht erlaubt. Zurückgegeben wird bewusst das Listen-Element und
// nicht die Eingabe, damit nachgelagerte URLs keinen ungeprüften Wert enthalten.
func erlaubterCoverHost(hostname string) string {
	for _, h := range erlaubteCoverHosts {
		if hostname == h {
			return h
		}
	}
	return ""
}

// baueSichereCoverURL validiert die Cover-URL gegen die Host-Allowlist und baut sie
// aus geprüften Teilen neu auf: Schema fest HTTPS, Host aus der Allowlist-Konstante —
// nur Pfad und Query stammen aus der Eingabe und können Host wie Schema des Requests
// nicht mehr beeinflussen (SSRF-Schutz).
func baueSichereCoverURL(urlStr string) (string, bool) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return "", false
	}
	host := erlaubterCoverHost(parsed.Hostname())
	if host == "" {
		return "", false
	}
	sichereURL := "https://" + host + "/" + strings.TrimPrefix(parsed.EscapedPath(), "/")
	if parsed.RawQuery != "" {
		sichereURL += "?" + parsed.RawQuery
	}
	return sichereURL, true
}

// coverHTTPClient lädt Cover mit Schutzmaßnahmen, die http.DefaultClient fehlen:
// Verbindungen zu nicht-öffentlichen IPs werden auf Dialer-Ebene abgelehnt — nach
// der DNS-Auflösung und damit auch für jeden Redirect-Hop (OpenLibrary leitet z. B.
// real auf archive.org um) und bei DNS-Rebinding. Dazu ein hartes Gesamt-Timeout.
var coverHTTPClient = &http.Client{
	Timeout: 20 * time.Second,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
			Control: verbieteInterneZieladressen,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

// verbieteInterneZieladressen lehnt Verbindungen zu Loopback-, privaten, Link-Local-,
// Multicast- und unspezifizierten Adressen ab. Läuft als Dialer-Control nach der
// DNS-Auflösung, address ist also immer eine aufgelöste IP:Port-Kombination.
func verbieteInterneZieladressen(_, address string, _ syscall.RawConn) error {
	addrPort, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("cover download: unerwartete Zieladresse %q: %w", address, err)
	}
	// Unmap: IPv4-in-IPv6 (::ffff:127.0.0.1) würde sonst an den Is*-Checks vorbeigehen.
	addr := addrPort.Addr().Unmap()
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return fmt.Errorf("cover download: Ziel-IP %s ist nicht öffentlich", addr)
	}
	return nil
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

	resp, err := coverHTTPClient.Do(req)
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

	// SSRF-Schutz: URL aus validierten Teilen neu aufbauen
	sichereURL, ok := baueSichereCoverURL(urlStr)
	if !ok {
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
	if err := holeUndKonvertiereCover(r.Context(), root, sichereURL, fileName); err != nil {
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
