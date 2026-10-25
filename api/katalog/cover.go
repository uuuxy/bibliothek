package katalog

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/coverquelle"
	"bibliothek/pkg/httpresp"
	"bibliothek/pkg/ratelimit"
	"bibliothek/pkg/safehttp"
)

// coverDownloadsProSekunde begrenzt nur den teuren Zweig dieses Endpunkts: den Fehltreffer im
// Cache, der einen fremden Download samt vollständiger Bilddekodierung auslöst. Treffer im Cache
// bleiben ungebremst; sie lesen nur eine Datei, und ein Katalogaufruf lädt Dutzende davon
// gleichzeitig (deshalb steht der Pfad in der Ausnahmeliste der globalen Bremse,
// api/anfrage_bremse.go).
//
// Der Endpunkt ist ohne Anmeldung erreichbar und löst eine ausgehende Verbindung und ein
// image.Decode aus; ungebremst hieße das je Aufruf einen Download und eine Dekodierung, beliebig
// oft nebeneinander. 30 je Sekunde und Adresse sind so bemessen, dass eine Katalogseite beim
// ersten Aufruf ihre Kacheln in einem Zug lädt.
const coverDownloadsProSekunde = 30

// coverCacheVerzeichnis ist die Ablage der umgewandelten Cover im Betrieb.
const coverCacheVerzeichnis = "uploads/covers"

// istCoverCacheSchluessel prüft den vom Aufrufer gewählten Namen, bevor daraus ein Dateiname
// wird. Zulässig sind Ziffern, Trennstriche und ein abschließendes X (Prüfziffer der ISBN-10),
// genau die Zeichen, die eine ISBN oder EAN haben kann.
//
// Nicht über isbnutil.OhneTrenner geprüft, obwohl das naheliegt: Es entfernt auch Leerzeichen,
// weil es ISBNs vergleichbar machen soll; "9783161 84100" wäre danach gültig. Ob zwei
// Zeichenketten dieselbe ISBN sind und ob eine ein Dateiname sein darf, sind zwei Fragen mit
// verschiedenen Antworten.
//
// Das ist Hygiene, keine Mengenbegrenzung: Gegen das Vollschreiben der Platte hilft die
// Download-Bremse oben. Die Prüfung hält fern, was erkennbar keine ISBN ist, und hält die
// Dateinamen vorhersagbar.
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

// transparentesGIF ist ein durchsichtiges Bild aus einem Pixel. Die Tür liefert es bei jedem
// Fehler aus: Am anderen Ende hängt ein <img>, und ein Fehlercode erzeugte dort nur einen roten
// Eintrag in der Konsole des Browsers.
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

// baueSichereCoverURL prüft die Adresse eines Covers gegen die Liste erlaubter Hosts und baut
// sie aus geprüften Teilen neu auf. Liste und Neuaufbau liegen in pkg/coverquelle, weil die
// Katalogdienste dieselbe Prüfung brauchen (Cover-Download und Abruf der Metadaten).
func baueSichereCoverURL(adresse string) (string, bool) {
	return coverquelle.SichereURL(adresse, coverquelle.BildHosts)
}

// coverHTTPClient lädt Cover mit dem Schutz, der http.DefaultClient fehlt: Verbindungen zu
// nicht öffentlichen Adressen lehnt er beim Verbindungsaufbau ab, nach der DNS-Auflösung und
// damit auch für jede Weiterleitung (OpenLibrary leitet auf archive.org um) und bei
// DNS-Rebinding. Für den ganzen Abruf gilt eine feste Zeitgrenze von 20 Sekunden. Die
// Umsetzung liegt in pkg/safehttp, weil die Katalogdienste für ihre Abrufe denselben Schutz
// brauchen.
var coverHTTPClient = safehttp.NewClient(20 * time.Second)

// coverAbruf ist, womit die Tür ein Cover holt, erkennt und ablegt. Der Nullwert ist der
// Betrieb; ein Test setzt einen Client gegen eine Attrappe, eine Ablage im Wegwerf-Ordner oder
// ein eigenes Ersatzbild, ohne Zustand anderer Tests zu berühren.
type coverAbruf struct {
	client        *http.Client      // nil: coverHTTPClient
	verzeichnis   string            // leer: coverCacheVerzeichnis
	istErsatzbild func([]byte) bool // nil: coverquelle.IstErsatzbild
}

func (c coverAbruf) httpClient() *http.Client {
	if c.client != nil {
		return c.client
	}
	return coverHTTPClient
}

func (c coverAbruf) ablage() string {
	if c.verzeichnis != "" {
		return c.verzeichnis
	}
	return coverCacheVerzeichnis
}

func (c coverAbruf) ersatzbild(antwort []byte) bool {
	if c.istErsatzbild != nil {
		return c.istErsatzbild(antwort)
	}
	return coverquelle.IstErsatzbild(antwort)
}

// holeUndKonvertiere holt das Cover nach den Regeln aus coverquelle.Hole, die auch beim Anlegen
// eines Titels gelten, und legt es im Cache ab.
func (c coverAbruf) holeUndKonvertiere(ctx context.Context, root *os.Root, adresse, dateiname string) error {
	fertig, err := coverquelle.Hole(ctx, c.httpClient(), adresse, c.ersatzbild)
	if err != nil {
		return err
	}
	return schreibeCoverDatei(root, dateiname, fertig)
}

// schreibeCoverDatei legt das fertige Cover im Cache ab. Eine angefangene Datei wird bei einem
// Fehler wieder entfernt.
func schreibeCoverDatei(root *os.Root, dateiname string, fertig []byte) error {
	// 0600 wie in pkg/coverablage/coverablage.go, nicht 0666: Im Container fängt die umask das meist
	// ab, aber „meist" ist bei Dateirechten keine Zusage. Nur dieser Prozess muss die Dateien je
	// anfassen; ausgeliefert werden sie über den Server, nicht über das Dateisystem.
	datei, err := root.OpenFile(dateiname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	_, err = datei.Write(fertig)
	// Ein gescheitertes Close kann eine abgeschnittene Datei hinterlassen; es zählt wie ein
	// Fehler beim Schreiben.
	if cerr := datei.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		if rerr := root.Remove(dateiname); rerr != nil { // die angefangene Datei entfernen
			slog.Warn("cover cache: cleanup failed", "datei", dateiname, "fehler", rerr)
		}
		return err
	}
	return nil
}

// Cover liefert das Cover eines Titels als WebP aus dem Cache; fehlt es, lädt die Tür es von
// einer erlaubten Adresse und wandelt es um. Bei jedem Fehler kommt transparentesGIF. Die
// Bremse der Fehltreffer gilt für alle Anfragen an die gelieferte Tür; der Router baut sie
// einmal.
func (t *Tueren) Cover() http.HandlerFunc {
	bremse := ratelimit.New(coverDownloadsProSekunde)
	return func(w http.ResponseWriter, r *http.Request) {
		t.liefereCover(w, r, bremse)
	}
}

// liefereCover liefert das Cover zur ISBN aus dem Cache oder lädt es; bei jedem Fehler
// transparentesGIF (siehe Cover). bremse begrenzt die Fehltreffer je Adresse.
func (t *Tueren) liefereCover(w http.ResponseWriter, r *http.Request, bremse *ratelimit.Bremse) {
	isbn := r.URL.Query().Get("isbn")
	adresse := r.URL.Query().Get("url")

	if isbn == "" || adresse == "" {
		liefereTransparentesGIF(w)
		return
	}

	// Der Name wird gleich Teil eines Dateinamens: vor allem anderen prüfen.
	if !istCoverCacheSchluessel(isbn) {
		liefereTransparentesGIF(w)
		return
	}

	// Gegen SSRF: die Adresse aus geprüften Teilen neu aufbauen.
	sichereURL, ok := baueSichereCoverURL(adresse)
	if !ok {
		liefereTransparentesGIF(w)
		return
	}

	verzeichnis := t.cover.ablage()
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

	// Ein Pfad mit Verzeichnissen fällt hier auf, bevor etwas geladen wird; root.OpenFile
	// hielte ihn später ohnehin auf.
	if filepath.Base(isbn) != isbn {
		liefereTransparentesGIF(w)
		return
	}

	// ISBN und Hash der geprüften Adresse: Zwei Adressen teilen sich nie eine Datei, und wer
	// zuerst ruft, bestimmt nicht, was andere unter dieser ISBN sehen.
	dateiname := coverDateiname(isbn, sichereURL)

	// Liegt das Cover im Cache, von dort ausliefern.
	if _, err := root.Stat(dateiname); err == nil {
		liefereCoverAusCache(w, r, root, dateiname)
		return
	}

	// Ab hier wird es teuer: fremder Download plus vollständige Dekodierung. Nur dieser
	// Zweig ist gebremst, und er antwortet im Grenzfall mit dem transparenten GIF statt mit
	// 429, weil am anderen Ende ein <img> hängt: Ein Fehlercode erzeugt dort nur einen
	// roten Konsoleneintrag, das Bild fehlt so oder so. Der nächste Aufruf holt es nach.
	if !bremse.LaesstDurch(clientip.AusAnfrage(r)) {
		liefereTransparentesGIF(w)
		return
	}

	// Nur Adressen, die der Server für diese ISBN selbst kennt oder herleiten kann
	// (cover_quelle_bindung.go); sonst wäre der Endpunkt ein offener Proxy für alle
	// Bilder der erlaubten Hosts und der Cache ein Plattenfüller.
	if !t.istCoverQuelleErlaubt(r, isbn, sichereURL) {
		liefereTransparentesGIF(w)
		return
	}

	// Fehlt es, laden und umwandeln.
	if err := t.cover.holeUndKonvertiere(r.Context(), root, sichereURL, dateiname); err != nil {
		liefereTransparentesGIF(w)
		return
	}

	// Die eben umgewandelte Datei ausliefern.
	if _, err := root.Stat(dateiname); err == nil {
		liefereCoverAusCache(w, r, root, dateiname)
	} else {
		liefereTransparentesGIF(w)
	}
}
