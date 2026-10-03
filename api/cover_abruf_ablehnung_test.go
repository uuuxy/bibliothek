package api

import (
	"context"
	"image/color"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Was der Cover-Abruf abweist, landet nicht im Cache: eine Antwort ohne Status 200, eine Seite
// statt eines Bilds, eine Antwort über der Größengrenze und Bytes, die kein Bild sind.
func TestHoleUndKonvertiereCover_AbweisungenLegenNichtsAb(t *testing.T) {
	bild := pngEinfarbig(t, color.RGBA{0, 0, 255, 255})
	schreibe := func(w http.ResponseWriter, typ string, inhalt []byte) {
		w.Header().Set("Content-Type", typ)
		if _, err := w.Write(inhalt); err != nil {
			t.Errorf("Stub-Antwort schreiben: %v", err)
		}
	}
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/gestoert":
			w.WriteHeader(http.StatusBadGateway)
		case "/seite":
			schreibe(w, "text/html; charset=utf-8", bild)
		case "/zu-gross":
			schreibe(w, "image/png", make([]byte, maxCoverBytes+1))
		case "/kein-bild":
			schreibe(w, "image/png", []byte("kein Bild"))
		default:
			schreibe(w, "image/png", bild)
		}
	}))
	t.Cleanup(stub.Close)
	orig := coverHTTPClient
	coverHTTPClient = stub.Client()
	t.Cleanup(func() { coverHTTPClient = orig })

	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Errorf("Cache-Verzeichnis schließen: %v", err)
		}
	})

	for _, f := range []struct{ pfad, meldung string }{
		{"/gestoert", "unerwarteter Status 502"},
		{"/seite", "Nicht-Bild-Antwort"},
		{"/zu-gross", "überschreitet 10 MB"},
		{"/kein-bild", "Bild-Header"},
	} {
		err := holeUndKonvertiereCover(context.Background(), root, stub.URL+f.pfad, "cover.webp")
		if err == nil || !strings.Contains(err.Error(), f.meldung) {
			t.Errorf("%s: Fehler %v, erwartet eine Abweisung mit %q", f.pfad, err, f.meldung)
		}
		if _, statErr := root.Stat("cover.webp"); statErr == nil {
			t.Errorf("%s: die abgewiesene Antwort liegt als Datei im Cache", f.pfad)
		}
	}

	if err := holeUndKonvertiereCover(context.Background(), root, stub.URL+"/bild", "cover.webp"); err != nil {
		t.Fatalf("ein Bild wird abgewiesen: %v", err)
	}
	if info, err := root.Stat("cover.webp"); err != nil || info.Size() == 0 {
		t.Errorf("das Cover liegt nicht im Cache: %v", err)
	}
}
