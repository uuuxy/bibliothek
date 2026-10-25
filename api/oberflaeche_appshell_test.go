package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/sse"
)

// Nach einem Rückbau auf einen älteren Stand sind die Dateien auf dem Server älter als die im
// Browser. Fragt der Browser mit dem jüngeren Datum nach, muss er die Datei des Servers bekommen,
// sonst lädt er die Dateien des verworfenen Baus: ohne Service Worker über die App-Shell, mit ihm
// über sw.js, an dem der Browser einen anderen Bau erkennt. Eine Datei unter assets/ trägt einen
// Prüfwert im Namen; der Bau des Rückbaus nennt andere.
func TestOberflaeche_NachRueckbau(t *testing.T) {
	dist := t.TempDir()
	gebaut := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	dateien := map[string]string{
		"index.html":             "<!doctype html><title>Stand vom Rückbau</title>",
		"sw.js":                  "precacheAndRoute([{url:'index.html',revision:'rueckbau'}])",
		"manifest.webmanifest":   `{"name":"Bibliothek"}`,
		"assets/index-4f3a9c.js": "console.log('Bau vom Rückbau')",
	}
	for name, inhalt := range dateien {
		pfad := filepath.Join(dist, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(pfad), 0o700); err != nil {
			t.Fatalf("Ordner für %s: %v", name, err)
		}
		if err := os.WriteFile(pfad, []byte(inhalt), 0o600); err != nil {
			t.Fatalf("%s anlegen: %v", name, err)
		}
		if err := os.Chtimes(pfad, gebaut, gebaut); err != nil {
			t.Fatalf("Datum von %s setzen: %v", name, err)
		}
	}
	root, err := os.OpenRoot(dist)
	if err != nil {
		t.Fatalf("OpenRoot: %v", err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Errorf("Root schließen: %v", err)
		}
	})
	liefere := liefereOberflaeche(root)
	rufe := func(pfad string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, pfad, nil)
		req.Header.Set("If-Modified-Since", gebaut.Add(48*time.Hour).Format(http.TimeFormat))
		rec := httptest.NewRecorder()
		liefere(rec, req)
		return rec
	}

	for _, fall := range []struct{ pfad, datei string }{
		{"/", "index.html"},
		{"/leserdatei", "index.html"},
		{"/sw.js", "sw.js"},
		{"/manifest.webmanifest", "manifest.webmanifest"},
	} {
		t.Run(fall.pfad, func(t *testing.T) {
			rec := rufe(fall.pfad)
			if rec.Code != http.StatusOK {
				t.Fatalf("Status %d, erwartet 200: Der Browser behielte die Datei des verworfenen Baus", rec.Code)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
				t.Errorf("Cache-Control %q, erwartet no-cache", got)
			}
			if got := rec.Header().Get("Last-Modified"); got != "" {
				t.Errorf("Last-Modified %q: Der nächste Aufruf verglich wieder nach Datum", got)
			}
			if got := rec.Body.String(); got != dateien[fall.datei] {
				t.Errorf("Inhalt %q, erwartet %s", got, fall.datei)
			}
		})
	}

	t.Run("/assets/index-4f3a9c.js", func(t *testing.T) {
		if rec := rufe("/assets/index-4f3a9c.js"); rec.Code != http.StatusNotModified {
			t.Errorf("Status %d, erwartet 304: Eine Datei mit Prüfwert im Namen darf der Browser behalten", rec.Code)
		}
	})
}

// Eine Adresse unter /api/, die keine Route nimmt, meldet sich mit jeder Methode als unbekannt:
// Ein POST an einen umbenannten Endpunkt ist kein Fehler der Methode. Geprüft am Router samt
// Middleware, deshalb tragen die ändernden Anfragen das Token gegen CSRF.
func TestOberflaeche_UnbekannterEndpunkt(t *testing.T) {
	t.Chdir(t.TempDir())
	router := NewServer(&db.Verbindung{}, nil, sse.NewBroker(), ServerKonfig{}).Routes()
	for _, fall := range []struct{ methode, pfad string }{
		{http.MethodGet, "/api/gibt-es-nicht"},
		{http.MethodPost, "/api/gibt-es-nicht"},
		{http.MethodPut, "/api/admin/books/reorder"},
		{http.MethodPost, "/api/admin/books/import"},
		{http.MethodDelete, "/api/books/1"},
	} {
		t.Run(fall.methode+" "+fall.pfad, func(t *testing.T) {
			req := httptest.NewRequest(fall.methode, fall.pfad, nil)
			req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "probe"})
			req.Header.Set("X-CSRF-Token", "probe")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			var antwort struct {
				Fehler string `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
				t.Fatalf("Antwort ist kein JSON-Fehler (%s): %v", rec.Body.String(), err)
			}
			want := "Unbekannter Endpunkt: " + fall.methode + " " + fall.pfad
			if rec.Code != http.StatusNotFound || antwort.Fehler != want {
				t.Errorf("Status %d, Meldung %q, erwartet 404 und %q", rec.Code, antwort.Fehler, want)
			}
		})
	}
}
