package inventur

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// clientLimiter speichert den Rate-Limiter und den letzten Zugriffszeitpunkt pro IP.
type clientLimiter struct {
	limiter        *rate.Limiter
	letzterZugriff time.Time
}

// IPBasedRateLimiter verwaltet Rate-Limiter pro IP-Adresse mit automatischem Cleanup.
type IPBasedRateLimiter struct {
	sync.RWMutex
	clients     map[string]*clientLimiter
	r           rate.Limit
	b           int
	lebensdauer time.Duration
	stopCh      chan struct{}
}

// NewIPBasedRateLimiter erstellt einen neuen IP-basierten Rate-Limiter.
// r = Refill-Rate (Ereignisse pro Sekunde), b = Burst-Größe.
// Alte IP-Einträge werden automatisch nach 15 Minuten Inaktivität bereinigt.
func NewIPBasedRateLimiter(r rate.Limit, b int) *IPBasedRateLimiter {
	rl := &IPBasedRateLimiter{
		clients:     make(map[string]*clientLimiter),
		r:           r,
		b:           b,
		lebensdauer: 15 * time.Minute,
		stopCh:      make(chan struct{}),
	}

	go rl.bereinigungLoop()

	return rl
}

// Stop beendet die Bereinigung-Goroutine sauber (Graceful Shutdown).
func (rl *IPBasedRateLimiter) Stop() {
	close(rl.stopCh)
}

// bereinigungLoop entfernt inaktive IP-Einträge in regelmäßigen Abständen.
func (rl *IPBasedRateLimiter) bereinigungLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			rl.Lock()
			jetzt := time.Now()
			for ip, client := range rl.clients {
				if jetzt.Sub(client.letzterZugriff) > rl.lebensdauer {
					delete(rl.clients, ip)
				}
			}
			rl.Unlock()
		}
	}
}

// getLimiter gibt den Rate-Limiter für die angegebene IP-Adresse zurück.
func (rl *IPBasedRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.Lock()
	defer rl.Unlock()

	cl, exists := rl.clients[ip]
	if !exists {
		cl = &clientLimiter{
			limiter:        rate.NewLimiter(rl.r, rl.b),
			letzterZugriff: time.Now(),
		}
		rl.clients[ip] = cl
	} else {
		cl.letzterZugriff = time.Now()
	}

	return cl.limiter
}

func parseIP(value string) net.IP {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	host, _, err := net.SplitHostPort(trimmed)
	if err == nil {
		trimmed = host
	}

	return net.ParseIP(trimmed)
}

func istVertrauenswuerdigerProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// X-Real-IP wird nur von lokalen oder privaten Proxy-Hops akzeptiert.
	return ip.IsLoopback() || ip.IsPrivate()
}

// extrahiereClientIP ermittelt die Client-IP für das Rate-Limit.
// X-Real-IP wird nur akzeptiert, wenn der direkte Peer ein vertrauenswürdiger Proxy ist.
func extrahiereClientIP(request *http.Request) string {
	peerHost := request.RemoteAddr
	host, _, err := net.SplitHostPort(peerHost)
	if err == nil {
		peerHost = host
	}

	peerIP := parseIP(peerHost)
	if istVertrauenswuerdigerProxy(peerIP) {
		if xRealIP := parseIP(request.Header.Get("X-Real-IP")); xRealIP != nil {
			return xRealIP.String()
		}
	}

	if peerIP != nil {
		return peerIP.String()
	}

	return peerHost
}

// Middleware prüft das Rate-Limit pro IP und blockiert exzessive Anfragen.
func (rl *IPBasedRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ip := extrahiereClientIP(request)

		limiter := rl.getLimiter(ip)
		if !limiter.Allow() {
			writeError(writer, http.StatusTooManyRequests, "zu viele anfragen, bitte warte einen moment")
			return
		}

		next.ServeHTTP(writer, request)
	})
}
