package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFTokenOhneCookieErzeugtNeuesToken(t *testing.T) {
	s := &Server{nurHTTPS: true}
	handler := s.csrfToken()

	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %v, erwartet %v", rec.Code, http.StatusOK)
	}

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	token, exists := resp["csrf_token"]
	if !exists {
		t.Fatalf("Antwort = %s, erwartet das Feld csrf_token", rec.Body)
	}
	if token == "" {
		t.Fatal("token ist leer")
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("len(rec.Result().Cookies()) = %d, erwartet 1", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != csrfCookieName {
		t.Errorf("cookie.Name = %q, erwartet %q", cookie.Name, csrfCookieName)
	}
	if cookie.Value != token {
		t.Errorf("cookie.Value = %q, erwartet das Token der Antwort %q", cookie.Value, token)
	}
	if !cookie.Secure {
		t.Error("cookie.Secure = false, erwartet true: der Server läuft nur über HTTPS")
	}
	if cookie.HttpOnly {
		t.Error("cookie.HttpOnly = true, erwartet false: die Oberfläche liest das Cookie")
	}
}

func TestCSRFTokenMitCookieLiefertDasVorhandene(t *testing.T) {
	s := &Server{nurHTTPS: true}
	handler := s.csrfToken()

	existingToken := "mein-existierendes-token-123"

	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: existingToken})
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %v, erwartet %v", rec.Code, http.StatusOK)
	}

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	token, exists := resp["csrf_token"]
	if !exists {
		t.Fatalf("Antwort = %s, erwartet das Feld csrf_token", rec.Body)
	}
	if token != existingToken {
		t.Errorf("token = %q, erwartet das vorhandene %q", token, existingToken)
	}

	if got := rec.Header().Get("Set-Cookie"); got != "" {
		t.Errorf("Set-Cookie = %q, erwartet leer: das Cookie besteht schon", got)
	}
}

func TestCSRFTokenMitLeeremCookieErzeugtNeuesToken(t *testing.T) {
	s := &Server{nurHTTPS: true}
	handler := s.csrfToken()

	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	// Ein Cookie aus Leerzeichen gilt als fehlend.
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "   "})
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %v, erwartet %v", rec.Code, http.StatusOK)
	}

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	token, exists := resp["csrf_token"]
	if !exists {
		t.Fatalf("Antwort = %s, erwartet das Feld csrf_token", rec.Body)
	}
	if token == "" {
		t.Fatal("token ist leer")
	}
	if token == "   " {
		t.Errorf("token = %q, erwartet ein neues", token)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("len(rec.Result().Cookies()) = %d, erwartet %d", len(cookies), 1)
	}
	if cookies[0].Value != token {
		t.Errorf("cookies[0].Value = %q, erwartet %q", cookies[0].Value, token)
	}
}

func TestCSRFToken_NeuesToken(t *testing.T) {
	s := &Server{nurHTTPS: true}
	handler := s.csrfToken()

	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200", rec.Code)
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

func TestCSRFToken_BestehendesToken(t *testing.T) {
	s := &Server{nurHTTPS: true}
	handler := s.csrfToken()

	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "existing-tok-123"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200", rec.Code)
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
		t.Errorf("response[\"csrf_token\"] = %q, erwartet 'existing-tok-123'", response["csrf_token"])
	}
}
