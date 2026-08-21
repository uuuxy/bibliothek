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

func TestGenerateAdminToken(t *testing.T) {
	secret := "test-secret"
	tokenStr := generateAdminToken(secret)
	assert.NotEmpty(t, tokenStr)

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)
	assert.True(t, token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, "ADMIN", claims["rolle"])
	assert.Equal(t, "admin", claims["barcode_id"])
}

func TestBaueRequest(t *testing.T) {
	baseURL := "http://example.com/api"
	token := "dummy-token"
	jsonData := []byte(`{"test": "data"}`)

	runner := newStresstestRunner(baseURL, token, jsonData, 1)
	req, err := runner.baueRequest()
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, baseURL, req.URL.String())
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	assert.Equal(t, "dummy-csrf-token-12345", req.Header.Get("X-CSRF-Token"))

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, jsonData, body)

	err = req.Body.Close()
	require.NoError(t, err)

	cookies := req.Cookies()
	require.Len(t, cookies, 2)
	assert.Equal(t, "session_token", cookies[0].Name)
	assert.Equal(t, token, cookies[0].Value)
	assert.Equal(t, "csrf_token", cookies[1].Name)
	assert.Equal(t, "dummy-csrf-token-12345", cookies[1].Value)
}

func TestStresstestRunner_EndToEnd(t *testing.T) {
	var mu sync.Mutex
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()

		// Simulate some processing time
		time.Sleep(5 * time.Millisecond)

		// Return 200 OK
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	numRequests := 10
	runner := newStresstestRunner(server.URL, "token", []byte(`{}`), numRequests)

	runner.starteAlle()
	runner.gibStartsignal()
	runner.wg.Wait()

	assert.Equal(t, numRequests, requestCount)

	runner.resultsMu.Lock()
	defer runner.resultsMu.Unlock()
	assert.Equal(t, numRequests, runner.statusCounts[http.StatusOK])
	assert.Len(t, runner.statusCounts, 1)
}

func TestStresstestRunner_NetworkError(t *testing.T) {
	// Use an invalid URL that will definitely fail to connect immediately
	runner := newStresstestRunner("http://127.0.0.1:0/invalid", "token", []byte(`{}`), 5)

	runner.starteAlle()
	runner.gibStartsignal()
	runner.wg.Wait()

	runner.resultsMu.Lock()
	defer runner.resultsMu.Unlock()

	// Network errors should be logged under status code 0
	assert.Equal(t, 5, runner.statusCounts[0])
}

func TestLoadConfig_Success(t *testing.T) {
	// Back up existing .env if it exists
	var existingEnv []byte
	hasExistingEnv := true
	if data, err := os.ReadFile(".env"); err == nil {
		existingEnv = data
	} else {
		hasExistingEnv = false
	}

	// Setup a temporary .env file in the current working directory
	envContent := []byte("PORT=9090\nJWT_SECRET=supersecret\n")
	err := os.WriteFile(".env", envContent, 0644)
	require.NoError(t, err)
	defer func() {
		if hasExistingEnv {
			restoreErr := os.WriteFile(".env", existingEnv, 0644)
			if restoreErr != nil {
				t.Logf("failed to restore .env: %v", restoreErr)
			}
		} else {
			err := os.Remove(".env")
			if err != nil {
				t.Logf("failed to remove .env: %v", err)
			}
		}
	}()

	cfg := loadConfig()
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "supersecret", cfg.JWTSecret)
}

func TestDruckeErgebnisse(t *testing.T) {
	runner := &stresstestRunner{
		statusCounts: map[int]int{
			200: 42,
			0:   3,
			500: 5,
		},
	}

	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = old }()

	runner.druckeErgebnisse()

	err = w.Close()
	require.NoError(t, err)

	out, err := io.ReadAll(r)
	require.NoError(t, err)

	err = r.Close()
	require.NoError(t, err)

	output := string(out)
	assert.Contains(t, output, "42x 200 OK")
	assert.Contains(t, output, "3x Network Error")
	assert.Contains(t, output, "5x 500 Internal Server Error")
}