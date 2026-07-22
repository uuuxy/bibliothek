package api

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pkg/clientip"
)

type visitor struct {
	tokens     float64
	lastRefill time.Time
}

type ipRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int // max requests per second
}

func newIPRateLimiter(limit int) *ipRateLimiter {
	return &ipRateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
	}
}

// allow checks if the IP is allowed to perform a request under token bucket rules.
func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	v, exists := l.visitors[ip]
	if !exists {
		v = &visitor{
			tokens:     float64(l.limit),
			lastRefill: now,
		}
		l.visitors[ip] = v

		// Clean up stale visitor entries to prevent memory leaks
		if len(l.visitors) > 5000 {
			for k, vis := range l.visitors {
				if now.Sub(vis.lastRefill) > 5*time.Minute {
					delete(l.visitors, k)
				}
			}
		}

		v.tokens -= 1.0
		return true
	}

	elapsed := now.Sub(v.lastRefill)
	v.lastRefill = now

	// Refill tokens proportional to elapsed time
	v.tokens += elapsed.Seconds() * float64(l.limit)
	if v.tokens > float64(l.limit) {
		v.tokens = float64(l.limit)
	}

	if v.tokens >= 1.0 {
		v.tokens -= 1.0
		return true
	}

	return false
}

// getIP extracts the client IP address from the request via the shared,
// trusted-proxy-aware resolver (see pkg/clientip). Used for both rate limiting
// and audit-log attribution.
func getIP(r *http.Request) string {
	return clientip.FromRequest(r)
}

func RateLimitMiddleware(limit int) func(http.Handler) http.Handler {
	limiter := newIPRateLimiter(limit)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bilder und Uploads vom Rate-Limiter ausschließen, da ein Seitenaufruf oft dutzende Bilder gleichzeitig lädt.
			// /api/barcode gehört genau in diese Klasse: die Etiketten-Vorschau (LabelPreview.svelte) rendert einen
			// kompletten A4-Bogen (Standard 52 = 52 Etiketten) und feuert pro Etikett ein <img src="/api/barcode…">
			// nahezu gleichzeitig ab — das sprengt sonst das 50-Requests/s-Bucket und liefert 429 (authentifiziert,
			// view_books, Antwort 1 Jahr cachebar, winzige PNGs). Exakter Pfad-Match, damit /api/barcode/next (JSON)
			// weiter limitiert bleibt.
			// /events (SSE) ist eine langlebige Verbindung, die bei jedem (Re-)Connect sonst einen Token verbraucht
			// und flaky Clients unnötig ausbremst.
			if strings.HasPrefix(r.URL.Path, "/api/images/cover") || strings.HasPrefix(r.URL.Path, "/uploads/") || r.URL.Path == "/api/barcode" || r.URL.Path == "/events" {
				next.ServeHTTP(w, r)
				return
			}

			ip := getIP(r)
			if !limiter.allow(ip) {
				apierrors.SendHTTPError(w, http.StatusTooManyRequests, errors.New("rate limit exceeded"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
