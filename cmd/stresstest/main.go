// Command stresstest schickt gleichzeitige Scans an die Theke eines lokalen Servers und zählt
// die Antworten nach ihrem Status (docs/SCRIPTS.md, Abschnitt 5).
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

// konfig hält die Werte aus der .env.
type konfig struct {
	port         string
	jwtGeheimnis string
}

// leseKonfig liest PORT und JWT_SECRET aus der .env des Projekts oder, ersatzweise, des
// Arbeitsverzeichnisses.
func leseKonfig() konfig {
	inhalt, err := os.ReadFile("../../.env")
	if err != nil {
		inhalt, err = os.ReadFile(".env")
		if err != nil {
			log.Fatalf("Could not read .env file: %v", err)
		}
	}

	konfigWerte := konfig{port: "8080", jwtGeheimnis: ""}
	zeilen := strings.Split(string(inhalt), "\n")
	for _, zeile := range zeilen {
		zeile = strings.TrimSpace(zeile)
		if strings.HasPrefix(zeile, "PORT=") {
			konfigWerte.port = strings.TrimPrefix(zeile, "PORT=")
		} else if strings.HasPrefix(zeile, "JWT_SECRET=") {
			konfigWerte.jwtGeheimnis = strings.TrimPrefix(zeile, "JWT_SECRET=")
		}
	}

	if konfigWerte.jwtGeheimnis == "" {
		log.Fatalf("JWT_SECRET not found in .env")
	}

	return konfigWerte
}

// erzeugeAdminToken stellt ein Token für das Admin-Konto der Testdaten aus, eine Stunde gültig.
func erzeugeAdminToken(geheimnis string) string {
	claims := jwt.MapClaims{
		// Die Kennung des Admin-Kontos aus seed.sql.
		"user_id":    "00000000-0000-0000-0000-000000000001",
		"barcode_id": "admin",
		"rolle":      "ADMIN",
		"exp":        time.Now().Add(1 * time.Hour).Unix(),
		"iat":        time.Now().Unix(),
		"nbf":        time.Now().Unix(),
		"iss":        "bibliothek-system",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signiert, err := token.SignedString([]byte(geheimnis))
	if err != nil {
		log.Fatalf("Failed to sign token: %v", err)
	}

	return signiert
}

// aktionsAnfrage ist der Rumpf eines Scans an der Theke (POST /api/action).
type aktionsAnfrage struct {
	Suchtext       string  `json:"query"`
	AktiverLeserID *string `json:"active_leser_id,omitempty"`
}

// lastlauf bündelt den geteilten Zustand der Goroutinen: die Barriere für den Start (damit alle
// Anfragen genau gleichzeitig losgehen) und die gesammelten Zähler je Status.
type lastlauf struct {
	zielURL string
	rumpf   []byte
	token   string
	anzahl  int

	alle        sync.WaitGroup
	startSperre sync.Mutex
	startSignal *sync.Cond
	bereit      sync.WaitGroup
	start       bool

	ergebnisSperre sync.Mutex
	jeStatus       map[int]int
}

// newLastlauf baut den Lauf und meldet die Anfragen bei der Startbarriere an.
func newLastlauf(zielURL, token string, rumpf []byte, anzahl int) *lastlauf {
	r := &lastlauf{
		zielURL:  zielURL,
		token:    token,
		rumpf:    rumpf,
		anzahl:   anzahl,
		jeStatus: make(map[int]int),
	}
	r.startSignal = sync.NewCond(&r.startSperre)
	r.bereit.Add(anzahl)
	return r
}

// baueAnfrage baut die Anfrage POST mit Kopfzeilen und den Cookies für CSRF und Sitzung.
func (r *lastlauf) baueAnfrage() (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPost, r.zielURL, bytes.NewReader(r.rumpf))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Derselbe Wert in Kopfzeile und Cookie besteht die Prüfung auf CSRF.
	csrfToken := "dummy-csrf-token-12345"
	req.Header.Set("X-CSRF-Token", csrfToken)

	// Das Cookie der Sitzung.
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

// warteAufStart meldet die Goroutine als bereit und wartet, bis das Startsignal kommt.
func (r *lastlauf) warteAufStart() {
	r.bereit.Done()

	r.startSperre.Lock()
	for !r.start {
		r.startSignal.Wait()
	}
	r.startSperre.Unlock()
}

// erfasseErgebnis schickt die Anfrage und zählt den Status oder den Fehler des Netzes.
func (r *lastlauf) erfasseErgebnis(client *http.Client, req *http.Request) {
	antwort, err := client.Do(req)
	if err != nil {
		// Fehler des Netzes: Der Status bleibt 0. Der erste Fehler wird ausgegeben.
		r.ergebnisSperre.Lock()
		if len(r.jeStatus) == 0 {
			fmt.Printf("\n[DEBUG] Network error details: %v\n", err)
		}
		r.jeStatus[0]++
		r.ergebnisSperre.Unlock()
		return
	}
	status := antwort.StatusCode
	_, _ = io.Copy(io.Discard, antwort.Body) //nolint:errcheck
	_ = antwort.Body.Close()                 //nolint:errcheck

	r.ergebnisSperre.Lock()
	r.jeStatus[status]++
	r.ergebnisSperre.Unlock()
}

// sendeEineAnfrage ist der Rumpf einer Goroutine: Anfrage bauen, auf das gemeinsame Startsignal
// warten und dann schicken.
func (r *lastlauf) sendeEineAnfrage() {
	defer r.alle.Done()

	client := &http.Client{
		Timeout: 10 * time.Second,
		// Jede Goroutine hat ihren eigenen Client mit eigenem Transport, damit die Anfragen über
		// eigene Verbindungen und wirklich gleichzeitig laufen.
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
	}

	req, err := r.baueAnfrage()
	if err != nil {
		r.ergebnisSperre.Lock()
		r.jeStatus[-1]++ // -1: Die Anfrage ließ sich nicht bauen.
		r.ergebnisSperre.Unlock()
		return
	}

	r.warteAufStart()
	r.erfasseErgebnis(client, req)
}

// starteAlle startet anzahl Goroutinen, die zunächst an der Barriere warten.
func (r *lastlauf) starteAlle() {
	r.alle.Add(r.anzahl)
	for i := 0; i < r.anzahl; i++ {
		go r.sendeEineAnfrage()
	}
}

// gibStartsignal wartet, bis alle Goroutinen bereitstehen, und weckt sie gleichzeitig.
func (r *lastlauf) gibStartsignal() {
	r.bereit.Wait()

	r.startSperre.Lock()
	r.start = true
	r.startSignal.Broadcast()
	r.startSperre.Unlock()
}

// druckeErgebnisse schreibt die Zahl der Antworten je Status nach aus.
func (r *lastlauf) druckeErgebnisse(aus io.Writer) {
	fmt.Fprintln(aus, "\n--- Stress Test Results ---") //nolint:errcheck // Ausgabe auf der Konsole
	for code, count := range r.jeStatus {
		if code == 0 {
			fmt.Fprintf(aus, "%dx Network Error (Failed to execute request)\n", count) //nolint:errcheck // Ausgabe auf der Konsole
		} else {
			statusText := http.StatusText(code)
			fmt.Fprintf(aus, "%dx %d %s\n", count, code, statusText) //nolint:errcheck // Ausgabe auf der Konsole
		}
	}
}

func main() {
	portSchalter := flag.String("port", "", "Port to run the stress test against (overrides .env)")
	flag.Parse()

	konfigWerte := leseKonfig()
	port := konfigWerte.port
	if *portSchalter != "" {
		port = *portSchalter
	}

	token := erzeugeAdminToken(konfigWerte.jwtGeheimnis)
	zielURL := fmt.Sprintf("http://127.0.0.1:%s/api/action", port)

	// Testdaten aus seed.sql.
	leserID := "00000000-0000-0000-0000-000000000003" // Max Mustermann
	anfrageDaten := aktionsAnfrage{
		Suchtext:       "B-200",
		AktiverLeserID: &leserID,
	}

	rumpf, err := json.Marshal(anfrageDaten)
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	const anzahl = 50
	lauf := newLastlauf(zielURL, token, rumpf, anzahl)

	fmt.Printf("Starting stress test: 50 concurrent requests to %s\n", zielURL)

	lauf.starteAlle()
	lauf.gibStartsignal()
	lauf.alle.Wait()

	lauf.druckeErgebnisse(os.Stdout)
}
