package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Nach einem Rückbau auf einen älteren Stand ist die App-Shell auf dem Server älter als die im
// Browser. Fragt der Browser mit dem jüngeren Datum nach, muss er die Datei des Servers
// bekommen, sonst lädt er die Dateien des verworfenen Baus.
func TestOberflaeche_AppShellNachRueckbau(t *testing.T) {
	dist := t.TempDir()
	shell := filepath.Join(dist, "index.html")
	if err := os.WriteFile(shell, []byte("<!doctype html><title>Stand vom Rückbau</title>"), 0o600); err != nil {
		t.Fatalf("index.html anlegen: %v", err)
	}
	gebaut := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	if err := os.Chtimes(shell, gebaut, gebaut); err != nil {
		t.Fatalf("Datum setzen: %v", err)
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

	for _, pfad := range []string{"/", "/leserdatei"} {
		t.Run(pfad, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, pfad, nil)
			req.Header.Set("If-Modified-Since", gebaut.Add(48*time.Hour).Format(http.TimeFormat))
			rec := httptest.NewRecorder()
			liefere(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("Status %d, erwartet 200: Der Browser behielte die Seite des verworfenen Baus", rec.Code)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
				t.Errorf("Cache-Control %q, erwartet no-cache", got)
			}
			if got := rec.Header().Get("Last-Modified"); got != "" {
				t.Errorf("Last-Modified %q: Der nächste Aufruf verglich wieder nach Datum", got)
			}
			if got := rec.Body.String(); got != "<!doctype html><title>Stand vom Rückbau</title>" {
				t.Errorf("Inhalt %q", got)
			}
		})
	}
}
