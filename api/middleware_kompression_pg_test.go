package api

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/sse"
)

// kompressionHerkunft ist die erlaubte fremde Herkunft dieses Tests: Mit ihr setzt die
// CORS-Middleware ihren Vary-Kopf.
const kompressionHerkunft = "https://bibliothek.example"

// Die Kompression am Router des Betriebs: Was der Katalog und die Oberfläche holen, geht
// gepackt hinaus, alles andere unverändert. Gemessen wird an den echten Routen, damit
// kein Mount und kein Glied der Kette daran vorbeiführt.
func TestKompression_AmRouterDesBetriebs(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	t.Chdir(t.TempDir())
	t.Setenv("ALLOWED_ORIGIN", kompressionHerkunft)

	dateien := map[string][]byte{
		"frontend/dist/index.html":    []byte("<!doctype html><html><body>" + strings.Repeat("<p>Katalog</p>", 120) + "</body></html>"),
		"frontend/dist/assets/app.js": []byte(strings.Repeat("console.log('katalog');\n", 120)),
		// Trägt den Pfad des Live-Stroms als Teilwort und ist trotzdem eine Datei wie jede andere.
		"frontend/dist/assets/events-abc.js": []byte(strings.Repeat("console.log('ereignis');\n", 120)),
		"frontend/dist/assets/app.css":       []byte(strings.Repeat(".katalog { color: red; }\n", 120)),
		"frontend/dist/icons.svg":            []byte("<svg xmlns=\"http://www.w3.org/2000/svg\">" + strings.Repeat("<path d=\"M0 0h24v24H0z\"/>", 80) + "</svg>"),
		"frontend/dist/assets/logo.png":      append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte("bild"), 600)...),
		"uploads/cover_1.webp":               append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), bytes.Repeat([]byte("bild"), 600)...),
	}
	for pfad, inhalt := range dateien {
		if err := os.MkdirAll(filepath.Dir(pfad), 0o750); err != nil {
			t.Fatalf("Verzeichnis anlegen: %v", err)
		}
		if err := os.WriteFile(pfad, inhalt, 0o600); err != nil {
			t.Fatalf("Datei anlegen: %v", err)
		}
	}

	// Der Katalog führt nur Titel mit Exemplar; 150 Barcodes bringen die Barcode-Liste
	// über die Mindestgröße.
	for i := range 150 {
		titelID := seedMonitorTitel(t, pool, fmt.Sprintf("Kompression Titel %03d", i), fmt.Sprintf("KOM %03d", i), false, 0)
		exemplar(t, pool, titelID, fmt.Sprintf("B-KOMP-%04d", i), true, "")
	}

	authenticator, err := auth.NewAuthenticator(
		"kompression-testgeheimnis-mindestens-32-bytes!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	var kontoID string
	if err := pool.QueryRow(t.Context(), `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Kira', 'Katalog', 'kompression@example.org', 'admin', true)
		RETURNING id`).Scan(&kontoID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	sitzung, err := authenticator.GenerateToken(kontoID, "KOMP-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	router := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()

	hole := func(pfad string, kopf map[string]string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, pfad, nil)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		for name, wert := range kopf {
			req.Header.Set(name, wert)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	gzipAngeboten := map[string]string{"Accept-Encoding": "gzip"}

	t.Run("Oberfläche und Listen gehen gepackt hinaus", func(t *testing.T) {
		for _, pfad := range []string{
			"/assets/app.js", "/assets/events-abc.js", "/assets/app.css", "/icons.svg", "/katalog",
			"/api/books", "/api/action/buchbarcodes",
		} {
			offen := hole(pfad, nil)
			if offen.Code != http.StatusOK || offen.Body.Len() < kompressionMindestgroesse {
				t.Fatalf("GET %s: Status %d, %d Byte — ohne eine Antwort ab %d Byte misst der Test darunter nichts",
					pfad, offen.Code, offen.Body.Len(), kompressionMindestgroesse)
			}
			if kodierung := offen.Header().Get("Content-Encoding"); kodierung != "" {
				t.Errorf("GET %s ohne Angebot: Content-Encoding %q, erwartet keine", pfad, kodierung)
			}

			gepackt := hole(pfad, gzipAngeboten)
			if kodierung := gepackt.Header().Get("Content-Encoding"); kodierung != "gzip" {
				t.Errorf("GET %s mit Accept-Encoding gzip: Content-Encoding %q, erwartet gzip", pfad, kodierung)
				continue
			}
			if gepackt.Body.Len() >= offen.Body.Len() {
				t.Errorf("GET %s: gepackt %d Byte, unverpackt %d — das Packen spart nichts", pfad, gepackt.Body.Len(), offen.Body.Len())
			}
			if !bytes.Equal(entpackeGzip(t, pfad, gepackt.Body.Bytes()), offen.Body.Bytes()) {
				t.Errorf("GET %s: der entpackte Rumpf weicht vom unverpackten ab", pfad)
			}
			if !slices.Contains(gepackt.Header().Values("Vary"), "Accept-Encoding") {
				t.Errorf("GET %s: Vary %q nennt Accept-Encoding nicht — ein Zwischenspeicher gäbe die gepackte Fassung an jeden", pfad, gepackt.Header().Values("Vary"))
			}
			if a, b := offen.Header().Get("Cache-Control"), gepackt.Header().Get("Cache-Control"); a != b {
				t.Errorf("GET %s: Cache-Control %q unverpackt, %q gepackt — das Packen darf die Ablage-Regel nicht ändern", pfad, a, b)
			}
		}
	})

	t.Run("gepackt wird nur mit gzip", func(t *testing.T) {
		if kodierung := hole("/assets/app.js", map[string]string{"Accept-Encoding": "zstd, gzip"}).Header().Get("Content-Encoding"); kodierung != "gzip" {
			t.Errorf("Angebot „zstd, gzip“: Content-Encoding %q, erwartet gzip", kodierung)
		}
		nurZstd := hole("/assets/app.js", map[string]string{"Accept-Encoding": "zstd"})
		if kodierung := nurZstd.Header().Get("Content-Encoding"); kodierung != "" {
			t.Errorf("Angebot „zstd“: Content-Encoding %q, erwartet keine", kodierung)
		}
		if !bytes.Equal(nurZstd.Body.Bytes(), dateien["frontend/dist/assets/app.js"]) {
			t.Error("Angebot „zstd“: der Rumpf ist nicht die Datei")
		}
	})

	t.Run("Bilder bleiben unverändert", func(t *testing.T) {
		for pfad, datei := range map[string]string{
			"/assets/logo.png":      "frontend/dist/assets/logo.png",
			"/uploads/cover_1.webp": "uploads/cover_1.webp",
		} {
			rec := hole(pfad, gzipAngeboten)
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s: Status %d", pfad, rec.Code)
			}
			if kodierung := rec.Header().Get("Content-Encoding"); kodierung != "" {
				t.Errorf("GET %s: Content-Encoding %q — ein Bild ist schon gepackt", pfad, kodierung)
			}
			if !bytes.Equal(rec.Body.Bytes(), dateien[datei]) {
				t.Errorf("GET %s: der Rumpf ist nicht die Datei", pfad)
			}
			if laenge := rec.Header().Get("Content-Length"); laenge != fmt.Sprint(len(dateien[datei])) {
				t.Errorf("GET %s: Content-Length %q, erwartet %d", pfad, laenge, len(dateien[datei]))
			}
		}
	})

	t.Run("kleine Antworten bleiben unverpackt", func(t *testing.T) {
		// Die Antwort mit dem CSRF-Token ist die einzige, die ein Geheimnis im Rumpf trägt.
		for pfad, beginn := range map[string]string{
			"/health":         `{"status":"healthy"}`,
			"/api/csrf-token": `{"csrf_token":"`,
		} {
			rec := hole(pfad, gzipAngeboten)
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s: Status %d", pfad, rec.Code)
			}
			if kodierung := rec.Header().Get("Content-Encoding"); kodierung != "" {
				t.Errorf("GET %s: Content-Encoding %q, erwartet keine", pfad, kodierung)
			}
			if !strings.HasPrefix(rec.Body.String(), beginn) {
				t.Errorf("GET %s: Rumpf %q, erwartet die Antwort im Klartext", pfad, rec.Body.String())
			}
		}
	})

	t.Run("unverändert antwortet weiter ohne Rumpf", func(t *testing.T) {
		stand := hole("/api/action/buchbarcodes", gzipAngeboten).Header().Get("ETag")
		if stand == "" {
			t.Fatal("die Barcode-Liste trägt keinen ETag — ohne ihn misst der Test darunter nichts")
		}
		rec := hole("/api/action/buchbarcodes", map[string]string{"Accept-Encoding": "gzip", "If-None-Match": stand})
		if rec.Code != http.StatusNotModified || rec.Body.Len() != 0 {
			t.Errorf("If-None-Match %s: Status %d mit %d Byte Rumpf, erwartet 304 ohne Rumpf", stand, rec.Code, rec.Body.Len())
		}
		if kodierung := rec.Header().Get("Content-Encoding"); kodierung != "" {
			t.Errorf("304: Content-Encoding %q, erwartet keine", kodierung)
		}
	})

	t.Run("Vary nennt Herkunft und Kodierung", func(t *testing.T) {
		rec := hole("/assets/app.js", map[string]string{"Accept-Encoding": "gzip", "Origin": kompressionHerkunft})
		vary := rec.Header().Values("Vary")
		if !slices.Contains(vary, "Origin") || !slices.Contains(vary, "Accept-Encoding") {
			t.Errorf("Vary %q, erwartet Origin und Accept-Encoding", vary)
		}
	})

	t.Run("der Live-Strom läuft an der Kompression vorbei", func(t *testing.T) {
		server := httptest.NewServer(router)
		t.Cleanup(server.Close)
		ctx, abbrechen := context.WithTimeout(t.Context(), 5*time.Second)
		t.Cleanup(abbrechen)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/events", nil)
		if err != nil {
			t.Fatalf("Anfrage bauen: %v", err)
		}
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		req.Header.Set("Accept-Encoding", "gzip")
		antwort, err := server.Client().Do(req)
		if err != nil {
			t.Fatalf("GET /events: %v — der Strom muss sofort antworten", err)
		}
		t.Cleanup(func() {
			if err := antwort.Body.Close(); err != nil {
				t.Logf("Strom schließen: %v", err)
			}
		})
		if antwort.StatusCode != http.StatusOK || !strings.HasPrefix(antwort.Header.Get("Content-Type"), "text/event-stream") {
			t.Fatalf("GET /events: Status %d, Content-Type %q", antwort.StatusCode, antwort.Header.Get("Content-Type"))
		}
		if kodierung := antwort.Header.Get("Content-Encoding"); kodierung != "" {
			t.Errorf("GET /events: Content-Encoding %q, erwartet keine", kodierung)
		}
		if vary := antwort.Header.Values("Vary"); slices.Contains(vary, "Accept-Encoding") {
			t.Errorf("GET /events: Vary %q — der Strom lief durch die Kompression", vary)
		}
		zeile, err := bufio.NewReader(antwort.Body).ReadString('\n')
		if err != nil || strings.TrimSpace(zeile) != "event: connected" {
			t.Errorf("erste Zeile des Stroms %q (%v), erwartet „event: connected“", zeile, err)
		}
	})
}

// entpackeGzip liest einen gzip-Rumpf ganz aus.
func entpackeGzip(t *testing.T, pfad string, rumpf []byte) []byte {
	t.Helper()
	leser, err := gzip.NewReader(bytes.NewReader(rumpf))
	if err != nil {
		t.Fatalf("GET %s: Rumpf ist kein gzip: %v", pfad, err)
	}
	klar, err := io.ReadAll(leser)
	if err != nil {
		t.Fatalf("GET %s: gzip-Rumpf bricht ab: %v", pfad, err)
	}
	return klar
}
