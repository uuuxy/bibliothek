// Package ratelimit bremst Anfragen je Adresse: ein Vorrat je Adresse, der sich mit der Zeit
// wieder füllt. Die Middleware des Servers nimmt es für jede Anfrage, der Abruf der Cover für
// die Fehltreffer, die er bei fremden Diensten nachfragt.
package ratelimit

import (
	"sync"
	"time"
)

type visitor struct {
	tokens     float64
	lastRefill time.Time
}

// Limiter führt je Adresse einen Vorrat von limit Anfragen, der sich in einer Sekunde wieder
// füllt.
type Limiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int // max requests per second
}

// New liefert eine Bremse mit limit Anfragen je Sekunde und Adresse.
func New(limit int) *Limiter {
	return &Limiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
	}
}

func (l *Limiter) cleanupStaleVisitors(now time.Time) {
	if len(l.visitors) <= 5000 {
		return
	}
	for k, vis := range l.visitors {
		if now.Sub(vis.lastRefill) > 5*time.Minute {
			delete(l.visitors, k)
		}
	}
}

// Allow checks if the IP is allowed to perform a request under token bucket rules.
func (l *Limiter) Allow(ip string) bool {
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
		l.cleanupStaleVisitors(now)

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
