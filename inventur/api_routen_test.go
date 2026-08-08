package inventur

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNeuteredFileSystem_Open(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "neutered_fs_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // clean up

	// Create a file in the root
	if err := os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create a subdirectory with an index.html
	subDirWithIndex := filepath.Join(tempDir, "with_index")
	if err := os.Mkdir(subDirWithIndex, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDirWithIndex, "index.html"), []byte("index"), 0644); err != nil {
		t.Fatalf("Failed to create index.html: %v", err)
	}

	// Create a subdirectory without an index.html
	subDirWithoutIndex := filepath.Join(tempDir, "without_index")
	if err := os.Mkdir(subDirWithoutIndex, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	fs := neuteredFileSystem{http.Dir(tempDir)}

	t.Run("Open existing file", func(t *testing.T) {
		f, err := fs.Open("file.txt")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if f != nil {
			f.Close()
		}
	})

	t.Run("Open non-existing file", func(t *testing.T) {
		_, err := fs.Open("nonexistent.txt")
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("Open directory with index.html", func(t *testing.T) {
		f, err := fs.Open("with_index")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if f != nil {
			f.Close()
		}
	})

	t.Run("Open directory without index.html", func(t *testing.T) {
		_, err := fs.Open("without_index")
		if err == nil {
			t.Error("Expected error, got nil")
		} else if !errors.Is(err, os.ErrNotExist) && !os.IsNotExist(err) {
			// In Go 1.20+, http.Dir returns fs.ErrNotExist which Is(os.ErrNotExist).
			// We check if it is some form of "not exist" error.
			t.Logf("Expected some 'not exist' error, got %v", err)
		}
	})
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
