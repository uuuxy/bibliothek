package inventur

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNeuteredFileSystem_Open hält die Aufgabe dieses Dateisystems fest: Ein Verzeichnis
// lässt sich nicht öffnen, mit oder ohne index.html. Öffnet es sich, zeigt der
// Go-Dateiserver an seiner Stelle die Dateinamen darin als Liste.
func TestNeuteredFileSystem_Open(t *testing.T) {
	// t.TempDir raeumt selbst auf — kein deferter RemoveAll, dessen Fehler niemand liest.
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Datei anlegen: %v", err)
	}

	subDirWithIndex := filepath.Join(tempDir, "with_index")
	if err := os.Mkdir(subDirWithIndex, 0755); err != nil {
		t.Fatalf("Verzeichnis anlegen: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDirWithIndex, "index.html"), []byte("index"), 0644); err != nil {
		t.Fatalf("index.html anlegen: %v", err)
	}

	subDirWithoutIndex := filepath.Join(tempDir, "without_index")
	if err := os.Mkdir(subDirWithoutIndex, 0755); err != nil {
		t.Fatalf("Verzeichnis anlegen: %v", err)
	}

	fs := neuteredFileSystem{http.Dir(tempDir)}

	// schliesse macht aus dem Close ein gepruefte Handlung: bliebe hier ein Handle
	// offen, faende man die Ursache spaeter nur noch als Datei-Limit im Betrieb.
	schliesse := func(t *testing.T, f http.File) {
		t.Helper()
		if f == nil {
			return
		}
		if err := f.Close(); err != nil {
			t.Errorf("Schliessen fehlgeschlagen: %v", err)
		}
	}

	t.Run("vorhandene Datei", func(t *testing.T) {
		f, err := fs.Open("file.txt")
		if err != nil {
			t.Errorf("kein Fehler erwartet, bekam %v", err)
		}
		schliesse(t, f)
	})

	t.Run("nicht vorhandene Datei", func(t *testing.T) {
		_, err := fs.Open("nonexistent.txt")
		if err == nil {
			t.Error("Fehler erwartet, bekam nil")
		}
	})

	t.Run("Verzeichnis mit index.html", func(t *testing.T) {
		f, err := fs.Open("with_index")
		if err == nil {
			schliesse(t, f)
			t.Fatal("das Verzeichnis liess sich oeffnen — der Dateiserver zeigt dann seine Dateinamen als Liste")
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("os.ErrNotExist erwartet, bekam %v", err)
		}
	})

	t.Run("Open directory without index.html", func(t *testing.T) {
		f, err := fs.Open("without_index")
		if err == nil {
			schliesse(t, f)
			t.Fatal("das Verzeichnis liess sich oeffnen — der Dateiserver wuerde jetzt ein Listing ausliefern")
		}
		// Es muss ein „existiert nicht" sein: nur daraus macht net/http ein 404.
		// Ein anderer Fehler wuerde als 500 durchschlagen und die SPA-Weiche stoeren.
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("os.ErrNotExist erwartet, bekam %v", err)
		}
	})
}

// /uploads/ ist ohne Anmeldung lesbar. Ausgeliefert wird deshalb nur, was direkt im
// Verzeichnis liegt: Dort legen Upload und Cover-Abruf ihre Dateien ab.
func TestUploadsLiefertNurDateienDirektImVerzeichnis(t *testing.T) {
	verzeichnis := imTestVerzeichnis(t)
	lege := func(rel, inhalt string) {
		t.Helper()
		pfad := filepath.Join(verzeichnis, rel)
		if err := os.MkdirAll(filepath.Dir(pfad), 0o750); err != nil {
			t.Fatalf("Verzeichnis anlegen: %v", err)
		}
		if err := os.WriteFile(pfad, []byte(inhalt), 0o600); err != nil {
			t.Fatalf("Datei anlegen: %v", err)
		}
	}
	lege("uploads/cover_1.webp", "cover-inhalt")
	lege("uploads/fotos/S-10041.jpg", "foto-inhalt")
	// Eine index.html im Unterordner öffnet das Verzeichnis nicht: Der Dateiserver zeigte
	// sonst die Dateinamen darin als Liste.
	lege("uploads/fotos/index.html", "index-inhalt")
	lege("uploads/covers/abc.webp", "zwischenspeicher-inhalt")

	durchlass := func(next http.Handler) http.Handler { return next }
	handler := NewAPIHandler(APIHandlerConfig{
		RequireViewBooks:     durchlass,
		RequireEditBooks:     durchlass,
		RequireDeleteBooks:   durchlass,
		RequireAuthenticated: durchlass,
	})

	faelle := []struct {
		pfad   string
		status int
	}{
		{"/uploads/cover_1.webp", http.StatusOK},
		{"/uploads/fotos/S-10041.jpg", http.StatusNotFound},
		{"/uploads/covers/abc.webp", http.StatusNotFound},
		{"/uploads/fotos/", http.StatusNotFound},
		{"/uploads/fotos", http.StatusNotFound},
		{"/uploads/", http.StatusNotFound},
	}
	for _, f := range faelle {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, f.pfad, nil))
		if rec.Code != f.status {
			t.Errorf("GET %s: Status %d, erwartet %d", f.pfad, rec.Code, f.status)
		}
		// Der Treffer belegt, dass der Server aus diesem Verzeichnis liest — sonst wären
		// die 404 darunter auch ohne die Regel grün.
		if f.status == http.StatusOK && rec.Body.String() != "cover-inhalt" {
			t.Errorf("GET %s: Rumpf %q, erwartet den Inhalt der Datei", f.pfad, rec.Body.String())
		}
		if f.status == http.StatusNotFound && strings.Contains(rec.Body.String(), "-inhalt") {
			t.Errorf("GET %s: die Antwort trägt den Inhalt einer Datei aus einem Unterordner", f.pfad)
		}
		if f.status == http.StatusNotFound && strings.Contains(rec.Body.String(), "S-10041") {
			t.Errorf("GET %s: die Antwort nennt eine Datei aus einem Unterordner", f.pfad)
		}
	}
}

func TestNewAPIHandler_And_ServeHTTP(t *testing.T) {
	// Dummy configurations
	config := APIHandlerConfig{
		Repo:      nil,
		Metadaten: nil,
		RequireViewBooks: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware", "ViewBooks")
				// We don't call next.ServeHTTP because repo is nil and the underlying handler would panic
				w.WriteHeader(http.StatusOK)
			})
		},
		RequireEditBooks: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware", "EditBooks")
				// We don't call next.ServeHTTP because repo is nil and the underlying handler would panic
				w.WriteHeader(http.StatusOK)
			})
		},
		RequireDeleteBooks: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware", "DeleteBooks")
				w.WriteHeader(http.StatusOK)
			})
		},
		RequireAuthenticated: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware", "Authenticated")
				// We don't call next.ServeHTTP because repo is nil and the underlying handler would panic
				w.WriteHeader(http.StatusOK)
			})
		},
	}

	handler := NewAPIHandler(config)

	if handler.repo != nil {
		t.Errorf("Expected repo to be nil")
	}
	if handler.metadaten != nil {
		t.Errorf("Expected metadaten to be nil")
	}
	if handler.mux == nil {
		t.Errorf("Expected mux to be initialized")
	}

	tests := []struct {
		name           string
		method         string
		path           string
		expectedHeader string
		expectedStatus int
	}{
		{
			name:           "GET /api/books (ViewBooks middleware)",
			method:         "GET",
			path:           "/api/books",
			expectedHeader: "ViewBooks",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST /api/books (EditBooks middleware)",
			method:         "POST",
			path:           "/api/books",
			expectedHeader: "EditBooks",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET /api/books/123 (ViewBooks middleware)",
			method:         "GET",
			path:           "/api/books/123",
			expectedHeader: "ViewBooks",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET /api/admin/ (EditBooks middleware)",
			method:         "GET",
			path:           "/api/admin/some-path",
			expectedHeader: "EditBooks",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "DELETE /api/admin/ (EditBooks middleware)",
			method:         "DELETE",
			path:           "/api/admin/some-path",
			expectedHeader: "EditBooks",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET /api/unknown-route (No middleware, 404)",
			method:         "GET",
			path:           "/api/unknown-route",
			expectedHeader: "",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rr.Code)
			}

			if tc.expectedHeader != "" {
				got := rr.Header().Get("X-Middleware")
				if got != tc.expectedHeader {
					t.Errorf("Expected header %q, got %q", tc.expectedHeader, got)
				}
			}
		})
	}
}
