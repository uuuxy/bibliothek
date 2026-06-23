package api

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"bibliothek/apierrors"
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

// getIP extracts the client IP address from the request.
// X-Forwarded-For and X-Real-IP are only trusted when the direct connection
// comes from a loopback address (i.e. behind Caddy reverse proxy).
func getIP(r *http.Request) string {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr ohne Port (z. B. Unix-Socket): unverändert übernehmen
		remoteIP = r.RemoteAddr
	}
	if isLoopback(remoteIP) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				return strings.TrimSpace(parts[0])
			}
		}
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return ip
		}
	}
	if remoteIP != "" {
		return remoteIP
	}
	return r.RemoteAddr
}

// isLoopback checks if an IP string is a loopback address (trusted proxy).
func isLoopback(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip == "localhost"
	}
	return parsed.IsLoopback()
}

func RateLimitMiddleware(limit int) func(http.Handler) http.Handler {
	limiter := newIPRateLimiter(limit)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bilder und Uploads vom Rate-Limiter ausschließen, da ein Seitenaufruf oft dutzende Bilder gleichzeitig lädt.
			// /events (SSE) ist eine langlebige Verbindung, die bei jedem (Re-)Connect sonst einen Token verbraucht
			// und flaky Clients unnötig ausbremst.
			if strings.HasPrefix(r.URL.Path, "/api/images/cover") || strings.HasPrefix(r.URL.Path, "/uploads/") || r.URL.Path == "/events" {
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
