package katalog

import (
	"bytes"
	"context"
	"errors"
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

	"bibliothek/api/tuer"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/coverquelle"
	"bibliothek/pkg/httpresp"
	"bibliothek/pkg/imageutil"
	"bibliothek/pkg/ratelimit"
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
// Pfad in der Ausnahmeliste des globalen Limiters, siehe api/rate_limit.go).
//
// Ohne diese Bremse war der einzige unauthentifizierte Endpunkt der Anwendung, der
// eine ausgehende Verbindung und ein image.Decode auslöst, vollständig unbegrenzt:
// ein Aufruf = ein Download + eine Dekodierung, beliebig oft parallel.
// 30/s pro IP trägt den Erstaufruf einer Katalogseite (24 Kacheln) in einem Zug.
const coverDownloadsProSekunde = 30

// coverDownloadBremse bremst die Fehltreffer pro Client-IP.
var coverDownloadBremse = ratelimit.New(coverDownloadsProSekunde)

// coverCacheVerzeichnis ist eine Variable, damit Tests den Cache in ein Wegwerf-
// Verzeichnis legen können, ohne das Arbeitsverzeichnis zu wechseln.
var coverCacheVerzeichnis = "uploads/covers"

// istCoverCacheSchluessel prüft den vom Aufrufer gewählten Cache-Namen, bevor daraus
// ein Dateiname wird.
//
// Der Wert kommt aus buecher_titel.isbn und wurde ohne Prüfung zum Dateinamen im
// Cache-Verzeichnis. Zulässig sind Ziffern, Trennstriche und ein abschließendes X
// (ISBN-10-Prüfziffer) — genau die Zeichen, die eine ISBN oder EAN haben kann.
//
// Bewusst NICHT über isbnutil.OhneTrenner geprüft, obwohl das naheliegt: Der Reiniger
// entfernt auch Leerzeichen, weil er ISBNs VERGLEICHBAR machen soll. Hier geht es um
// einen Dateinamen, und "9783161 84100" wäre danach gültig gewesen. Zwei verschiedene
// Fragen an dieselbe Zeichenkette (ist das dieselbe ISBN? / darf das ein Dateiname
// sein?) vertragen keine gemeinsame Antwort.
//
// Das ist Hygiene, keine Mengenbegrenzung: Gegen das Vollschreiben der Platte hilft
// die Download-Bremse oben, nicht diese Prüfung. Sie hält nur alles fern, was
// erkennbar keine Bestellnummer ist, und hält die Dateinamen vorhersagbar.
func istCoverCacheSchluessel(schluessel string) bool {
	if schluessel == "" || len(schluessel) > 17 {
		return false
	}
	stellen := 0
	for i := 0; i < len(schluessel); i++ {
		switch c := schluessel[i]; {
		case c >= '0' && c <= '9':
			stellen++
		case c == '-':
			// Trennstriche zählen nicht als Stelle.
		case (c == 'X' || c == 'x') && i == len(schluessel)-1:
			stellen++
		default:
			return false
		}
	}
	return stellen >= 8 && stellen <= 14
}

// transparentesGIF ist ein transparentes 1x1-GIF, das bei Fehlern ausgeliefert
// wird, um Browser-Konsolen-Spam zu vermeiden.
var transparentesGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00,
	0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x21,
	0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
	0x01, 0x00, 0x3b,
}

func liefereTransparentesGIF(w http.ResponseWriter) {
	w.Header().Set(tuer.HeaderContentType, "image/gif")
	w.Header().Set(tuer.HeaderCacheControl, "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	httpresp.Schreibe(w, transparentesGIF)
}

func liefereCoverAusCache(w http.ResponseWriter, r *http.Request, root *os.Root, dateiname string) {
	w.Header().Set(tuer.HeaderCacheControl, "public, max-age=31536000")
	w.Header().Set(tuer.HeaderContentType, "image/webp")
	http.ServeFileFS(w, r, root.FS(), dateiname)
}

// baueSichereCoverURL validiert die Cover-URL gegen die Host-Allowlist und baut sie
// aus geprüften Teilen neu auf.
//
// Liste und Neuaufbau liegen seit dem 04.08.2026 in pkg/coverquelle: Dieselbe Prüfung
// braucht auch das Inventur-Modul (Cover-Download und Metadaten-Abruf), und die Liste
// stand dort in eigenen Kopien, die bereits auseinandergelaufen waren.
func baueSichereCoverURL(adresse string) (string, bool) {
	return coverquelle.SichereURL(adresse, coverquelle.BildHosts)
}

// coverHTTPClient lädt Cover mit Schutzmaßnahmen, die http.DefaultClient fehlen:
// Verbindungen zu nicht-öffentlichen IPs werden auf Dialer-Ebene abgelehnt — nach
// der DNS-Auflösung und damit auch für jeden Redirect-Hop (OpenLibrary leitet z. B.
// real auf archive.org um) und bei DNS-Rebinding. Dazu ein hartes Gesamt-Timeout.
//
// Die Umsetzung liegt seit dem Audit in pkg/safehttp, weil das Inventur-Modul für
// seine Cover- und Metadaten-Abrufe denselben Schutz braucht und ihn vorher nicht
// hatte — dort standen blanke http.Client{Timeout: …}.
var coverHTTPClient = safehttp.NewClient(20 * time.Second)

// holeUndKonvertiereCover lädt das Cover herunter und speichert es als WebP im
// Cache-Verzeichnis. Bei Encode-/Close-Fehler wird die evtl. angefangene Datei
// wieder entfernt.
func holeUndKonvertiereCover(ctx context.Context, root *os.Root, adresse, dateiname string) error {
	rohBytes, err := holeCoverBytes(ctx, adresse)
	if err != nil {
		return err
	}
	return schreibeCoverAlsWebP(root, dateiname, rohBytes)
}

// holeCoverBytes holt die Antwort der Quelle und weist ab, was kein Cover ist, bevor
// irgendetwas dekodiert wird.
func holeCoverBytes(ctx context.Context, adresse string) ([]byte, error) {
	anfrage, err := http.NewRequestWithContext(ctx, http.MethodGet, adresse, nil)
	if err != nil {
		return nil, err
	}
	anfrage.Header.Set("User-Agent", "Inventur/1.0")

	antwort, err := coverHTTPClient.Do(anfrage)
	if err != nil {
		return nil, err
	}
	defer closeutil.Schliesse(antwort.Body, "cover download")
	if antwort.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cover download: unerwarteter Status %d", antwort.StatusCode)
	}

	// Nicht-Bild-Antworten sofort verwerfen: Bot-Schranken (DNB/Anubis) liefern bei
	// unerwartetem User-Agent HTTP 200 mit einer HTML-Challenge. Gleiche Prüfung wie im
	// Cover-Downloader des Inventur-Moduls.
	if ct := antwort.Header.Get(tuer.HeaderContentType); strings.Contains(ct, "html") || strings.Contains(ct, "text/") || strings.Contains(ct, "json") {
		return nil, fmt.Errorf("cover download: Nicht-Bild-Antwort (%s)", ct)
	}

	// Erst begrenzt einlesen, dann den Header prüfen, dann dekodieren — in dieser
	// Reihenfolge, weil jeder Schritt den nächsten teurer macht:
	//   image.Decode direkt auf resp.Body hätte die Größe der Pixelmatrix vollständig
	//   dem fremden Server überlassen. Ein 30000×30000-PNG sind wenige hundert KB auf
	//   der Leitung und rund 3,6 GB im Speicher dieses Prozesses. Der Endpunkt ist
	//   öffentlich, und covers.openlibrary.org steht zwar auf der Allowlist, wird aber
	//   von Freiwilligen befüllt — die Allowlist begrenzt das Ziel, nicht den Inhalt.
	// PruefeBildgroesse liest dafür nur den Bild-Header (image.DecodeConfig
	// allokiert keine Pixeldaten).
	rohBytes, err := io.ReadAll(io.LimitReader(antwort.Body, maxCoverBytes+1))
	if err != nil {
		return nil, err
	}
	if len(rohBytes) > maxCoverBytes {
		return nil, fmt.Errorf("cover download: Antwort überschreitet %d MB", maxCoverBytes>>20)
	}
	// Ein Ersatzbild der Quelle ist kein Cover: Der Aufrufer zeigt dann die nächste Quelle
	// oder die Initiale des Titels.
	if coverquelle.IstErsatzbild(rohBytes) {
		return nil, errors.New("cover download: die Quelle hat zu dieser ISBN kein Cover (Ersatzbild)")
	}
	if err := imageutil.PruefeBildgroesse(rohBytes); err != nil {
		return nil, err
	}
	return rohBytes, nil
}

// schreibeCoverAlsWebP dekodiert das geprüfte Bild und legt es als WebP in den Cache. Eine
// angefangene Datei wird bei einem Fehler wieder entfernt.
func schreibeCoverAlsWebP(root *os.Root, dateiname string, rohBytes []byte) error {
	bild, _, err := image.Decode(bytes.NewReader(rohBytes))
	if err != nil {
		return err
	}

	// 0600 wie in uploads_pfad.go, nicht 0666: Im Container federt die umask das meist ab —
	// „meist" ist bei Dateirechten aber keine Zusage, sondern eine Wette auf die
	// Laufzeitumgebung. Der Prozess ist der einzige, der diese Dateien je anfassen muss;
	// ausgeliefert werden sie über den FileServer, nicht über das Dateisystem.
	datei, err := root.OpenFile(dateiname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	err = webp.Encode(datei, bild, &webp.Options{Lossless: false, Quality: 80})
	// A failed Close can leave a truncated cache file, so treat it like an encode error.
	if cerr := datei.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		if rerr := root.Remove(dateiname); rerr != nil { // cleanup if encoding/close fails
			log.Printf("cover cache: cleanup of %s failed: %v", dateiname, rerr)
		}
		return err
	}
	return nil
}

// Cover serves a locally cached WebP image by ISBN, or downloads and converts it from URL if missing.
// On errors (invalid host, download failure), it serves a transparent 1x1 GIF to prevent browser console spam.
func (t *Tueren) Cover() http.HandlerFunc {
	return t.liefereCover
}

// liefereCover liefert ein lokal gecachtes WebP-Cover zur ISBN aus oder lädt und
// konvertiert es bei Bedarf. Bei jedem Fehler (ungültiger Host, Download-Fehler) wird
// ein transparentes 1x1-GIF ausgeliefert, um Browser-Konsolen-Spam zu vermeiden.
func (t *Tueren) liefereCover(w http.ResponseWriter, r *http.Request) {
	isbn := r.URL.Query().Get("isbn")
	adresse := r.URL.Query().Get("url")

	if isbn == "" || adresse == "" {
		liefereTransparentesGIF(w)
		return
	}

	// Der Cache-Name wird gleich zum Dateinamen — vor allem anderen prüfen.
	if !istCoverCacheSchluessel(isbn) {
		liefereTransparentesGIF(w)
		return
	}

	// SSRF-Schutz: URL aus validierten Teilen neu aufbauen
	sichereURL, ok := baueSichereCoverURL(adresse)
	if !ok {
		liefereTransparentesGIF(w)
		return
	}

	verzeichnis := coverCacheVerzeichnis
	if err := os.MkdirAll(verzeichnis, 0750); err != nil {
		liefereTransparentesGIF(w)
		return
	}

	root, err := os.OpenRoot(verzeichnis)
	if err != nil {
		liefereTransparentesGIF(w)
		return
	}
	defer closeutil.Schliesse(root, "cover cache dir")

	// Sanity check to avoid unnecessary download/processing steps for obvious path traversals
	// even though root.OpenFile would safely block them later.
	if filepath.Base(isbn) != isbn {
		liefereTransparentesGIF(w)
		return
	}

	// ISBN + Hash der geprüften Adresse: Zwei Adressen teilen sich nie eine Datei, der
	// Erstaufrufer bestimmt nicht mehr, was andere unter dieser ISBN sehen.
	dateiname := coverDateiname(isbn, sichereURL)

	// Serve cached version if it exists
	if _, err := root.Stat(dateiname); err == nil {
		liefereCoverAusCache(w, r, root, dateiname)
		return
	}

	// Ab hier wird es teuer: fremder Download plus vollständige Dekodierung. Nur dieser
	// Zweig ist gebremst — und er antwortet im Grenzfall mit dem Fallback-GIF statt mit
	// 429, weil am anderen Ende ein <img> hängt: Ein Fehlercode erzeugt dort nur einen
	// roten Konsoleneintrag, das Bild fehlt so oder so. Der nächste Aufruf holt es nach.
	if !coverDownloadBremse.LaesstDurch(clientip.AusAnfrage(r)) {
		liefereTransparentesGIF(w)
		return
	}

	// Nur Adressen, die der Server für diese ISBN selbst kennt oder herleiten kann
	// (cover_quelle_bindung.go) — sonst wäre der Endpunkt ein offener Proxy für alle
	// Bilder der erlaubten Hosts und der Cache ein Plattenfüller.
	if !t.istCoverQuelleErlaubt(r, isbn, sichereURL) {
		liefereTransparentesGIF(w)
		return
	}

	// Download & convert if missing
	if err := holeUndKonvertiereCover(r.Context(), root, sichereURL, dateiname); err != nil {
		liefereTransparentesGIF(w)
		return
	}

	// Serve the newly converted file if it exists
	if _, err := root.Stat(dateiname); err == nil {
		liefereCoverAusCache(w, r, root, dateiname)
	} else {
		liefereTransparentesGIF(w)
	}
}
