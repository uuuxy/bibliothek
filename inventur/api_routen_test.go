package inventur

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func setupTestAPIHandler() *APIHandler {
	return &APIHandler{
		jwtKey:        []byte("test-secret"),
		jwtIssuer:     "test-issuer",
		jwtAudience:   "test-audience",
		tokenVersion:  1,
		adminTokenTTL: time.Hour,
		guestTokenTTL: time.Hour,
		adminCookie:   "admin_token",
		guestCookie:   "guest_token",
		csrfCookie:    "csrf_cookie",
		csrfHeader:    "X-CSRF-Token",
	}
}

func TestRequireAdmin(t *testing.T) {
	handler := setupTestAPIHandler()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	middleware := handler.requireAdmin(nextHandler)

	t.Run("MissingToken", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("InvalidToken", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
		req.AddCookie(&http.Cookie{Name: handler.adminCookie, Value: "invalid.token.str"})
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("ValidGuestToken", func(t *testing.T) {
		token, _ := handler.issueToken(false)
		req := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
		req.AddCookie(&http.Cookie{Name: handler.guestCookie, Value: token})
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("ValidAdminToken_GET", func(t *testing.T) {
		token, _ := handler.issueToken(true)
		req := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
		req.AddCookie(&http.Cookie{Name: handler.adminCookie, Value: token})
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("ValidAdminToken_POST_MissingCSRF", func(t *testing.T) {
		token, _ := handler.issueToken(true)
		req := httptest.NewRequest(http.MethodPost, "/api/admin", nil)
		req.AddCookie(&http.Cookie{Name: handler.adminCookie, Value: token})
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rr.Code)
		}
	})

	t.Run("ValidAdminToken_POST_ValidCSRF", func(t *testing.T) {
		token, _ := handler.issueToken(true)
		csrfToken := "valid-csrf-token"

		req := httptest.NewRequest(http.MethodPost, "/api/admin", nil)
		req.AddCookie(&http.Cookie{Name: handler.adminCookie, Value: token})
		req.AddCookie(&http.Cookie{Name: handler.csrfCookie, Value: csrfToken})
		req.Header.Set(handler.csrfHeader, csrfToken)

		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})
}

func TestRequireAuth(t *testing.T) {
	handler := setupTestAPIHandler()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	middleware := handler.requireAuth(nextHandler)

	t.Run("MissingToken", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/books", nil)
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("ValidGuestToken_GET", func(t *testing.T) {
		token, _ := handler.issueToken(false)
		req := httptest.NewRequest(http.MethodGet, "/api/books", nil)
		req.AddCookie(&http.Cookie{Name: handler.guestCookie, Value: token})
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("ValidAdminToken_GET", func(t *testing.T) {
		token, _ := handler.issueToken(true)
		req := httptest.NewRequest(http.MethodGet, "/api/books", nil)
		req.AddCookie(&http.Cookie{Name: handler.adminCookie, Value: token})
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})
}
