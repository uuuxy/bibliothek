package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Config represents values extracted from .env
type Config struct {
	Port      string
	JWTSecret string
}

func loadConfig() Config {
	data, err := os.ReadFile("../../.env")
	if err != nil {
		data, err = os.ReadFile(".env")
		if err != nil {
			log.Fatalf("Could not read .env file: %v", err)
		}
	}

	cfg := Config{Port: "8080", JWTSecret: ""}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PORT=") {
			cfg.Port = strings.TrimPrefix(line, "PORT=")
		} else if strings.HasPrefix(line, "JWT_SECRET=") {
			cfg.JWTSecret = strings.TrimPrefix(line, "JWT_SECRET=")
		}
	}

	if cfg.JWTSecret == "" {
		log.Fatalf("JWT_SECRET not found in .env")
	}

	return cfg
}

func generateAdminToken(secret string) string {
	claims := jwt.MapClaims{
		// Hardcoded admin ID from seed.sql
		"user_id":    "00000000-0000-0000-0000-000000000001",
		"barcode_id": "admin",
		"rolle":      "ADMIN",
		"exp":        time.Now().Add(1 * time.Hour).Unix(),
		"iat":        time.Now().Unix(),
		"nbf":        time.Now().Unix(),
		"iss":        "bibliothek-system",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Fatalf("Failed to sign token: %v", err)
	}

	return signedToken
}

type ActionRequest struct {
	Query           string  `json:"query"`
	ActiveStudentID *string `json:"active_student_id,omitempty"`
}

// stresstestRunner bündelt die geteilte Concurrency-State: die Start-Barriere (damit
// alle Worker exakt gleichzeitig feuern) und die gesammelten Status-Code-Zähler.
type stresstestRunner struct {
	baseURL     string
	jsonData    []byte
	token       string
	numRequests int

	wg         sync.WaitGroup
	startMu    sync.Mutex
	startCond  *sync.Cond
	readyCount int
	start      bool

	resultsMu    sync.Mutex
	statusCounts map[int]int
}

func newStresstestRunner(baseURL, token string, jsonData []byte, numRequests int) *stresstestRunner {
	r := &stresstestRunner{
		baseURL:      baseURL,
		token:        token,
		jsonData:     jsonData,
		numRequests:  numRequests,
		statusCounts: make(map[int]int),
	}
	r.startCond = sync.NewCond(&r.startMu)
	return r
}

// baueRequest erstellt den POST-Request inkl. Header sowie CSRF- und Session-Cookie.
func (r *stresstestRunner) baueRequest() (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPost, r.baseURL, bytes.NewReader(r.jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Dummy CSRF Token protection bypass
	csrfToken := "dummy-csrf-token-12345"
	req.Header.Set("X-CSRF-Token", csrfToken)

	// Session cookie
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: r.token,
	})
	req.AddCookie(&http.Cookie{
		Name:  "csrf_token",
		Value: csrfToken,
	})
	return req, nil
}

// wartAufStart meldet den Worker als bereit und blockiert, bis das Startsignal kommt.
func (r *stresstestRunner) wartAufStart() {
	r.startMu.Lock()
	r.readyCount++
	if r.readyCount == r.numRequests {
		r.startCond.Broadcast() // Wake up everyone if this is the last one
	}
	for !r.start {
		r.startCond.Wait()
	}
	r.startMu.Unlock()
}

// erfasseErgebnis feuert den Request und protokolliert den Status-Code bzw. Netzwerkfehler.
func (r *stresstestRunner) erfasseErgebnis(client *http.Client, req *http.Request) {
	resp, err := client.Do(req)
	if err != nil {
		// network error → statusCode bleibt 0

		// Log the first error for debugging
		r.resultsMu.Lock()
		if len(r.statusCounts) == 0 {
			fmt.Printf("\n[DEBUG] Network error details: %v\n", err)
		}
		r.statusCounts[0]++
		r.resultsMu.Unlock()
		return
	}
	statusCode := resp.StatusCode
	_, _ = io.Copy(io.Discard, resp.Body) //nolint:errcheck
	_ = resp.Body.Close()                 //nolint:errcheck

	r.resultsMu.Lock()
	r.statusCounts[statusCode]++
	r.resultsMu.Unlock()
}

// fuehreWorkerAus ist der Rumpf einer Worker-Goroutine: Request bauen, auf das
// gemeinsame Startsignal warten und dann feuern.
func (r *stresstestRunner) fuehreWorkerAus() {
	defer r.wg.Done()

	client := &http.Client{
		Timeout: 10 * time.Second,
		// Wir schalten HTTP/2 aus und zwingen das Programm neue Connections aufzubauen,
		// um die Parallelität bei manchen OS/Go-Versionen sicherzustellen.
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
	}

	req, err := r.baueRequest()
	if err != nil {
		r.resultsMu.Lock()
		r.statusCounts[-1]++ // -1 means request creation failed
		r.resultsMu.Unlock()
		return
	}

	r.wartAufStart()
	r.erfasseErgebnis(client, req)
}

// starteAlle startet numRequests Worker-Goroutinen (die zunächst an der Barriere warten).
func (r *stresstestRunner) starteAlle() {
	r.wg.Add(r.numRequests)
	for i := 0; i < r.numRequests; i++ {
		go r.fuehreWorkerAus()
	}
}

// gibStartsignal wartet, bis alle Worker bereitstehen, und weckt sie gleichzeitig.
func (r *stresstestRunner) gibStartsignal() {
	// Give goroutines a moment to spin up and wait on condition
	time.Sleep(100 * time.Millisecond)

	r.startMu.Lock()
	if r.readyCount < r.numRequests {
		r.startCond.Wait()
	}
	r.start = true
	r.startCond.Broadcast()
	r.startMu.Unlock()
}

func (r *stresstestRunner) druckeErgebnisse() {
	fmt.Println("\n--- Stress Test Results ---")
	for code, count := range r.statusCounts {
		if code == 0 {
			fmt.Printf("%dx Network Error (Failed to execute request)\n", count)
		} else {
			statusText := http.StatusText(code)
			fmt.Printf("%dx %d %s\n", count, code, statusText)
		}
	}
}

func main() {
	portFlag := flag.String("port", "", "Port to run the stress test against (overrides .env)")
	flag.Parse()

	cfg := loadConfig()
	port := cfg.Port
	if *portFlag != "" {
		port = *portFlag
	}

	token := generateAdminToken(cfg.JWTSecret)
	baseURL := fmt.Sprintf("http://127.0.0.1:%s/api/action", port)

	// Hardcoded test data from seed.sql
	studentID := "00000000-0000-0000-0000-000000000003" // Max Mustermann
	reqBody := ActionRequest{
		Query:           "B-200",
		ActiveStudentID: &studentID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	const numRequests = 50
	runner := newStresstestRunner(baseURL, token, jsonData, numRequests)

	fmt.Printf("Starting stress test: 50 concurrent requests to %s\n", baseURL)

	runner.starteAlle()
	runner.gibStartsignal()
	runner.wg.Wait()

	runner.druckeErgebnisse()
}
