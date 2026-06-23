package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pkg/closeutil"
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
			if entry, isResult := cached.(*AntolinResult); isResult && time.Since(entry.cachedAt) < antolinCacheTTL {
				RespondJSON(w, http.StatusOK, entry)
				return
			}
		}

		// #nosec G107 - URL wird sicher aus internen Const/Whitelist generiert
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet,
			"https://www.antolin.de/all/jsonBuecher.do?isbn="+url.QueryEscape(isbn), nil)
		if err != nil {
			log.Printf("Antolin Request Creation Error: %v", err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to create request"))
			return
		}
		req.Header.Set("User-Agent", "Schulbibliothek/1.0")

		result := &AntolinResult{cachedAt: time.Now()}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Antolin API Error: %v", err)
			apierrors.SendHTTPError(w, http.StatusBadGateway, fmt.Errorf("antolin service unavailable"))
			return
		}
		defer closeutil.LogClose(resp.Body, "antolin response body")

		var apiResp antolinAPIResp
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			log.Printf("Antolin JSON Decode Error: %v", err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to parse antolin response"))
			return
		}

		if len(apiResp.Antwort) > 0 {
			b := apiResp.Antwort[0]
			result.Found = true
			result.Stufen = b.Klassen
			result.Punkte = b.Punkte
		}

		antolinCache.Store(isbn, result)
		RespondJSON(w, http.StatusOK, result)
	}
}
