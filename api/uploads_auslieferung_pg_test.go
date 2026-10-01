package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/sse"
)

// Der Weg von außen, am Router des Betriebs und ohne Sitzung: Ein Cover kommt an, eine
// Datei aus einem Unterordner von uploads/ nicht. Die Regel selbst prüft
// inventur/api_routen_test.go; hier steht, dass kein Mount an ihr vorbeiführt.
func TestUploadsOhneAnmeldungNurDateienDirektImVerzeichnis(t *testing.T) {
	pool := pgTestPool(t)
	t.Chdir(t.TempDir())

	lege := func(rel, inhalt string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(rel), 0o750); err != nil {
			t.Fatalf("Verzeichnis anlegen: %v", err)
		}
		if err := os.WriteFile(rel, []byte(inhalt), 0o600); err != nil {
			t.Fatalf("Datei anlegen: %v", err)
		}
	}
	lege("uploads/cover_1.webp", "cover-inhalt")
	lege("uploads/fotos/S-10041.jpg", "foto-inhalt")

	authenticator, err := auth.NewAuthenticator(
		"uploads-auslieferung-testgeheimnis-32-bytes!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	router := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()

	hole := func(pfad string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, pfad, nil))
		return rec
	}

	if rec := hole("/uploads/cover_1.webp"); rec.Code != http.StatusOK || rec.Body.String() != "cover-inhalt" {
		t.Fatalf("GET /uploads/cover_1.webp: Status %d, Rumpf %q — ohne diesen Treffer misst der Test darunter nichts",
			rec.Code, rec.Body.String())
	}
	for _, pfad := range []string{"/uploads/fotos/S-10041.jpg", "/uploads/fotos/"} {
		rec := hole(pfad)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s ohne Anmeldung: Status %d, erwartet 404", pfad, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "foto-inhalt") {
			t.Errorf("GET %s ohne Anmeldung: die Antwort trägt den Inhalt der Datei", pfad)
		}
	}
}
