package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// „Externe Cover nachladen" ersetzt die Adresse im Netz durch das heruntergeladene Bild. Ein
// Cover, das schon lokal liegt — auch ein von Hand hochgeladenes —, fasst der Lauf nicht an
// (Bugklasse „Maschine gegen Hand", cover_hand_gegen_maschine_pg_test.go). Am echten Postgres,
// die Katalogdienste sind nachgestellt: Google nennt den Titel, die DNB liefert das Bild.
func TestRetryExternalCovers_ErsetztNurAdressenImNetz(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	titel := func(name, isbn, cover string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_titel (titel, isbn, cover_url) VALUES ($1, $2, $3)
			ON CONFLICT (isbn) DO UPDATE SET titel = EXCLUDED.titel, cover_url = EXCLUDED.cover_url
			RETURNING id`, name, isbn, cover).Scan(&id); err != nil {
			t.Fatalf("Titel %q anlegen: %v", name, err)
		}
		t.Cleanup(func() {
			if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE id = $1`, id); err != nil {
				t.Logf("Aufräumen: %v", err)
			}
		})
		return id
	}
	cover := func(id string) string {
		t.Helper()
		var c string
		if err := pool.QueryRow(ctx, `SELECT coalesce(cover_url, '') FROM buecher_titel WHERE id = $1`, id).Scan(&c); err != nil {
			t.Fatalf("Cover lesen: %v", err)
		}
		return c
	}
	const handCover = "/uploads/covers/hand.webp"
	extern := titel("Cover-Retry extern", "9780000000002", "https://extern.example/cover.jpg")
	hand := titel("Cover-Retry Hand", "9780000000019", handCover)

	bild := image.NewRGBA(image.Rect(0, 0, 15, 15))
	for x := 0; x < 15; x++ {
		for y := 0; y < 15; y++ {
			bild.Set(x, y, color.RGBA{0, 0, 255, 255})
		}
	}
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, bild); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	antwort := func(status int, inhalt []byte, typ string) *http.Response {
		resp := &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewReader(inhalt)), Header: make(http.Header)}
		if typ != "" {
			resp.Header.Set("Content-Type", typ)
		}
		return resp
	}
	katalog := &mockTransportCover{roundTripFunc: func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.String(), "portal.dnb.de/opac/mvb/cover"):
			return antwort(http.StatusOK, pngBytes.Bytes(), "image/png"), nil
		case strings.Contains(req.URL.String(), "googleapis.com"):
			return antwort(http.StatusOK, []byte(`{"items":[{"volumeInfo":{"title":"Katalogtitel","authors":["Katalog"],"imageLinks":{"thumbnail":""}}}]}`), ""), nil
		}
		return antwort(http.StatusNotFound, nil, ""), nil
	}}
	handler := &APIHandler{repo: NewBookRepository(pool), metadaten: &MetadatenClient{httpClient: &http.Client{Transport: katalog}}}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/books/retry-covers",
		strings.NewReader(`{"ids":["`+extern+`","`+hand+`"]}`))
	w := httptest.NewRecorder()
	// Was der Lauf herunterlädt, kommt nach dem Test wieder weg — auch wenn er scheitert.
	t.Cleanup(func() {
		for _, isbn := range []string{"9780000000002", "9780000000019"} {
			bilder, err := filepath.Glob(filepath.Join("uploads", "cover_auto_"+isbn+"_*"))
			if err != nil {
				t.Errorf("Testbilder suchen: %v", err)
			}
			for _, bild := range bilder {
				if err := os.Remove(bild); err != nil {
					t.Errorf("heruntergeladenes Testbild aufräumen: %v", err)
				}
			}
		}
	})
	handler.handleRetryExternalCovers(w, req)
	neu := cover(extern)

	if w.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data map[string]int `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Antwort: %v (%s)", err, w.Body.String())
	}
	if resp.Data["retried"] != 2 || resp.Data["updated"] != 1 || resp.Data["skipped"] != 1 || resp.Data["failed"] != 0 {
		t.Errorf("Zahlen %v — erwartet 2 versucht, 1 aktualisiert, 1 übersprungen, 0 gescheitert", resp.Data)
	}
	if !strings.HasPrefix(neu, "/uploads/cover_auto_") {
		t.Errorf("das externe Cover steht weiter als %q da — erwartet den lokalen Pfad des heruntergeladenen Bilds", neu)
	}
	if got := cover(hand); got != handCover {
		t.Errorf("das von Hand hochgeladene Cover wurde ersetzt: %q", got)
	}
	// Der Lauf verspricht ein Bild: Titel und Autor aus dem Katalog überschreiben nichts.
	var name string
	if err := pool.QueryRow(ctx, `SELECT titel FROM buecher_titel WHERE id = $1`, extern).Scan(&name); err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}
	if name != "Cover-Retry extern" {
		t.Errorf("der Lauf hat den Titel überschrieben: %q", name)
	}
}
