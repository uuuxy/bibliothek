package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/pkg/clientip"
)

// failN feuert n Requests durch die Middleware, deren innerer Handler immer 401
// liefert (fehlgeschlagener Login), und gibt den letzten Statuscode zurück.
func failN(t *testing.T, remoteAddr, xff string, n int) int {
	t.Helper()
	handler := AuthRateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	last := 0
	for i := 0; i < n; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = remoteAddr
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		last = rec.Code
	}
	return last
}

func resetLoginState() {
	failedLoginsMutex.Lock()
	failedLogins = make(map[string]*failedAttempt)
	failedLoginsMutex.Unlock()
}

// burstN feuert n GET-Requests auf path durch den allgemeinen IP-Rate-Limiter
// (Limit=5, damit der Burst klein bleibt) und zählt, wie viele 429 zurückkamen.
func burstN(t *testing.T, path string, n int) (blocked int) {
	t.Helper()
	handler := RateLimitMiddleware(5)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for i := 0; i < n; i++ {
		req := httptest.NewRequest("GET", path, nil)
		req.RemoteAddr = "127.0.0.1:5000"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			blocked++
		}
	}
	return blocked
}

// Regressionstest für die 429-Flut der Etiketten-Vorschau: ein A4-Bogen rendert
// dutzende <img src="/api/barcode…"> gleichzeitig. /api/barcode muss — wie
// /api/images/cover — vom Bucket ausgenommen sein, sonst blockiert ein einziger
// Seitenaufruf einen Teil seiner eigenen Barcodes.
func TestRateLimit_BarcodeBurstIsExempt(t *testing.T) {
	clientip.Configure(nil) // nur Loopback
	t.Cleanup(func() { clientip.Configure(nil) })

	// 40 Barcodes in einem Rutsch, Bucket=5 → ohne Ausnahme wären 35 geblockt.
	if got := burstN(t, "/api/barcode?content=112622&width=150&height=50", 40); got != 0 {
		t.Fatalf("/api/barcode: %d Requests mit 429; want 0 (Endpoint ist vom Rate-Limiter ausgenommen)", got)
	}

	// Gegenprobe: ein normaler API-Pfad wird nach dem 5. Token weiter limitiert.
	if got := burstN(t, "/api/books", 40); got == 0 {
		t.Fatal("/api/books: 0 Requests mit 429; want >0 (normale Pfade bleiben limitiert)")
	}

	// /api/barcode/next (JSON, kein Fan-out) bleibt ebenfalls limitiert — der
	// exakte Pfad-Match darf den Unterpfad nicht mit ausnehmen.
	if got := burstN(t, "/api/barcode/next", 40); got == 0 {
		t.Fatal("/api/barcode/next: 0 Requests mit 429; want >0 (nur exakt /api/barcode ist ausgenommen)")
	}
}

// Der eigentliche DoS-Regressionstest: Hinter einem vertrauenswürdigen Proxy
// müssen zwei verschiedene echte Clients unabhängige Fehlversuchszähler haben.
// Vor dem Fix wurde auf r.RemoteAddr (= Proxy) gekeyt, sodass fünf Fehlversuche
// IRGENDEINES Nutzers alle aussperrten.
func TestAuthRateLimit_ProxyClientsAreCountedIndependently(t *testing.T) {
	clientip.Configure([]string{"172.16.0.0/12"})
	t.Cleanup(func() { clientip.Configure(nil) })
	resetLoginState()

	const proxy = "172.18.0.5:40000"

	// Angreifer verbrennt seine 5 Versuche und wird gesperrt.
	if got := failN(t, proxy, "203.0.113.9", 5); got != http.StatusUnauthorized {
		t.Fatalf("5. Versuch des Angreifers: Status %d; want 401 (noch nicht gesperrt)", got)
	}
	if got := failN(t, proxy, "203.0.113.9", 1); got != http.StatusTooManyRequests {
		t.Fatalf("6. Versuch des Angreifers: Status %d; want 429 (gesperrt)", got)
	}

	// Ein anderer, unbeteiligter Nutzer hinter demselben Proxy darf sich
	// weiterhin einloggen — sein Zähler ist eigenständig.
	if got := failN(t, proxy, "198.51.100.7", 1); got != http.StatusUnauthorized {
		t.Fatalf("unbeteiligter Nutzer: Status %d; want 401 (nicht mitgesperrt)", got)
	}
}

// Ein direkt (nicht über einen vertrauten Proxy) verbundener Angreifer darf sich
// nicht durch gefälschte X-Forwarded-For-Header eine frische Identität erschummeln.
func TestAuthRateLimit_UntrustedPeerCannotSpoofXFF(t *testing.T) {
	clientip.Configure([]string{"172.16.0.0/12"})
	t.Cleanup(func() { clientip.Configure(nil) })
	resetLoginState()

	const attacker = "203.0.113.50:40000"

	// Trotz rotierender, gefälschter XFF-Werte zählt nur die echte Peer-IP.
	failN(t, attacker, "1.1.1.1", 3)
	failN(t, attacker, "2.2.2.2", 2)
	if got := failN(t, attacker, "3.3.3.3", 1); got != http.StatusTooManyRequests {
		t.Fatalf("Angreifer nach 6 Versuchen: Status %d; want 429 (XFF-Spoof wirkungslos)", got)
	}
}

func TestAuthRateLimit_SuccessResetsCounter(t *testing.T) {
	clientip.Configure(nil) // nur Loopback
	t.Cleanup(func() { clientip.Configure(nil) })
	resetLoginState()

	handler := AuthRateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /login mit korrektem Passwort → 200
		if r.Header.Get("X-Test-Outcome") == "ok" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))

	do := func(outcome string) int {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "127.0.0.1:5000"
		if outcome != "" {
			req.Header.Set("X-Test-Outcome", outcome)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	do("")   // 1 Fehlversuch
	do("")   // 2
	do("ok") // Erfolg setzt den Zähler zurück
	for i := 0; i < 5; i++ {
		do("") // wieder 5 Fehlversuche nötig
	}
	if got := do(""); got != http.StatusTooManyRequests {
		t.Fatalf("nach Reset + 5 Fehlversuchen: Status %d; want 429", got)
	}
}
