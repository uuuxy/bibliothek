package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHTTPSRedirectMiddleware(t *testing.T) {
	s := &Server{CookieSecure: true}

	t.Run("no ALLOWED_ORIGIN fallback to valid r.Host", func(t *testing.T) {
		_ = os.Setenv("ALLOWED_ORIGIN", "") //nolint:errcheck
		req := httptest.NewRequest(http.MethodGet, "http://example.com/foo", nil)
		rr := httptest.NewRecorder()

		handler := s.HTTPSRedirectMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusMovedPermanently {
			t.Errorf("Expected 301, got %d", rr.Code)
		}

		loc := rr.Header().Get("Location")
		if loc != "https://example.com/foo" {
			t.Errorf("Expected https://example.com/foo, got %s", loc)
		}
	})

	// Die Sonde des Containers spricht Klartext-HTTP gegen 127.0.0.1 und trägt weder TLS
	// noch X-Forwarded-Proto. Wird sie umgeleitet, folgt busybox-wget der 301 auf https://
	// und scheitert am TLS-Handshake gegen den Klartext-Port ("wrong version number") —
	// in Produktion also immer, lokal (CookieSecure=false) nie. Der Container galt damit
	// dauerhaft als unhealthy, und jede Entscheidung, die daran hing, war blind.
	t.Run("health probe is never redirected", func(t *testing.T) {
		_ = os.Setenv("ALLOWED_ORIGIN", "https://bibliothek.schule.de") //nolint:errcheck
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8083/health", nil)
		rr := httptest.NewRecorder()

		durchgelassen := false
		handler := s.HTTPSRedirectMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			durchgelassen = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(rr, req)

		if !durchgelassen || rr.Code != http.StatusOK {
			t.Errorf("Health-Sonde wurde umgeleitet (Status %d, Location %q) — der Container "+
				"meldet sich damit dauerhaft als unhealthy", rr.Code, rr.Header().Get("Location"))
		}
	})

	t.Run("no ALLOWED_ORIGIN fallback to invalid r.Host rejects", func(t *testing.T) {
		_ = os.Setenv("ALLOWED_ORIGIN", "") //nolint:errcheck
		req := httptest.NewRequest(http.MethodGet, "http://example.com/foo", nil)
		req.Host = "example.com\r\n"
		rr := httptest.NewRecorder()

		handler := s.HTTPSRedirectMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", rr.Code)
		}
	})

	t.Run("with ALLOWED_ORIGIN overrides r.Host", func(t *testing.T) {
		_ = os.Setenv("ALLOWED_ORIGIN", "https://bibliothek.schule.de") //nolint:errcheck
		req := httptest.NewRequest(http.MethodGet, "http://attacker.com/foo", nil)
		rr := httptest.NewRecorder()

		handler := s.HTTPSRedirectMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusMovedPermanently {
			t.Errorf("Expected 301, got %d", rr.Code)
		}

		loc := rr.Header().Get("Location")
		if loc != "https://bibliothek.schule.de/foo" {
			t.Errorf("Expected https://bibliothek.schule.de/foo, got %s", loc)
		}
	})

	t.Run("https skips redirect", func(t *testing.T) {
		_ = os.Setenv("ALLOWED_ORIGIN", "https://bibliothek.schule.de") //nolint:errcheck
		req := httptest.NewRequest(http.MethodGet, "https://bibliothek.schule.de/foo", nil)
		rr := httptest.NewRecorder()

		handler := s.HTTPSRedirectMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", rr.Code)
		}
	})
}
