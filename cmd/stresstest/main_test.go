package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErzeugeAdminToken(t *testing.T) {
	geheimnis := "test-secret"
	tokenText := erzeugeAdminToken(geheimnis)
	assert.NotEmpty(t, tokenText)

	token, err := jwt.Parse(tokenText, func(token *jwt.Token) (interface{}, error) {
		return []byte(geheimnis), nil
	})
	require.NoError(t, err)
	assert.True(t, token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, "ADMIN", claims["rolle"])
	assert.Equal(t, "admin", claims["barcode_id"])
}

func TestBaueAnfrage(t *testing.T) {
	zielURL := "http://example.com/api"
	token := "dummy-token"
	rumpf := []byte(`{"test": "data"}`)

	lauf := newLastlauf(zielURL, token, rumpf, 1)
	req, err := lauf.baueAnfrage()
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, zielURL, req.URL.String())
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	assert.Equal(t, "dummy-csrf-token-12345", req.Header.Get("X-CSRF-Token"))

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, rumpf, body)

	err = req.Body.Close()
	require.NoError(t, err)

	cookies := req.Cookies()
	require.Len(t, cookies, 2)
	assert.Equal(t, "session_token", cookies[0].Name)
	assert.Equal(t, token, cookies[0].Value)
	assert.Equal(t, "csrf_token", cookies[1].Name)
	assert.Equal(t, "dummy-csrf-token-12345", cookies[1].Value)
}

func TestLastlauf_VonAnfangBisEnde(t *testing.T) {
	var sperre sync.Mutex
	anfragen := 0

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sperre.Lock()
		anfragen++
		sperre.Unlock()

		// Etwas Bearbeitungszeit vortäuschen.
		time.Sleep(5 * time.Millisecond)

		// 200 OK zurückgeben.
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	anzahl := 10
	lauf := newLastlauf(testServer.URL, "token", []byte(`{}`), anzahl)

	lauf.starteAlle()
	lauf.gibStartsignal()
	lauf.alle.Wait()

	assert.Equal(t, anzahl, anfragen)

	lauf.ergebnisSperre.Lock()
	defer lauf.ergebnisSperre.Unlock()
	assert.Equal(t, anzahl, lauf.jeStatus[http.StatusOK])
	assert.Len(t, lauf.jeStatus, 1)
}

func TestLastlauf_Netzfehler(t *testing.T) {
	// Eine Adresse, an der die Verbindung sofort scheitert.
	lauf := newLastlauf("http://127.0.0.1:0/invalid", "token", []byte(`{}`), 5)

	lauf.starteAlle()
	lauf.gibStartsignal()
	lauf.alle.Wait()

	lauf.ergebnisSperre.Lock()
	defer lauf.ergebnisSperre.Unlock()

	// Fehler des Netzes zählen unter dem Status 0.
	assert.Equal(t, 5, lauf.jeStatus[0])
}

func TestLeseKonfig_Erfolg(t *testing.T) {
	// Abgeschlossen in einem leeren Verzeichnis: leseKonfig liest zuerst ../../.env (die echte
	// .env im Wurzelverzeichnis mit PORT=8081) und nur ersatzweise die .env im Arbeitsverzeichnis.
	// Ein Test, der die .env im Arbeitsverzeichnis schreibt, wäre nur grün, wenn ../../.env fehlt,
	// also in der CI (dort ist sie in .gitignore) und nicht lokal. Ein t.TempDir als
	// Arbeitsverzeichnis hat weder ../../.env noch eine störende echte .env; die hier geschriebene
	// wird sicher gelesen, und die echte Datei bleibt unangetastet.
	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile(".env", []byte("PORT=9090\nJWT_SECRET=supersecret\n"), 0o644))

	konfigWerte := leseKonfig()
	assert.Equal(t, "9090", konfigWerte.port)
	assert.Equal(t, "supersecret", konfigWerte.jwtGeheimnis)
}

func TestDruckeErgebnisse(t *testing.T) {
	lauf := &lastlauf{
		jeStatus: map[int]int{
			200: 42,
			0:   3,
			500: 5,
		},
	}

	altStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = altStdout }()

	lauf.druckeErgebnisse()

	err = w.Close()
	require.NoError(t, err)

	out, err := io.ReadAll(r)
	require.NoError(t, err)

	err = r.Close()
	require.NoError(t, err)

	gelesen := string(out)
	assert.Contains(t, gelesen, "42x 200 OK")
	assert.Contains(t, gelesen, "3x Network Error")
	assert.Contains(t, gelesen, "5x 500 Internal Server Error")
}
