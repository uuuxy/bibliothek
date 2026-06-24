package inventur

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// MetadatenClient ist der zentrale HTTP-Client zur Abfrage von Buchmetadaten
// über externe APIs (DNB, Google Books, OpenLibrary etc.).
type MetadatenClient struct {
	httpClient *http.Client
	coverCache sync.Map
}

// MetadatenErgebnis bündelt die gefundenen Metadaten eines Buches in einer
// einheitlichen Struktur über alle genutzten APIs hinweg.
type MetadatenErgebnis struct {
	ISBN         string `json:"isbn"`
	Titel        string `json:"title"`
	Untertitel   string `json:"subtitle"`
	Autor        string `json:"author"`
	CoverURL     string `json:"coverUrl"`
	Fach         string `json:"subject"`
	KlassenStufe string `json:"grade"`
	Verlag       string `json:"verlag"`
	Jahr         string `json:"jahr"`
}

// NeuerMetadatenClient initialisiert den HTTP Client mit einem Timeout von 8 Sekunden,
// um ewig ladende APIs zu unterbrechen.
func NeuerMetadatenClient() *MetadatenClient {
	return &MetadatenClient{
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}
}

// SucheNachISBN iteriert der Reihe nach über verschiedene Buch-APIs (DNB, Google, OpenLibrary),
// bis für die gesuchte ISBN gültige Titel-/Autorendaten gefunden wurden.
func (client *MetadatenClient) SucheNachISBN(kontext context.Context, isbn string) (*MetadatenErgebnis, error) {
	var ergebnis *MetadatenErgebnis
	var fehler error

	// ISBN von Strichen und Leerzeichen befreien
	saubereIsbn := strings.ReplaceAll(isbn, "-", "")
	saubereIsbn = strings.ReplaceAll(saubereIsbn, " ", "")

	// SSRF und Argument Injection Schutz: Nur gültige ISBN-Zeichenfolgen zulassen
	// ⚡ Bolt: Using validiereISBN() which utilizes a pre-compiled regex instead of regexp.MatchString
	// to avoid expensive implicit regex compilation on every API invocation.
	if !validiereISBN(saubereIsbn) {
		return nil, fmt.Errorf("ungültiges ISBN format: sicherheitsabbruch")
	}

	// 1. DNB (Deutsche Nationalbibliothek)
	ergebnis, fehler = client.sucheDNB(kontext, saubereIsbn)
	if fehler == nil && ergebnis != nil && (ergebnis.Titel != "" || ergebnis.Autor != "") {
		return client.beendeSuche(kontext, ergebnis, saubereIsbn), nil
	}

	// 2. Google Books
	ergebnis, fehler = client.sucheGoogleBooks(kontext, saubereIsbn)
	if fehler == nil && ergebnis != nil && (ergebnis.Titel != "" || ergebnis.Autor != "") {
		return client.beendeSuche(kontext, ergebnis, saubereIsbn), nil
	}

	// 3. OpenLibrary
	ergebnis, fehler = client.sucheOpenLibrary(kontext, saubereIsbn)
	if fehler == nil && ergebnis != nil && (ergebnis.Titel != "" || ergebnis.Autor != "") {
		return client.beendeSuche(kontext, ergebnis, saubereIsbn), nil
	}

	return nil, fmt.Errorf("keine metadaten für ISBN gefunden")
}

// beendeSuche füllt das MetadatenErgebnis mit fehlenden Informationen auf.
// Primär geht es darum, bei jedem gefundenen Buch die automatische Kategorisierung
// (Fach + Klasse) durchzuführen sowie CoverURLs als Fallback vorzugeben.
func (client *MetadatenClient) beendeSuche(kontext context.Context, ergebnis *MetadatenErgebnis, isbn string) *MetadatenErgebnis {
	if ergebnis.CoverURL == "" {
		isbn13 := konvertiereISBN10zu13(isbn)
		dnbCoverURL := fmt.Sprintf("https://portal.dnb.de/opac/mvb/cover?isbn=%s", url.QueryEscape(isbn13))

		// Prüfen, ob das Cover-Ergebnis für diese ISBN bereits gecacht ist
		cachedErgebnis, found := client.coverCache.Load(isbn13)
		if found {
			if cachedErgebnis.(bool) {
				ergebnis.CoverURL = dnbCoverURL
			}
		} else {
			// #nosec G107 - URL wird sicher aus internen Const/Whitelist generiert
			anfrage, fehler := http.NewRequestWithContext(kontext, http.MethodHead, dnbCoverURL, nil)
			if fehler == nil {
				anfrage.Header.Set("User-Agent", coverFetchUserAgent)
				antwort, fehler := client.httpClient.Do(anfrage)
				if fehler == nil {
					defer func() { _ = antwort.Body.Close() }()

					hasCover := antwort.StatusCode == http.StatusOK
					client.coverCache.Store(isbn13, hasCover)

					if hasCover {
						ergebnis.CoverURL = dnbCoverURL
					}
				}
			}
		}

		if ergebnis.CoverURL == "" {
			// Try Google Books just for the cover
			if gbResult, err := client.sucheGoogleBooks(kontext, isbn); err == nil && gbResult != nil && gbResult.CoverURL != "" {
				ergebnis.CoverURL = gbResult.CoverURL
			}
		}

		if ergebnis.CoverURL == "" {
			// Fallback: OpenLibrary gibt immerhin ein leeres Platzhalter-Bild statt 404 zurück
			ergebnis.CoverURL = fmt.Sprintf("https://covers.openlibrary.org/b/isbn/%s-L.jpg", isbn13)
		}
	}

	// Falls wir jetzt (oder durch den API-Call) ein Cover gefunden haben,
	// laden wir dieses direkt auf den Server herunter, um es stabil für unsere
	// Nutzer lokal und ohne Tracking/Rate-Limit der APIs auszuliefern.
	if ergebnis.CoverURL != "" && strings.HasPrefix(ergebnis.CoverURL, "http") {
		ergebnis.CoverURL = downloadAndSaveCoverLocally(kontext, client.httpClient, ergebnis.CoverURL, isbn)
	}

	fach, klassenStufe := automatischeKategorisierung(ergebnis.Titel, ergebnis.Untertitel)
	ergebnis.Fach = fach
	ergebnis.KlassenStufe = klassenStufe
	return ergebnis
}

// holeInhalt ist eine HTTP-GET Wrapper-Funktion, die die Antwort einer API als Bytearray zurückliefert.
func (client *MetadatenClient) holeInhalt(kontext context.Context, apiURL string) ([]byte, error) {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("SSRF Schutz: ungültige URL")
	}
	switch parsed.Hostname() {
	case "services.dnb.de", "www.googleapis.com", "openlibrary.org", "covers.openlibrary.org", "portal.dnb.de":
		// OK
	default:
		return nil, fmt.Errorf("SSRF Schutz: Hostname %s ist nicht in der Whitelist", parsed.Hostname())
	}

	// #nosec G107 - URL wird sicher aus internen Const/Whitelist generiert
	anfrage, fehler := http.NewRequestWithContext(kontext, http.MethodGet, apiURL, nil)
	if fehler != nil {
		return nil, fehler
	}
	anfrage.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	antwort, fehler := client.httpClient.Do(anfrage)
	if fehler != nil {
		return nil, fehler
	}
	defer func() { _ = antwort.Body.Close() }()
	if antwort.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", antwort.StatusCode)
	}
	return io.ReadAll(io.LimitReader(antwort.Body, 2<<20)) // Max 2 MB
}
