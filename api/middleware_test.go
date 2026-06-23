package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPSRedirectMiddleware(t *testing.T) {
	s := &Server{CookieSecure: true}

	t.Run("valid r.Host", func(t *testing.T) {
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

	t.Run("invalid r.Host rejects", func(t *testing.T) {
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

	t.Run("https skips redirect", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://example.com/foo", nil)
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
