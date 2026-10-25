package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"bibliothek/db"
	"bibliothek/pkg/oberflaechenstand"
	"bibliothek/sse"
)

// indexWieVomBau ist index.html in der Form, die vite build schreibt (frontend/dist).
const indexWieVomBau = `<!doctype html>
<html lang="de">
	<head>
		<meta charset="UTF-8" />
		<script type="module" crossorigin src="/assets/index-BQZzTtp2.js"></script>
		<link rel="stylesheet" crossorigin href="/assets/index-D7RhG9_i.css">
	<link rel="manifest" href="/manifest.webmanifest"></head>
	<body><div id="app"></div></body>
</html>
`

// legeOberflaecheAn legt frontend/dist mit index.html im Arbeitsordner des Tests an.
func legeOberflaecheAn(t *testing.T, index string) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(filepath.Join("frontend", "dist", "assets"), 0o700); err != nil {
		t.Fatalf("dist anlegen: %v", err)
	}
	if err := os.WriteFile(filepath.Join("frontend", "dist", "index.html"), []byte(index), 0o600); err != nil {
		t.Fatalf("index.html anlegen: %v", err)
	}
}

// Jede Antwort nennt den Stand der Oberfläche, auch eine Fehlerantwort und die App-Shell:
// Ein altes Fenster soll ihn an der ersten Antwort nach einem Update erfahren, gleich welcher.
func TestOberflaechenstand_JedeAntwortNenntIhn(t *testing.T) {
	legeOberflaecheAn(t, indexWieVomBau)
	router := NewServer(&db.Verbindung{}, nil, sse.NewBroker(), ServerKonfig{}).Routes()
	for _, pfad := range []string{"/api/gibt-es-nicht", "/", "/leserdatei"} {
		t.Run(pfad, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, pfad, nil))
			if got := rec.Header().Get(oberflaechenstand.Kopf); got != "/assets/index-BQZzTtp2.js" {
				t.Errorf("%s = %q, erwartet die Programmdatei aus index.html", oberflaechenstand.Kopf, got)
			}
		})
	}
}

// Ohne gebaute Oberfläche (Tests, ein Server ohne dist/) gibt es keinen Stand. Ein leerer Wert
// im Kopf hieße für die Oberfläche nichts, ein falscher verlangte ein Neuladen ohne Grund.
func TestOberflaechenstand_OhneOberflaecheKeinKopf(t *testing.T) {
	t.Chdir(t.TempDir())
	router := NewServer(&db.Verbindung{}, nil, sse.NewBroker(), ServerKonfig{}).Routes()
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/gibt-es-nicht", nil))
	if _, gesetzt := rec.Header()[oberflaechenstand.Kopf]; gesetzt {
		t.Errorf("%s gesetzt (%q), obwohl es keine Oberfläche gibt", oberflaechenstand.Kopf, rec.Header().Get(oberflaechenstand.Kopf))
	}
}
