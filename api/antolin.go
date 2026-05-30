package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"bibliothek/apierrors"
)

// AntolinResult holds a single Antolin lookup result and its cache timestamp.
type AntolinResult struct {
	Found    bool    `json:"found"`
	Stufen   string  `json:"stufen,omitempty"` // e.g. "3-6"
	Punkte   float64 `json:"punkte,omitempty"`
	cachedAt time.Time
}

var (
	antolinCache    sync.Map // map[string]*AntolinResult
	antolinCacheTTL = 24 * time.Hour
)

// antolinAPIResp matches the JSON response from antolin.de.
type antolinAPIResp struct {
	Antwort []struct {
		ISBN    string  `json:"isbn"`
		Titel   string  `json:"titel"`
		Klassen string  `json:"klassen"`
		Punkte  float64 `json:"punkte"`
	} `json:"antwort"`
}

// AntolinHandler handles GET /api/antolin?isbn=...
// Proxies to antolin.de (24-hour in-memory cache). Public endpoint.
func (s *Server) AntolinHandler() http.HandlerFunc {
	client := &http.Client{Timeout: 5 * time.Second}
	return func(w http.ResponseWriter, r *http.Request) {
		isbn := strings.ReplaceAll(strings.TrimSpace(r.URL.Query().Get("isbn")), "-", "")
		if isbn == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("isbn parameter required"))
			return
		}

		// Cache hit
		if cached, ok := antolinCache.Load(isbn); ok {
			entry := cached.(*AntolinResult)
			if time.Since(entry.cachedAt) < antolinCacheTTL {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(entry)
				return
			}
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet,
			"https://www.antolin.de/all/jsonBuecher.do?isbn="+isbn, nil)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(&AntolinResult{Found: false})
			return
		}
		req.Header.Set("User-Agent", "Schulbibliothek/1.0")

		result := &AntolinResult{cachedAt: time.Now()}
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			var apiResp antolinAPIResp
			if json.NewDecoder(resp.Body).Decode(&apiResp) == nil && len(apiResp.Antwort) > 0 {
				b := apiResp.Antwort[0]
				result.Found = true
				result.Stufen = b.Klassen
				result.Punkte = b.Punkte
			}
		}

		antolinCache.Store(isbn, result)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}
