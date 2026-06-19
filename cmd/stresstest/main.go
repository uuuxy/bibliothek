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
	var wg sync.WaitGroup
	wg.Add(numRequests)

	// To make sure all goroutines start at exactly the same time
	var startMu sync.Mutex
	startCond := sync.NewCond(&startMu)
	readyCount := 0
	start := false

	// Result collection
	var resultsMu sync.Mutex
	statusCounts := make(map[int]int)

	fmt.Printf("Starting stress test: 50 concurrent requests to %s\n", baseURL)

	for i := 0; i < numRequests; i++ {
		go func(workerID int) {
			defer wg.Done()

			client := &http.Client{
				Timeout: 10 * time.Second,
				// Wir schalten HTTP/2 aus und zwingen das Programm neue Connections aufzubauen,
				// um die Parallelität bei manchen OS/Go-Versionen sicherzustellen.
				Transport: &http.Transport{
					MaxIdleConnsPerHost: 100,
				},
			}

			req, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewReader(jsonData))
			if err != nil {
				resultsMu.Lock()
				statusCounts[-1]++ // -1 means request creation failed
				resultsMu.Unlock()
				return
			}

			req.Header.Set("Content-Type", "application/json")

			// Dummy CSRF Token protection bypass
			csrfToken := "dummy-csrf-token-12345"
			req.Header.Set("X-CSRF-Token", csrfToken)

			// Session cookie
			req.AddCookie(&http.Cookie{
				Name:  "session_token",
				Value: token,
			})
			req.AddCookie(&http.Cookie{
				Name:  "csrf_token",
				Value: csrfToken,
			})

			// Wait for start signal
			startMu.Lock()
			readyCount++
			if readyCount == numRequests {
				startCond.Broadcast() // Wake up everyone if this is the last one
			}
			for !start {
				startCond.Wait()
			}
			startMu.Unlock()

			// Fire!
			resp, err := client.Do(req)
			statusCode := 0
			if err != nil {
				statusCode = 0 // network error

				// Log the first error for debugging
				resultsMu.Lock()
				if len(statusCounts) == 0 {
					fmt.Printf("\n[DEBUG] Network error details: %v\n", err)
				}
				statusCounts[0]++
				resultsMu.Unlock()
				return
			} else {
				statusCode = resp.StatusCode
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				resultsMu.Lock()
				statusCounts[statusCode]++
				resultsMu.Unlock()
			}
		}(i)
	}

	// Give goroutines a moment to spin up and wait on condition
	time.Sleep(100 * time.Millisecond)

	startMu.Lock()
	if readyCount < numRequests {
		startCond.Wait()
	}
	start = true
	startCond.Broadcast()
	startMu.Unlock()

	// Wait for all to finish
	wg.Wait()

	fmt.Println("\n--- Stress Test Results ---")
	for code, count := range statusCounts {
		if code == 0 {
			fmt.Printf("%dx Network Error (Failed to execute request)\n", count)
		} else {
			statusText := http.StatusText(code)
			fmt.Printf("%dx %d %s\n", count, code, statusText)
		}
	}
}
