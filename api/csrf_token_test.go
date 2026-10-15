package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSRFTokenOhneCookieErzeugtNeuesToken(t *testing.T) {
	s := &Server{nurHTTPS: true}
	handler := s.csrfToken()

	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Prüfen, ob JSON-Antwort existiert
	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	token, exists := resp["csrf_token"]
	require.True(t, exists, "JSON-Antwort sollte csrf_token enthalten")
	require.NotEmpty(t, token)

	// Prüfen, ob Set-Cookie Kopfzeile gesetzt wurde
	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1, "Sollte genau ein Cookie setzen")

	cookie := cookies[0]
	assert.Equal(t, csrfCookieName, cookie.Name)
	assert.Equal(t, token, cookie.Value, "Cookie-Wert sollte dem JSON-Wert entsprechen")
	assert.Equal(t, true, cookie.Secure, "Cookie sollte Secure-Flag gemäß Server-Konfiguration haben")
	assert.Equal(t, false, cookie.HttpOnly, "Cookie sollte für das Frontend lesbar sein (HttpOnly=false)")
}

func TestCSRFTokenMitCookieLiefertDasVorhandene(t *testing.T) {
	s := &Server{nurHTTPS: true}
	handler := s.csrfToken()

	existingToken := "mein-existierendes-token-123"

	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: existingToken})
	rec := httptest.NewRecorder()

	handler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	token, exists := resp["csrf_token"]
	require.True(t, exists)
	assert.Equal(t, existingToken, token, "Sollte das existierende Token zurückgeben")

	// Prüfen, dass kein neues Cookie gesetzt wurde
	assert.Empty(t, rec.Header().Get("Set-Cookie"), "Sollte kein neues Cookie setzen, wenn bereits eines existiert")
}

func TestCSRFTokenMitLeeremCookieErzeugtNeuesToken(t *testing.T) {
	s := &Server{nurHTTPS: true}
	handler := s.csrfToken()

	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	// Leeres Cookie setzen
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "   "})
	rec := httptest.NewRecorder()

	handler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	token, exists := resp["csrf_token"]
	require.True(t, exists)
	require.NotEmpty(t, token)
	assert.NotEqual(t, "   ", token, "Sollte ein neues Token generieren, wenn das alte nur aus Leerzeichen bestand")

	// Prüfen, dass ein neues Cookie gesetzt wurde
	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, token, cookies[0].Value)
}

func TestCSRFToken_NeuesToken(t *testing.T) {
	s := &Server{nurHTTPS: true}
	handler := s.csrfToken()

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

func TestCSRFToken_BestehendesToken(t *testing.T) {
	s := &Server{nurHTTPS: true}
	handler := s.csrfToken()

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
