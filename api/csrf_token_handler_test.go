package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFTokenHandler_NeuesToken(t *testing.T) {
	s := &Server{CookieSecure: true}
	handler := s.CSRFTokenHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200 OK, erhalten %d", rec.Code)
	}

	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, csrfCookieName+"=") {
		t.Errorf("Set-Cookie Header fehlt oder hat falschen Namen: %q", setCookie)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON konnte nicht geparst werden: %v", err)
	}

	if response["csrf_token"] == "" {
		t.Error("csrf_token in JSON Antwort ist leer")
	}

	if !strings.Contains(setCookie, response["csrf_token"]) {
		t.Errorf("Cookie-Wert stimmt nicht mit JSON-Antwort überein. Cookie: %q, JSON: %q", setCookie, response["csrf_token"])
	}
}

func TestCSRFTokenHandler_BestehendesToken(t *testing.T) {
	s := &Server{CookieSecure: true}
	handler := s.CSRFTokenHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "existing-tok-123"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200 OK, erhalten %d", rec.Code)
	}

	setCookie := rec.Header().Get("Set-Cookie")
	if setCookie != "" {
		t.Errorf("Set-Cookie Header sollte nicht gesetzt werden, wenn Token existiert: %q", setCookie)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON konnte nicht geparst werden: %v", err)
	}

	if response["csrf_token"] != "existing-tok-123" {
		t.Errorf("erwartet 'existing-tok-123', erhalten %q", response["csrf_token"])
	}
}
