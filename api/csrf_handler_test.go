package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSRFTokenHandlerOhneCookieErzeugtNeuesToken(t *testing.T) {
	s := &Server{CookieSecure: true}
	handler := s.CSRFTokenHandler()

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

func TestCSRFTokenHandlerMitCookieLiefertExistierendesToken(t *testing.T) {
	s := &Server{CookieSecure: true}
	handler := s.CSRFTokenHandler()

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

	// Prüfen, dass KEIN neues Cookie gesetzt wurde
	assert.Empty(t, rec.Header().Get("Set-Cookie"), "Sollte kein neues Cookie setzen, wenn bereits eines existiert")
}

func TestCSRFTokenHandlerMitLeerenCookieErzeugtNeuesToken(t *testing.T) {
	s := &Server{CookieSecure: true}
	handler := s.CSRFTokenHandler()

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
