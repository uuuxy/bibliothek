package inventur

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bibliothek/pkg/coverquelle"
	"bibliothek/pkg/isbnutil"
	"bibliothek/pkg/safehttp"
	"bibliothek/repository"
)

// MetadatenClient fragt die Katalogdienste (DNB, Google Books, OpenLibrary) über HTTP nach den
// Angaben zu einem Buch.
type MetadatenClient struct {
	httpClient *http.Client
}

// MetadatenErgebnis bündelt die gefundenen Metadaten eines Buches in einer
// einheitlichen Struktur über alle genutzten APIs hinweg.
type MetadatenErgebnis struct {
	ISBN       string `json:"isbn"`
	Titel      string `json:"title"`
	Untertitel string `json:"subtitle"`
	Autor      string `json:"author"`
	CoverURL   string `json:"coverUrl"`
	Fach       string `json:"subject"`
	// JahrgangVon und JahrgangBis nennen die Schulstufe aus dem Titel als Spanne; 0 und 0
	// heißt wie am Buch, dass keine bekannt ist (stufenAusText).
	JahrgangVon int    `json:"jahrgangVon"`
	JahrgangBis int    `json:"jahrgangBis"`
	Verlag      string `json:"verlag"`
	Jahr        string `json:"jahr"`
	Zielgruppe  string `json:"zielgruppe"` // Altersempfehlung des Verlags, z. B. "ab 10 Jahre" (DNB 653)
	// Preis ist der DNB-Ladenpreis (MARC21 020 $c) als Vorschlag. 0 = keiner ermittelbar.
	// Er gilt zum Erscheinungszeitpunkt und ist nicht der Schulpreis — die Oberflaeche
	// darf ihn nur vorschlagen, nie als erfasste Ausgabe verbuchen.
	Preis float64 `json:"preis,omitempty"`
	// Stichwoerter sind die Gattungsbegriffe (655 $a) und die freien Wörter des Verlags (653 $a ohne
	// Vorsatz) der DNB, Kandidaten für den Vorschlag der Schlagworte beim Bestellen über die ISBN
	// (repository.ladeSchlagworteAusStichwoertern). Nie gespeichert und nie ausgeliefert: Ein Satz
	// trägt bis zu 98 Wörter des Verlags, darunter Werbung („TikTok", „Must Read"); vorgeschlagen
	// wird nur, was es in der eigenen Liste gibt.
	Stichwoerter []string `json:"-"`
	// Normdaten sind die Schlagwörter der Gemeinsamen Normdatei (MARC 600–651 mit $2 gnd), ein
	// festes Vokabular ohne Werbewörter. Angeboten werden sie auch, wenn die eigene Liste sie noch
	// nicht kennt (repository.ladeSchlagworteAusNormdaten). Nie gespeichert und nie ausgeliefert.
	Normdaten []repository.Normdatenbegriff `json:"-"`
}

// NewMetadatenClient baut den Client mit einer Frist von 8 Sekunden, damit ein hängender Dienst
// die Anfrage nicht aufhält.
//
// Der Client kommt aus pkg/safehttp und nicht mehr blank aus http.Client: Er spricht
// mit fremden Servern (DNB, Google, OpenLibrary, lobid) und lädt über dieselbe Instanz
// auch Cover herunter. Die Hostnamen-Allowlist in ladeCoverBytes prüft nur die URL,
// die WIR bilden — wohin ein erlaubter Host per Redirect weiterschickt, sah sie nicht.
// safehttp lehnt nicht-öffentliche Ziele auf Dialer-Ebene ab und damit bei jedem Hop.
func NewMetadatenClient() *MetadatenClient {
	return &MetadatenClient{
		httpClient: safehttp.NewClient(8 * time.Second),
	}
}

// SetzeHTTPClientFuerTest injiziert einen HTTP-Client mit eigenem Transport. Der Zweck
// ist ausschließlich, in Tests anderer Pakete den echten Netzzugriff zu kappen (DNB/
// Google/OpenLibrary): Innerhalb von inventur reicht das unexportierte httpClient-Feld,
// aber ein Test der Bestellsuche in internal/service kommt sonst nur über unsafe/reflect an
// das Feld — genau die Fragilität, die bei einer Feldumbenennung still bricht. Diese
// eine exportierte Naht macht den Weg sicher und benennt ihn ehrlich.
func (c *MetadatenClient) SetzeHTTPClientFuerTest(httpClient *http.Client) {
	c.httpClient = httpClient
}

// ErrKatalogdiensteNichtErreichbar heißt: Keine der Quellen (DNB, Google, OpenLibrary) war
// erreichbar. Das ist ein Ausfall (502 beim Aufrufer), kein Fehlen eines Treffers (404); sähen
// beide gleich aus, katalogisierte die Theke bei einer Störung des WLANs Bücher von Hand, die
// längst in der DNB stehen.
var ErrKatalogdiensteNichtErreichbar = errors.New("katalogdienste (DNB, Google Books, OpenLibrary) nicht erreichbar")

// errQuelleNichtErreichbar markiert Ausfälle einer Quelle (holeInhalt): Verbindung
// gescheitert oder Status ≥ 500. Alles andere — 4xx, leere Antwort, „nicht gefunden",
// unlesbare Nutzlast — heißt: Die Quelle war da, sie hatte nur nichts.
var errQuelleNichtErreichbar = errors.New("quelle nicht erreichbar")

// ErrDNBNichtErreichbar heißt: Die DNB hat nicht geantwortet (Verbindung gescheitert oder
// Status ≥ 500) — ein Ausfall, kein „kennt die ISBN nicht" (502 beim Aufrufer).
var ErrDNBNichtErreichbar = errors.New("DNB nicht erreichbar")

// ErrNichtInDerDNB heißt: Die DNB hat geantwortet und kennt die ISBN nicht.
var ErrNichtInDerDNB = errors.New("die DNB kennt diese ISBN nicht")

// ErrUngueltigeISBN heißt: Die Eingabe hat nicht die Form einer ISBN; gefragt wurde niemand.
var ErrUngueltigeISBN = errors.New("ungültiges ISBN-Format")

// SucheDNBNachISBN fragt nur die DNB, für den Vorschlag der Schlagworte im Formular des Titels
// und beim Nachbestellen (GET /api/schlagworte/dnb-vorschlag). Schlagworte liefert keine andere
// Quelle; SucheNachISBN fragt nach der DNB auch Google Books und OpenLibrary und lädt das Cover
// herunter, für einen Vorschlag unnötig, und das anonyme Kontingent von Google für den Tag teilt
// das Programm mit allen.
func (c *MetadatenClient) SucheDNBNachISBN(kontext context.Context, isbn string) (*MetadatenErgebnis, error) {
	saubereIsbn := isbnutil.OhneTrenner(isbn)
	if !validiereISBN(saubereIsbn) {
		return nil, ErrUngueltigeISBN
	}
	ergebnis, fehler := c.sucheDNB(kontext, saubereIsbn)
	switch {
	case errors.Is(fehler, errQuelleNichtErreichbar):
		return nil, fmt.Errorf("%w: %v", ErrDNBNichtErreichbar, fehler)
	case fehler != nil:
		// „nicht gefunden", 4xx oder unlesbare Antwort: Die DNB war da (wie in SucheNachISBN).
		return nil, fmt.Errorf("%w: %v", ErrNichtInDerDNB, fehler)
	}
	return ergebnis, nil
}

// SucheNachISBN iteriert der Reihe nach über verschiedene Buch-APIs (DNB, Google, OpenLibrary),
// bis für die gesuchte ISBN gültige Titel-/Autorendaten gefunden wurden.
func (c *MetadatenClient) SucheNachISBN(kontext context.Context, isbn string) (*MetadatenErgebnis, error) {
	var ergebnis *MetadatenErgebnis
	var fehler error

	// ISBN von Strichen und Leerzeichen befreien
	saubereIsbn := isbnutil.OhneTrenner(isbn)

	// SSRF und Argument Injection Schutz: Nur gültige ISBN-Zeichenfolgen zulassen
	// validiereISBN nimmt einen vorab übersetzten regulären Ausdruck, statt ihn bei jedem Aufruf
	// neu zu übersetzen.
	if !validiereISBN(saubereIsbn) {
		return nil, fmt.Errorf("ungültiges ISBN format: sicherheitsabbruch")
	}

	// Der Reihe nach: DNB (beste Quelle für deutsche Schulbücher), Google, OpenLibrary.
	// Nebenbei wird gezählt, ob überhaupt eine Quelle geantwortet hat: „alle nicht
	// erreichbar" ist ein Ausfall (Sentinel → 502 beim Aufrufer), kein Nicht-Treffer.
	quellen := []func(context.Context, string) (*MetadatenErgebnis, error){
		c.sucheDNB,
		c.sucheBeiGoogle,
		c.sucheOpenLibrary,
	}
	erreichbar := false
	var letzterAusfall error
	for _, quelle := range quellen {
		ergebnis, fehler = quelle(kontext, saubereIsbn)
		if fehler == nil && ergebnis != nil && (ergebnis.Titel != "" || ergebnis.Autor != "") {
			return c.beendeSuche(kontext, ergebnis, saubereIsbn), nil
		}
		if errors.Is(fehler, errQuelleNichtErreichbar) {
			letzterAusfall = fehler
			continue
		}
		// „nicht gefunden", leere Nutzlast oder unlesbare Antwort: Die Quelle war da.
		erreichbar = true
	}

	if !erreichbar {
		return nil, fmt.Errorf("%w (zuletzt: %v)", ErrKatalogdiensteNichtErreichbar, letzterAusfall)
	}
	return nil, fmt.Errorf("keine metadaten für ISBN gefunden")
}

// beendeSuche füllt das MetadatenErgebnis mit fehlenden Informationen auf:
// automatische Kategorisierung (Fach und Schulstufe) sowie das lokal gespeicherte Cover.
func (c *MetadatenClient) beendeSuche(kontext context.Context, ergebnis *MetadatenErgebnis, isbn string) *MetadatenErgebnis {
	ergebnis.CoverURL = c.aufloeseCover(kontext, ergebnis, isbn)

	ergebnis.Fach, ergebnis.JahrgangVon, ergebnis.JahrgangBis = automatischeKategorisierung(ergebnis.Titel, ergebnis.Untertitel)
	return ergebnis
}

// aufloeseCover probiert mehrere Cover-Quellen der Reihe nach durch und gibt den lokalen
// /uploads/-Pfad der ersten Quelle zurück, die tatsächlich ein dekodierbares Bild liefert.
// Schlägt jede Quelle fehl, wird "" zurückgegeben (= „kein Cover", der Titel bleibt erhalten).
//
// Wir verlassen uns bewusst nicht mehr auf einen HEAD-Verfügbarkeits-Check: Eine Bot-Schranke
// kann mit HTTP 200 + HTML antworten (siehe coverFetchUserAgent), wodurch ein HEAD-200
// fälschlich „Cover vorhanden" bedeutete, die Fallback-Quellen blockierte und am Ende eine
// HTML-Seite statt eines Bildes lieferte. Stattdessen laden wir jeden Kandidaten echt herunter;
// downloadAndSaveCoverLocally akzeptiert nur dekodierbare Bilder ausreichender Größe. Ein
// fehlgeschlagener Download verwirft hier nicht mehr ein bereits gefundenes Cover.
func (c *MetadatenClient) aufloeseCover(kontext context.Context, ergebnis *MetadatenErgebnis, isbn string) string {
	// DNB und OpenLibrary werden mit der Normalform gefragt, derselben Regel wie in der
	// Datenbank: Eine zehnstellige ISBN wird nur mit richtigem Prüfzeichen dreizehnstellig.
	nummer := isbnutil.Normalform(isbn)

	// Kandidaten in Prioritätsreihenfolge: DNB primär (beste Quelle für deutsche Schulbücher).
	kandidaten := make([]string, 0, 4)

	// 1. Bereits aus der Metadatensuche (Google/OpenLibrary) bekanntes Cover.
	if ergebnis.CoverURL != "" && strings.HasPrefix(ergebnis.CoverURL, "http") {
		kandidaten = append(kandidaten, ergebnis.CoverURL)
	}

	// 2. DNB MVB Cover (primär).
	kandidaten = append(kandidaten, fmt.Sprintf("https://portal.dnb.de/opac/mvb/cover?isbn=%s", url.QueryEscape(nummer)))

	// 3. Google Books — separat abfragen, falls die Metadaten von DNB kamen (DNB liefert keine Cover-URL im Datensatz).
	if gb, err := c.sucheBeiGoogle(kontext, isbn); err == nil && gb != nil && gb.CoverURL != "" {
		kandidaten = append(kandidaten, gb.CoverURL)
	}

	// 4. OpenLibrary — default=false erzwingt ein echtes 404 statt eines 1×1-Platzhalters.
	kandidaten = append(kandidaten, fmt.Sprintf("https://covers.openlibrary.org/b/isbn/%s-L.jpg?default=false", nummer))

	for _, kandidat := range kandidaten {
		if lokal := speichereCoverLokal(kontext, c.httpClient, kandidat, isbn); lokal != "" {
			return lokal
		}
	}
	return ""
}

// holeInhalt holt eine Adresse mit GET und liefert den Rumpf der Antwort.
func (c *MetadatenClient) holeInhalt(kontext context.Context, adresse string) ([]byte, error) {
	// Angefragt wird die NEU gebaute URL, nicht apiURL: Geprüft hätte url.Parse,
	// losgeschickt worden wäre der rohe String. Wo sich die beiden Parser über ein
	// Zeichen uneinig sind, geht die Anfrage an einen anderen Host als den geprüften
	// (Parsing Differential, gosec G704). Die Liste liegt in pkg/coverquelle und ist
	// bewusst enger als die Cover-Liste.
	sichereURL, ok := coverquelle.SichereURL(adresse, coverquelle.MetadatenHosts)
	if !ok {
		return nil, fmt.Errorf("SSRF Schutz: Hostname ist nicht in der Whitelist: %s", adresse)
	}

	// #nosec G107 - URL aus geprüften Teilen neu gebaut (Host aus der Allowlist-Konstante)
	anfrage, fehler := http.NewRequestWithContext(kontext, http.MethodGet, sichereURL, nil)
	if fehler != nil {
		return nil, fehler
	}
	anfrage.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	antwort, fehler := c.httpClient.Do(anfrage)
	if fehler != nil {
		// Transportfehler tragen das Sentinel: Die Quelle war nicht da — der Unterschied
		// zwischen „Netz weg" (502) und „Buch unbekannt" (404) entsteht genau hier.
		return nil, fmt.Errorf("%w: %v", errQuelleNichtErreichbar, fehler)
	}
	defer func() { _ = antwort.Body.Close() }() //nolint:errcheck
	if antwort.StatusCode >= http.StatusInternalServerError {
		// 5xx: Die Quelle ist kaputt, nicht leer.
		return nil, fmt.Errorf("%w: status %d", errQuelleNichtErreichbar, antwort.StatusCode)
	}
	if antwort.StatusCode != http.StatusOK {
		// 4xx: Die Quelle hat geantwortet — das zählt als erreichbar ohne Treffer.
		return nil, fmt.Errorf("status %d", antwort.StatusCode)
	}
	return io.ReadAll(io.LimitReader(antwort.Body, 2<<20)) // Max 2 MB
}
