package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestErzeugeAdminToken(t *testing.T) {
	geheimnis := "test-secret"
	tokenText := erzeugeAdminToken(geheimnis)
	if tokenText == "" {
		t.Error("erzeugeAdminToken(geheimnis) ist leer")
	}

	token, err := jwt.Parse(tokenText, func(token *jwt.Token) (interface{}, error) {
		return []byte(geheimnis), nil
	})
	if err != nil {
		t.Fatalf("jwt.Parse: %v", err)
	}
	if !token.Valid {
		t.Error("token.Valid = false, erwartet true")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("token.Claims = %T, erwartet jwt.MapClaims", token.Claims)
	}
	if claims["rolle"] != "ADMIN" {
		t.Errorf("claims[\"rolle\"] = %v, erwartet %v", claims["rolle"], "ADMIN")
	}
	if claims["barcode_id"] != "admin" {
		t.Errorf("claims[\"barcode_id\"] = %v, erwartet %v", claims["barcode_id"], "admin")
	}
}

func TestBaueAnfrage(t *testing.T) {
	zielURL := "http://example.com/api"
	token := "dummy-token"
	rumpf := []byte(`{"test": "data"}`)

	lauf := newLastlauf(zielURL, token, rumpf, 1)
	req, err := lauf.baueAnfrage()
	if err != nil {
		t.Fatalf("lauf.baueAnfrage: %v", err)
	}

	if req.Method != http.MethodPost {
		t.Errorf("req.Method = %q, erwartet %q", req.Method, http.MethodPost)
	}
	if got := req.URL.String(); got != zielURL {
		t.Errorf("req.URL.String() = %q, erwartet %q", got, zielURL)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, erwartet %q", got, "application/json")
	}
	if got := req.Header.Get("X-CSRF-Token"); got != "dummy-csrf-token-12345" {
		t.Errorf("X-CSRF-Token = %q, erwartet %q", got, "dummy-csrf-token-12345")
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	if !bytes.Equal(body, rumpf) {
		t.Errorf("body = %q, erwartet %q", body, rumpf)
	}

	err = req.Body.Close()
	if err != nil {
		t.Fatalf("req.Body.Close: %v", err)
	}

	cookies := req.Cookies()
	if len(cookies) != 2 {
		t.Fatalf("len(req.Cookies()) = %d, erwartet %d", len(cookies), 2)
	}
	if cookies[0].Name != "session_token" {
		t.Errorf("cookies[0].Name = %q, erwartet %q", cookies[0].Name, "session_token")
	}
	if cookies[0].Value != token {
		t.Errorf("cookies[0].Value = %q, erwartet %q", cookies[0].Value, token)
	}
	if cookies[1].Name != "csrf_token" {
		t.Errorf("cookies[1].Name = %q, erwartet %q", cookies[1].Name, "csrf_token")
	}
	if cookies[1].Value != "dummy-csrf-token-12345" {
		t.Errorf("cookies[1].Value = %q, erwartet %q", cookies[1].Value, "dummy-csrf-token-12345")
	}
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

	if anfragen != anzahl {
		t.Errorf("anfragen = %v, erwartet %v", anfragen, anzahl)
	}

	lauf.ergebnisSperre.Lock()
	defer lauf.ergebnisSperre.Unlock()
	if lauf.jeStatus[http.StatusOK] != anzahl {
		t.Errorf("lauf.jeStatus[http.StatusOK] = %v, erwartet %v", lauf.jeStatus[http.StatusOK], anzahl)
	}
	if len(lauf.jeStatus) != 1 {
		t.Errorf("len(lauf.jeStatus) = %d, erwartet %d", len(lauf.jeStatus), 1)
	}
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
	if lauf.jeStatus[0] != 5 {
		t.Errorf("lauf.jeStatus[0] = %v, erwartet %v", lauf.jeStatus[0], 5)
	}
}

func TestLeseKonfig_Erfolg(t *testing.T) {
	// Abgeschlossen in einem leeren Verzeichnis: leseKonfig liest zuerst ../../.env (die echte
	// .env im Wurzelverzeichnis mit PORT=8081) und nur ersatzweise die .env im Arbeitsverzeichnis.
	// Ein Test, der die .env im Arbeitsverzeichnis schreibt, wäre nur grün, wenn ../../.env fehlt,
	// also in der CI (dort ist sie in .gitignore) und nicht lokal. Ein t.TempDir als
	// Arbeitsverzeichnis hat weder ../../.env noch eine störende echte .env; die hier geschriebene
	// wird sicher gelesen, und die echte Datei bleibt unangetastet.
	t.Chdir(t.TempDir())
	if err := os.WriteFile(".env", []byte("PORT=9090\nJWT_SECRET=supersecret\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	konfigWerte := leseKonfig()
	if konfigWerte.port != "9090" {
		t.Errorf("konfigWerte.port = %q, erwartet %q", konfigWerte.port, "9090")
	}
	if konfigWerte.jwtGeheimnis != "supersecret" {
		t.Errorf("konfigWerte.jwtGeheimnis = %q, erwartet %q", konfigWerte.jwtGeheimnis, "supersecret")
	}
}

func TestDruckeErgebnisse(t *testing.T) {
	lauf := &lastlauf{
		jeStatus: map[int]int{
			200: 42,
			0:   3,
			500: 5,
		},
	}

	var aus strings.Builder
	lauf.druckeErgebnisse(&aus)

	gelesen := aus.String()
	if !strings.Contains(gelesen, "42x 200 OK") {
		t.Errorf("druckeErgebnisse schreibt %q, erwartet darin %q", gelesen, "42x 200 OK")
	}
	if !strings.Contains(gelesen, "3x Network Error") {
		t.Errorf("druckeErgebnisse schreibt %q, erwartet darin %q", gelesen, "3x Network Error")
	}
	if !strings.Contains(gelesen, "5x 500 Internal Server Error") {
		t.Errorf("druckeErgebnisse schreibt %q, erwartet darin %q", gelesen, "5x 500 Internal Server Error")
	}
}
