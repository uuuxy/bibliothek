package coverquelle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"strings"

	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/coverablage"
	"bibliothek/pkg/imageutil"
)

// kennung ist der User-Agent, mit dem das Programm sich bei DNB, Google und OpenLibrary als
// Programm ausweist. Die Schnittstelle der DNB für Cover (portal.dnb.de/opac/mvb/cover) liegt
// hinter der Bot-Schranke "Anubis", die gerade bei User-Agents anschlägt, die wie ein Browser
// aussehen: Solche Anfragen bekommen HTTP 200 mit einer Aufgabe in HTML statt des Bildes. Ein
// schlichter Name, der kein Browser ist, kommt durch; deshalb nicht auf den eines Browsers
// „aufrüsten".
const kennung = "Inventur/1.0"

// maxBytes begrenzt, wie viel von einer fremden Antwort in den Speicher gelesen wird. Ohne die
// Grenze bestimmte der fremde Server, wie viel Speicher ein Abruf kostet; 10 MB sind für ein
// Buchcover großzügig.
const maxBytes = 10 << 20

// mindestKante trennt Cover von Platzhaltern: Kleiner liefern Quellen nur ein Pixel oder ein
// Symbol, wenn sie zur ISBN kein Bild haben.
const mindestKante = 10

// webpQualitaet ist die Qualität, in der ein geholtes Cover abgelegt wird.
const webpQualitaet = 80

// leeresOpenLibraryCover ist die Adresse, die für eine leere ISBN entsteht. OpenLibrary liefert
// dort einen Platzhalter; die Anfrage wird gar nicht erst gestellt.
const leeresOpenLibraryCover = "https://covers.openlibrary.org/b/isbn/-L.jpg"

// ErrKeinCover heißt: Die Quelle hat zu dieser Adresse kein Cover. Das ist der gewöhnliche Fall,
// wenn eine Quelle einen Titel nicht kennt; alle anderen Fehler sind Störungen.
var ErrKeinCover = errors.New("die Quelle hat kein Cover")

// Hole lädt ein Cover von einem erlaubten Host und liefert es als WebP, höchstens
// coverablage.MaxBreite × coverablage.MaxHoehe groß. Jede Stelle, die ein Cover aus dem Netz
// holt, nimmt diese Funktion, damit für Anzeige und Anlegen eines Titels dieselben Regeln gelten.
// istErsatzbild erkennt die Ersatzbilder der Quellen, im Betrieb IstErsatzbild.
func Hole(ctx context.Context, client *http.Client, rohURL string, istErsatzbild func([]byte) bool) ([]byte, error) {
	roh, err := holeBild(ctx, client, rohURL, istErsatzbild)
	if err != nil {
		return nil, err
	}
	bild, _, err := image.Decode(bytes.NewReader(roh))
	if err != nil {
		return nil, fmt.Errorf("cover: Bild nicht lesbar: %w", err)
	}
	if rahmen := bild.Bounds(); rahmen.Dx() < mindestKante || rahmen.Dy() < mindestKante {
		return nil, fmt.Errorf("cover: %d×%d Pixel sind ein Platzhalter: %w", rahmen.Dx(), rahmen.Dy(), ErrKeinCover)
	}
	return imageutil.ErzeugeWebP(imageutil.Verkleinere(bild, coverablage.MaxBreite, coverablage.MaxHoehe), webpQualitaet)
}

// holeBild holt die Antwort der Quelle und weist ab, was kein Cover ist, bevor irgendetwas
// dekodiert wird.
func holeBild(ctx context.Context, client *http.Client, rohURL string, istErsatzbild func([]byte) bool) ([]byte, error) {
	if rohURL == "" || rohURL == leeresOpenLibraryCover {
		return nil, fmt.Errorf("cover: leere Adresse: %w", ErrKeinCover)
	}
	// Angefragt wird die neu gebaute Adresse, nicht die Eingabe: Wo sich zwei Parser über ein
	// Zeichen uneinig sind, ginge die Anfrage sonst an einen anderen Host als den geprüften.
	sichereURL, ok := SichereURL(rohURL, BildHosts)
	if !ok {
		return nil, errors.New("cover: Adresse nicht von einem erlaubten Host")
	}

	// #nosec G107 - URL aus geprüften Teilen neu gebaut (Host aus der Liste BildHosts)
	anfrage, err := http.NewRequestWithContext(ctx, http.MethodGet, sichereURL, nil)
	if err != nil {
		return nil, err
	}
	anfrage.Header.Set("User-Agent", kennung)
	anfrage.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,*/*;q=0.8")

	antwort, err := client.Do(anfrage)
	if err != nil {
		return nil, err
	}
	defer closeutil.Schliesse(antwort.Body, "cover download")
	if antwort.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("cover: Status 404: %w", ErrKeinCover)
	}
	if antwort.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cover: unerwarteter Status %d", antwort.StatusCode)
	}

	// Schranken gegen Roboter (DNB/Anubis) liefern bei unerwartetem User-Agent HTTP 200 mit
	// einer Prüfseite in HTML.
	if typ := antwort.Header.Get("Content-Type"); strings.Contains(typ, "html") || strings.Contains(typ, "text/") || strings.Contains(typ, "json") {
		return nil, fmt.Errorf("cover: Nicht-Bild-Antwort (%s)", typ)
	}

	// Erst begrenzt einlesen, dann den Kopf des Bilds prüfen, dann dekodieren, weil jeder Schritt
	// den nächsten teurer macht. Die Grenze der Leitung begrenzt den Speicher nicht: Ein PNG mit
	// 30000×30000 Pixeln sind wenige hundert KB und rund 3,6 GB nach dem Dekodieren.
	// PruefeBildgroesse liest nur den Kopf und legt keine Pixeldaten an.
	roh, err := io.ReadAll(io.LimitReader(antwort.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(roh) > maxBytes {
		return nil, fmt.Errorf("cover: Antwort überschreitet %d MB", maxBytes>>20)
	}
	// Ein Ersatzbild der Quelle ist kein Cover: Der Aufrufer fragt dann die nächste Quelle oder
	// zeigt die Initiale des Titels.
	if istErsatzbild(roh) {
		return nil, fmt.Errorf("cover: Ersatzbild der Quelle: %w", ErrKeinCover)
	}
	if err := imageutil.PruefeBildgroesse(roh); err != nil {
		return nil, err
	}
	return roh, nil
}
