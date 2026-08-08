package api

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
	"os"
	"path/filepath"
	"strings"
	"time"

	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/coverquelle"
	"bibliothek/pkg/httpresp"
	"bibliothek/pkg/imageutil"
	"bibliothek/pkg/safehttp"

	"github.com/chai2010/webp"
)

// maxCoverBytes begrenzt, wie viel von einer fremden Antwort überhaupt in den Speicher
// gelesen wird. Ohne diese Grenze bestimmt der fremde Server, wie viel RAM ein einzelner
// Aufruf dieses Endpunkts kostet — und der Endpunkt ist öffentlich. 10 MB sind für ein
// Buchcover großzügig; dieselbe Grenze gilt im Cover-Downloader des Inventur-Moduls.
const maxCoverBytes = 10 << 20

// coverDownloadsProSekunde begrenzt NUR den teuren Zweig dieses Endpunkts: den
// Cache-Fehltreffer, der einen fremden Download samt vollständiger Bilddekodierung
// auslöst. Ausgelieferte Cache-Treffer bleiben ungebremst — sie sind ein Datei-Read,
// und ein Katalogaufruf lädt dutzende davon gleichzeitig (genau deshalb steht der
// Pfad in der Ausnahmeliste des globalen Limiters, siehe rate_limit.go).
//
// Ohne diese Bremse war der einzige unauthentifizierte Endpunkt der Anwendung, der
// eine ausgehende Verbindung und ein image.Decode auslöst, vollständig unbegrenzt:
// ein Aufruf = ein Download + eine Dekodierung, beliebig oft parallel.
// 30/s pro IP trägt den Erstaufruf einer Katalogseite (24 Kacheln) in einem Zug.
const coverDownloadsProSekunde = 30

// coverDownloadLimiter bremst die Fehltreffer pro Client-IP.
var coverDownloadLimiter = newIPRateLimiter(coverDownloadsProSekunde)

// istCoverCacheSchluessel prüft den vom Aufrufer gewählten Cache-Namen, bevor daraus
// ein Dateiname wird.
//
// Der Wert kommt aus buecher_titel.isbn und wurde ohne Prüfung zum Dateinamen im
// Cache-Verzeichnis. Zulässig sind Ziffern, Trennstriche und ein abschließendes X
// (ISBN-10-Prüfziffer) — genau die Zeichen, die eine ISBN oder EAN haben kann.
//
// Bewusst NICHT über isbnutil.CleanISBN geprüft, obwohl das naheliegt: Der Reiniger
// entfernt auch Leerzeichen, weil er ISBNs VERGLEICHBAR machen soll. Hier geht es um
// einen Dateinamen, und "9783161 84100" wäre danach gültig gewesen. Zwei verschiedene
// Fragen an dieselbe Zeichenkette (ist das dieselbe ISBN? / darf das ein Dateiname
// sein?) vertragen keine gemeinsame Antwort.
//
// Das ist Hygiene, keine Mengenbegrenzung: Gegen das Vollschreiben der Platte hilft
// die Download-Bremse oben, nicht diese Prüfung. Sie hält nur alles fern, was
// erkennbar keine Bestellnummer ist, und hält die Dateinamen vorhersagbar.
func istCoverCacheSchluessel(s string) bool {
	if s == "" || len(s) > 17 {
		return false
	}
	stellen := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c >= '0' && c <= '9':
			stellen++
		case c == '-':
			// Trennstriche zählen nicht als Stelle.
		case (c == 'X' || c == 'x') && i == len(s)-1:
			stellen++
		default:
			return false
		}
	}
	return stellen >= 8 && stellen <= 14
}

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

// baueSichereCoverURL validiert die Cover-URL gegen die Host-Allowlist und baut sie
// aus geprüften Teilen neu auf.
//
// Liste und Neuaufbau liegen seit dem 04.08.2026 in pkg/coverquelle: Dieselbe Prüfung
// braucht auch das Inventur-Modul (Cover-Download und Metadaten-Abruf), und die Liste
// stand dort in eigenen Kopien, die bereits auseinandergelaufen waren.
func baueSichereCoverURL(urlStr string) (string, bool) {
	return coverquelle.SichereURL(urlStr, coverquelle.CoverHosts)
}

// coverHTTPClient lädt Cover mit Schutzmaßnahmen, die http.DefaultClient fehlen:
// Verbindungen zu nicht-öffentlichen IPs werden auf Dialer-Ebene abgelehnt — nach
// der DNS-Auflösung und damit auch für jeden Redirect-Hop (OpenLibrary leitet z. B.
// real auf archive.org um) und bei DNS-Rebinding. Dazu ein hartes Gesamt-Timeout.
//
// Die Umsetzung liegt seit dem Audit in pkg/safehttp, weil das Inventur-Modul für
// seine Cover- und Metadaten-Abrufe denselben Schutz braucht und ihn vorher nicht
// hatte — dort standen blanke http.Client{Timeout: …}.
var coverHTTPClient = safehttp.NeuerClient(20 * time.Second)

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

	// Nicht-Bild-Antworten sofort verwerfen: Bot-Schranken (DNB/Anubis) liefern bei
	// unerwartetem User-Agent HTTP 200 mit einer HTML-Challenge. Gleiche Prüfung wie im
	// Cover-Downloader des Inventur-Moduls.
	if ct := resp.Header.Get(headerContentType); strings.Contains(ct, "html") || strings.Contains(ct, "text/") || strings.Contains(ct, "json") {
		return fmt.Errorf("cover download: Nicht-Bild-Antwort (%s)", ct)
	}

	// Erst begrenzt einlesen, dann den Header prüfen, dann dekodieren — in dieser
	// Reihenfolge, weil jeder Schritt den nächsten teurer macht:
	//   image.Decode direkt auf resp.Body hätte die Größe der Pixelmatrix vollständig
	//   dem fremden Server überlassen. Ein 30000×30000-PNG sind wenige hundert KB auf
	//   der Leitung und rund 3,6 GB im Speicher dieses Prozesses. Der Endpunkt ist
	//   öffentlich, und covers.openlibrary.org steht zwar auf der Allowlist, wird aber
	//   von Freiwilligen befüllt — die Allowlist begrenzt das Ziel, nicht den Inhalt.
	// GuardImageDimensions liest dafür nur den Bild-Header (image.DecodeConfig
	// allokiert keine Pixeldaten).
	rohBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxCoverBytes+1))
	if err != nil {
		return err
	}
	if len(rohBytes) > maxCoverBytes {
		return fmt.Errorf("cover download: Antwort überschreitet %d MB", maxCoverBytes>>20)
	}
	if err := imageutil.GuardImageDimensions(rohBytes); err != nil {
		return err
	}

	img, _, err := image.Decode(bytes.NewReader(rohBytes))
	if err != nil {
		return err
	}

	// 0600 wie in uploads_pfad.go, nicht 0666: Der Cover-Cache schrieb als einzige Stelle
	// welt-schreibbar. Im Container federt die umask das meist ab — „meist" ist bei
	// Dateirechten aber keine Zusage, sondern eine Wette auf die Laufzeitumgebung. Der
	// Prozess ist der einzige, der diese Dateien je anfassen muss; ausgeliefert werden
	// sie über den FileServer, nicht über das Dateisystem.
	out, err := root.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
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

	// Der Cache-Name wird gleich zum Dateinamen — vor allem anderen prüfen.
	if !istCoverCacheSchluessel(isbn) {
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

	// Ab hier wird es teuer: fremder Download plus vollständige Dekodierung. Nur dieser
	// Zweig ist gebremst — und er antwortet im Grenzfall mit dem Fallback-GIF statt mit
	// 429, weil am anderen Ende ein <img> hängt: Ein Fehlercode erzeugt dort nur einen
	// roten Konsoleneintrag, das Bild fehlt so oder so. Der nächste Aufruf holt es nach.
	if !coverDownloadLimiter.allow(getIP(r)) {
		serveCoverFallback(w)
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
