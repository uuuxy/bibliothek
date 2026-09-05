package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v4"
)

// Der Fund vom 05.09.2026, am Handler nachgestellt: Ohne Bindung bestimmte der erste
// Aufrufer (ohne Anmeldung) das Bild, das unter einer ISBN ausgeliefert wird, und jeder
// ISBN-artige Schlüssel legte eine Datei an. Mit dem alten image_caching.go lieferte
// derselbe Aufbau das Angreiferbild byte-identisch beim zweiten Aufruf und sechs Dateien
// für sechs Schlüssel (rot gesehen, Probe vor dem Umbau).

const testISBN = "9783161484100"

func pngEinfarbig(t *testing.T, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// coverStub biegt den Cover-Client auf einen lokalen TLS-Server um, der je Pfad ein
// anderes Bild liefert, und zählt die Downloads.
func coverStub(t *testing.T) *int {
	t.Helper()
	downloads := 0
	rot := pngEinfarbig(t, color.RGBA{255, 0, 0, 255})
	blau := pngEinfarbig(t, color.RGBA{0, 0, 255, 255})
	stub := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloads++
		w.Header().Set("Content-Type", "image/png")
		bild := blau
		if r.URL.Path == "/b/id/FREMD-L.jpg" {
			bild = rot
		}
		if _, err := w.Write(bild); err != nil {
			t.Errorf("Stub-Antwort schreiben: %v", err)
		}
	}))
	t.Cleanup(stub.Close)

	orig := coverHTTPClient
	coverHTTPClient = &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // Wegwerf-Zertifikat des Teststubs
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("tcp", stub.Listener.Addr().String())
		},
	}}
	t.Cleanup(func() { coverHTTPClient = orig })

	origDir := coverCacheVerzeichnis
	coverCacheVerzeichnis = t.TempDir() + "/covers"
	t.Cleanup(func() { coverCacheVerzeichnis = origDir })
	return &downloads
}

func neuerMockPool(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mock.Close)
	return mock
}

func holeCover(s *Server, isbn, adresse string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/images/cover?isbn="+isbn+"&url="+url.QueryEscape(adresse), nil)
	req.RemoteAddr = "203.0.113.5:1234"
	s.serveCoverImage(rr, req)
	return rr
}

func erwarteKatalog(mock pgxmock.PgxPoolIface, coverURLs ...string) {
	rows := pgxmock.NewRows([]string{"cover_url"})
	for _, u := range coverURLs {
		rows.AddRow(u)
	}
	mock.ExpectQuery("FROM buecher_titel WHERE regexp_replace").WithArgs(testISBN).WillReturnRows(rows)
}

func TestCoverProxy_FremdeAdresseFuerKatalogISBNWirdNichtGeladen(t *testing.T) {
	downloads := coverStub(t)
	mock := neuerMockPool(t)
	s := &Server{DB: &db.Database{Pool: mock}}

	echt := "https://covers.openlibrary.org/b/isbn/" + testISBN + "-L.jpg"
	erwarteKatalog(mock, echt)
	rr := holeCover(s, testISBN, "https://covers.openlibrary.org/b/id/FREMD-L.jpg")

	if !bytes.Equal(rr.Body.Bytes(), coverFallbackGIF) {
		t.Fatalf("fremde Adresse wurde ausgeliefert (%d Bytes) — want Fallback-GIF", rr.Body.Len())
	}
	if *downloads != 0 {
		t.Fatalf("Downloads = %d; want 0 — der Server darf fremde Bilder gar nicht erst holen", *downloads)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCoverProxy_GespeicherteAdresseWirdGeladenUndGecacht(t *testing.T) {
	downloads := coverStub(t)
	mock := neuerMockPool(t)
	s := &Server{DB: &db.Database{Pool: mock}}

	// Gespeichert ist die http-Form mit Trennstrich-ISBN — der Vergleich läuft über die
	// geprüfte Form, nicht den rohen Text.
	erwarteKatalog(mock, "http://covers.openlibrary.org/b/isbn/"+testISBN+"-L.jpg")
	erster := holeCover(s, "978-3-16-148410-0", "https://covers.openlibrary.org/b/isbn/"+testISBN+"-L.jpg")
	zweiter := holeCover(s, "978-3-16-148410-0", "https://covers.openlibrary.org/b/isbn/"+testISBN+"-L.jpg")

	if ct := erster.Header().Get("Content-Type"); ct != "image/webp" {
		t.Fatalf("Content-Type = %q; want image/webp (Body %d Bytes)", ct, erster.Body.Len())
	}
	if *downloads != 1 {
		t.Fatalf("Downloads = %d; want 1 — der zweite Aufruf kommt aus dem Cache ohne DB", *downloads)
	}
	if !bytes.Equal(erster.Body.Bytes(), zweiter.Body.Bytes()) {
		t.Fatal("Cache-Treffer liefert ein anderes Bild als der Download")
	}
}

// Zwei erlaubte Adressen derselben ISBN teilen sich keine Datei: Wer die eine zuerst
// holt, bestimmt nicht, was unter der anderen liegt.
func TestCoverProxy_JedeAdresseHatIhreEigeneDatei(t *testing.T) {
	coverStub(t)
	mock := neuerMockPool(t)
	s := &Server{DB: &db.Database{Pool: mock}}

	kandidaten := coverKandidatenFuerISBN(testISBN)
	erwarteKatalog(mock, "") // ISBN im Katalog, aber ohne gespeichertes Cover
	holeCover(s, testISBN, kandidaten[0])
	erwarteKatalog(mock, "")
	holeCover(s, testISBN, kandidaten[1])

	eintraege, err := os.ReadDir(coverCacheVerzeichnis)
	if err != nil {
		t.Fatal(err)
	}
	if len(eintraege) != 2 {
		t.Fatalf("Dateien im Cache = %d; want 2", len(eintraege))
	}
}

// Ohne Anmeldung nur Katalog-ISBNs: Sonst wäre der Endpunkt ein Plattenfüller — jede
// erfundene Nummer eine Datei.
func TestCoverProxy_UnbekannteISBNOhneAnmeldungWirdNichtGeladen(t *testing.T) {
	downloads := coverStub(t)
	mock := neuerMockPool(t)
	s := &Server{DB: &db.Database{Pool: mock}}

	mock.ExpectQuery("FROM buecher_titel WHERE regexp_replace").WithArgs("9780000000001").
		WillReturnRows(pgxmock.NewRows([]string{"cover_url"}))
	rr := holeCover(s, "9780000000001", coverKandidatenFuerISBN("9780000000001")[1])

	if !bytes.Equal(rr.Body.Bytes(), coverFallbackGIF) || *downloads != 0 {
		t.Fatalf("unbekannte ISBN wurde geladen (Downloads %d, %d Bytes)", *downloads, rr.Body.Len())
	}
}

// Ohne erreichbare Datenbank wird nichts geholt — im Zweifel kein Download.
func TestCoverProxy_OhneDatenbankKeinDownload(t *testing.T) {
	downloads := coverStub(t)
	s := &Server{}
	rr := holeCover(s, testISBN, coverKandidatenFuerISBN(testISBN)[1])
	if !bytes.Equal(rr.Body.Bytes(), coverFallbackGIF) || *downloads != 0 {
		t.Fatalf("Download ohne Datenbank (Downloads %d)", *downloads)
	}
}

// Die Kandidaten-Adressen baut der Server GENAUSO wie coverKandidaten im Frontend —
// sonst lehnt er still ab, was die Oberfläche anfragt, und jede Kachel ohne
// gespeichertes Cover bleibt leer. Der Test liest die Frontend-Datei, weil die beiden
// Muster nirgends sonst zusammenkommen.
func TestCoverKandidaten_MusterGleichWieImFrontend(t *testing.T) {
	quelle, err := os.ReadFile(filepath.Join("..", "frontend", "src", "lib", "utils", "coverSrc.js"))
	if err != nil {
		t.Fatalf("coverSrc.js nicht lesbar: %v", err)
	}
	js := string(quelle)
	for _, k := range coverKandidatenFuerISBN("${sauber}") {
		if !strings.Contains(js, k) {
			t.Errorf("Frontend-Muster fehlt oder weicht ab:\n  Server: %s", k)
		}
	}
}
