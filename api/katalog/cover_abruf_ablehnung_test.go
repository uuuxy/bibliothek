package katalog

import (
	"context"
	"crypto/tls"
	"image/color"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Was der Cover-Abruf abweist, landet nicht im Cache: eine Antwort ohne Status 200, eine Seite
// statt eines Bilds und Bytes, die kein Bild sind. Welche Antworten coverquelle.Hole abweist,
// prüft pkg/coverquelle; hier zählt, dass danach keine Datei liegt.
func TestHoleUndKonvertiereCover_AbweisungenLegenNichtsAb(t *testing.T) {
	bild := pngEinfarbig(t, color.RGBA{0, 0, 255, 255})
	schreibe := func(w http.ResponseWriter, typ string, inhalt []byte) {
		w.Header().Set("Content-Type", typ)
		if _, err := w.Write(inhalt); err != nil {
			t.Errorf("Stub-Antwort schreiben: %v", err)
		}
	}
	stub := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/gestoert":
			w.WriteHeader(http.StatusBadGateway)
		case "/seite":
			schreibe(w, "text/html; charset=utf-8", bild)
		case "/kein-bild":
			schreibe(w, "image/png", []byte("kein Bild"))
		default:
			schreibe(w, "image/png", bild)
		}
	}))
	t.Cleanup(stub.Close)
	abruf := coverAbruf{client: &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // Wegwerf-Zertifikat des Teststubs
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("tcp", stub.Listener.Addr().String())
		},
	}}}

	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Errorf("Cache-Verzeichnis schließen: %v", err)
		}
	})

	for _, pfad := range []string{"/gestoert", "/seite", "/kein-bild"} {
		if err := abruf.holeUndKonvertiere(context.Background(), root, "https://covers.openlibrary.org"+pfad, "cover.webp"); err == nil {
			t.Errorf("%s: holeUndKonvertiere(…) nimmt die Antwort an, erwartet eine Abweisung", pfad)
		}
		if _, statErr := root.Stat("cover.webp"); statErr == nil {
			t.Errorf("%s: die abgewiesene Antwort liegt als Datei im Cache", pfad)
		}
	}

	if err := abruf.holeUndKonvertiere(context.Background(), root, "https://covers.openlibrary.org/bild", "cover.webp"); err != nil {
		t.Fatalf("ein Bild wird abgewiesen: %v", err)
	}
	if info, err := root.Stat("cover.webp"); err != nil || info.Size() == 0 {
		t.Errorf("das Cover liegt nicht im Cache: %v", err)
	}
}
