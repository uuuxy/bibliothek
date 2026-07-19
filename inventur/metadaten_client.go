package inventur

import (
	"bibliothek/pkg/isbnutil"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MetadatenClient ist der zentrale HTTP-Client zur Abfrage von Buchmetadaten
// über externe APIs (DNB, Google Books, OpenLibrary etc.).
type MetadatenClient struct {
	httpClient *http.Client
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
	Zielgruppe   string `json:"zielgruppe"`   // Altersempfehlung des Verlags, z. B. "ab 10 Jahre" (DNB 653)
	BibKategorie string `json:"bibKategorie"` // Signatur-Kategorie der Schülerbücherei (Kinderbuch, Jugendbuch, Comic, Manga)
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
	saubereIsbn := isbnutil.Clean(isbn)
	// removed duplicate space cleaning handled by isbnutil.Clean

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

// beendeSuche füllt das MetadatenErgebnis mit fehlenden Informationen auf:
// automatische Kategorisierung (Fach + Klasse) sowie das lokal gespeicherte Cover.
func (client *MetadatenClient) beendeSuche(kontext context.Context, ergebnis *MetadatenErgebnis, isbn string) *MetadatenErgebnis {
	ergebnis.CoverURL = client.aufloeseCover(kontext, ergebnis, isbn)

	fach, klassenStufe := automatischeKategorisierung(ergebnis.Titel, ergebnis.Untertitel)
	ergebnis.Fach = fach
	ergebnis.KlassenStufe = klassenStufe
	return ergebnis
}

// aufloeseCover probiert mehrere Cover-Quellen der Reihe nach durch und gibt den lokalen
// /uploads/-Pfad der ERSTEN Quelle zurück, die tatsächlich ein dekodierbares Bild liefert.
// Schlägt jede Quelle fehl, wird "" zurückgegeben (= „kein Cover", der Titel bleibt erhalten).
//
// Wir verlassen uns bewusst NICHT mehr auf einen HEAD-Verfügbarkeits-Check: Eine Bot-Schranke
// kann mit HTTP 200 + HTML antworten (siehe coverFetchUserAgent), wodurch ein HEAD-200
// fälschlich „Cover vorhanden" bedeutete, die Fallback-Quellen blockierte und am Ende eine
// HTML-Seite statt eines Bildes lieferte. Stattdessen laden wir jeden Kandidaten echt herunter;
// downloadAndSaveCoverLocally akzeptiert nur dekodierbare Bilder ausreichender Größe. Ein
// fehlgeschlagener Download verwirft hier NICHT mehr ein bereits gefundenes Cover.
func (client *MetadatenClient) aufloeseCover(kontext context.Context, ergebnis *MetadatenErgebnis, isbn string) string {
	isbn13 := konvertiereISBN10zu13(isbn)

	// Kandidaten in Prioritätsreihenfolge: DNB primär (beste Quelle für deutsche Schulbücher).
	kandidaten := make([]string, 0, 4)

	// 1. Bereits aus der Metadatensuche (Google/OpenLibrary) bekanntes Cover.
	if ergebnis.CoverURL != "" && strings.HasPrefix(ergebnis.CoverURL, "http") {
		kandidaten = append(kandidaten, ergebnis.CoverURL)
	}

	// 2. DNB MVB Cover (primär).
	kandidaten = append(kandidaten, fmt.Sprintf("https://portal.dnb.de/opac/mvb/cover?isbn=%s", url.QueryEscape(isbn13)))

	// 3. Google Books — separat abfragen, falls die Metadaten von DNB kamen (DNB liefert keine Cover-URL im Datensatz).
	if gb, err := client.sucheGoogleBooks(kontext, isbn); err == nil && gb != nil && gb.CoverURL != "" {
		kandidaten = append(kandidaten, gb.CoverURL)
	}

	// 4. OpenLibrary — default=false erzwingt ein echtes 404 statt eines 1×1-Platzhalters.
	kandidaten = append(kandidaten, fmt.Sprintf("https://covers.openlibrary.org/b/isbn/%s-L.jpg?default=false", isbn13))

	for _, kandidat := range kandidaten {
		if lokal := downloadAndSaveCoverLocally(kontext, client.httpClient, kandidat, isbn); lokal != "" {
			return lokal
		}
	}
	return ""
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
	defer func() { _ = antwort.Body.Close() }() //nolint:errcheck
	if antwort.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", antwort.StatusCode)
	}
	return io.ReadAll(io.LimitReader(antwort.Body, 2<<20)) // Max 2 MB
}
