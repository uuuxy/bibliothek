package inventur

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// holeInhalt trennt „die Quelle ist kaputt" von „die Quelle kennt das Buch nicht".
//
// Am Sentinel errQuelleNichtErreichbar hängt die Antwort nach draußen: 502 („Netz weg,
// später noch einmal") statt 404 („Buch unbekannt, von Hand erfassen"). Wer die beiden
// verwechselt, schickt die Bibliothek bei einem DNB-Ausfall ans Abtippen — oder lässt sie
// bei einem wirklich unbekannten Titel warten. (Zusammengeführt aus den Jules-PRs #606
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
// (Aus Jules-PR #610; ergänzt um die Prüfung, dass die Rückfallquellen wirklich gefragt
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
