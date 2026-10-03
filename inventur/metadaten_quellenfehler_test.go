package inventur

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// holeInhalt trennt „die Quelle ist kaputt" von „die Quelle kennt das Buch nicht".
//
// Am Sentinel errQuelleNichtErreichbar hängt die Antwort nach draußen: 502 („Netz weg,
// später noch einmal") statt 404 („Buch unbekannt, von Hand erfassen"). Wer die beiden
// verwechselt, schickt die Bibliothek bei einem DNB-Ausfall ans Abtippen — oder lässt sie
// bei einem wirklich unbekannten Titel warten. (Zusammengeführt aus den PRs #606
// und #608; #608 prüfte nur den Text, nicht, dass der 4xx das Sentinel NICHT trägt.)
func TestHoleInhalt_KaputteQuelleIstNichtLeer(t *testing.T) {
	antwort := func(status int) func(*http.Request) (*http.Response, error) {
		return func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewBufferString("")), Header: make(http.Header)}, nil
		}
	}
	faelle := []struct {
		name          string
		roundTrip     func(*http.Request) (*http.Response, error)
		nichtErreicht bool
		teil          string
	}{
		{"5xx: Quelle kaputt", antwort(http.StatusServiceUnavailable), true, "status 503"},
		{"4xx: Quelle hat geantwortet", antwort(http.StatusNotFound), false, "status 404"},
		{"Transportfehler", func(*http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		}, true, "connection refused"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			client := &MetadatenClient{httpClient: &http.Client{Transport: &mockTransport{roundTripFunc: f.roundTrip}}}
			_, err := client.holeInhalt(context.Background(), "https://services.dnb.de/sru/dnb")
			if err == nil {
				t.Fatal("kein Fehler")
			}
			if got := errors.Is(err, errQuelleNichtErreichbar); got != f.nichtErreicht {
				t.Errorf("errQuelleNichtErreichbar = %v, erwartet %v: %v", got, f.nichtErreicht, err)
			}
			if !strings.Contains(err.Error(), f.teil) {
				t.Errorf("Meldung nennt %q nicht: %v", f.teil, err)
			}
		})
	}
}

// aufloeseCover fragt die Quellen der Reihe nach; eine scheiternde hält die nächste nicht
// auf, und scheitern alle, bleibt das Cover leer — der Titel selbst wird trotzdem angelegt.
// (Aus PR #610; ergänzt um die Prüfung, dass die Rückfallquellen wirklich gefragt
// wurden — sonst wäre auch ein Abbruch nach der ersten Quelle grün.)
func TestAufloeseCover_AlleQuellenScheitern(t *testing.T) {
	var gefragt []string
	client := NeuerMetadatenClient()
	client.SetzeHTTPClientFuerTest(&http.Client{Transport: &mockTransport{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			gefragt = append(gefragt, req.URL.Host)
			return nil, http.ErrServerClosed
		},
	}})

	cover := client.aufloeseCover(context.Background(), &MetadatenErgebnis{}, "9783141011540")

	if cover != "" {
		t.Errorf("Cover %q, obwohl keine Quelle geantwortet hat", cover)
	}
	for _, host := range []string{"portal.dnb.de", "covers.openlibrary.org"} {
		if !strings.Contains(strings.Join(gefragt, " "), host) {
			t.Errorf("%s wurde nicht gefragt — eine scheiternde Quelle hat die Rückfallquellen verdeckt (gefragt: %v)", host, gefragt)
		}
	}
}

// Die Cover-Suche fragt mit der Normalform der Nummer (isbnutil.Normalform), der einen Regel
// für die Länge einer ISBN. Eine zehnstellige mit richtigem Prüfzeichen geht dreizehnstellig
// hinaus; eine mit falschem bleibt, wie sie ist — gerechnet führte sie auf die ISBN eines
// anderen Buchs, und dessen Cover läge dann am Titel. 3499500252 steht so am Testserver.
func TestAufloeseCover_FragtMitDerNormalform(t *testing.T) {
	faelle := []struct{ name, isbn, gefragt string }{
		{"zehnstellig, Prüfzeichen stimmt", "3866801920", "9783866801929"},
		{"zehnstellig, Prüfzeichen falsch", "3499500252", "3499500252"},
		{"dreizehnstellig", "9783141011540", "9783141011540"},
		{"zwölfstellig, keine ISBN", "012345678905", "012345678905"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			var adressen []string
			client := NeuerMetadatenClient()
			client.SetzeHTTPClientFuerTest(&http.Client{Transport: &mockTransport{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					adressen = append(adressen, req.URL.String())
					return nil, http.ErrServerClosed
				},
			}})

			client.aufloeseCover(context.Background(), &MetadatenErgebnis{}, f.isbn)

			for _, erwartet := range []string{
				"https://portal.dnb.de/opac/mvb/cover?isbn=" + f.gefragt,
				"https://covers.openlibrary.org/b/isbn/" + f.gefragt + "-L.jpg?default=false",
			} {
				if !slices.Contains(adressen, erwartet) {
					t.Errorf("%s wurde nicht gefragt (gefragt: %v)", erwartet, adressen)
				}
			}
		})
	}
}
