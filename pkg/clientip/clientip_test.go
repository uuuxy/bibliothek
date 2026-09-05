package clientip

import (
	"net/http"
	"testing"
)

func request(remoteAddr string, headers map[string]string) *http.Request {
	r := &http.Request{RemoteAddr: remoteAddr, Header: http.Header{}}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

func TestResolver_UntrustedPeerIgnoresForwardingHeaders(t *testing.T) {
	// Only loopback is trusted. A request arriving directly from a public peer
	// must never let X-Forwarded-For / X-Real-IP override the real connection —
	// otherwise an attacker who reaches the backend directly spoofs any IP.
	rs := NewResolver(loopbackCIDRs)

	got := rs.FromRequest(request("203.0.113.9:5000", map[string]string{
		"X-Forwarded-For": "1.2.3.4",
		"X-Real-IP":       "5.6.7.8",
	}))
	if got != "203.0.113.9" {
		t.Fatalf("got %q; want the real peer 203.0.113.9", got)
	}
}

func TestResolver_LoopbackPeerTrustsForwardedFor(t *testing.T) {
	rs := NewResolver(loopbackCIDRs)

	got := rs.FromRequest(request("127.0.0.1:5000", map[string]string{
		"X-Forwarded-For": "198.51.100.7",
	}))
	if got != "198.51.100.7" {
		t.Fatalf("got %q; want forwarded client 198.51.100.7", got)
	}
}

func TestResolver_TrustedProxyReturnsRealClient(t *testing.T) {
	// Caddy sits in the Docker network 172.16/12 and appends the real client.
	rs := NewResolver([]string{"172.16.0.0/12"})

	got := rs.FromRequest(request("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "198.51.100.7",
	}))
	if got != "198.51.100.7" {
		t.Fatalf("got %q; want 198.51.100.7", got)
	}
}

func TestResolver_SpoofedLeftmostEntryIsIgnored(t *testing.T) {
	// The client prepends a fake entry; Caddy appends the true peer to the right.
	// Only the rightmost entry counts — the attacker-controlled left value never does.
	rs := NewResolver([]string{"172.16.0.0/12"})

	got := rs.FromRequest(request("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "10.9.9.9, 198.51.100.7",
	}))
	if got != "198.51.100.7" {
		t.Fatalf("got %q; want the proxy-appended client 198.51.100.7", got)
	}
}

// Der Fund vom 05.09.2026: Die Schul-Clients liegen SELBST in den Netzen, die der
// Compose-Default als Proxy-Netze führt (172.16/12, 10/8, 192.168/16). Der alte
// rekursive Lauf übersprang sie als "Proxy" und fiel auf Caddys Adresse zurück — alle
// Geräte der Schule wurden für Login-Limiter, Rate-Limiter und Audit-Log EIN Client.
// Mit dem alten Code liefert dieser Test 172.18.0.5 für beide (rot gesehen).
func TestResolver_LanClientHinterProxy(t *testing.T) {
	rs := NewResolver([]string{"172.16.0.0/12", "10.0.0.0/8", "192.168.0.0/16"})

	a := rs.FromRequest(request("172.18.0.5:5000", map[string]string{"X-Forwarded-For": "192.168.1.50"}))
	b := rs.FromRequest(request("172.18.0.5:5000", map[string]string{"X-Forwarded-For": "192.168.1.77"}))
	if a != "192.168.1.50" || b != "192.168.1.77" {
		t.Fatalf("LAN-Clients aufgelöst zu %q und %q; want 192.168.1.50 und 192.168.1.77", a, b)
	}
}

// Zweite Hälfte desselben Funds: Reicht der Proxy den eingehenden Header durch und
// hängt die echte Adresse an, durfte der Client mit dem alten Lauf seine IP frei wählen,
// sobald seine echte Adresse in einem vertrauten Netz lag (alter Code: 8.8.8.8).
func TestResolver_DurchgereichterHeaderIstNichtWaehlbar(t *testing.T) {
	rs := NewResolver([]string{"172.16.0.0/12", "10.0.0.0/8", "192.168.0.0/16"})

	got := rs.FromRequest(request("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "8.8.8.8, 192.168.1.50",
	}))
	if got != "192.168.1.50" {
		t.Fatalf("got %q; want die vom Proxy angehängte 192.168.1.50", got)
	}
}

// Ein zweiter Proxy vor Caddy wird NICHT übersprungen: Genau ein Hop ist vertraut, und
// ein Netz-Treffer taugt nicht als Unterscheidung zwischen Proxy und Client (siehe
// Paketkommentar). Wer eine Kette betreibt, sieht hier den Edge-Proxy — sichtbar,
// nicht still falsch.
func TestResolver_KetteLiefertLetztenHop(t *testing.T) {
	rs := NewResolver([]string{"172.16.0.0/12", "192.0.2.0/24"})

	got := rs.FromRequest(request("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "203.0.113.9, 192.0.2.10",
	}))
	if got != "192.0.2.10" {
		t.Fatalf("got %q; want den letzten Hop 192.0.2.10", got)
	}
}

func TestResolver_OhneForwardedForFaelltAufRealIPZurueck(t *testing.T) {
	rs := NewResolver([]string{"172.16.0.0/12"})

	got := rs.FromRequest(request("172.18.0.5:5000", map[string]string{
		"X-Real-IP": "198.51.100.7",
	}))
	if got != "198.51.100.7" {
		t.Fatalf("got %q; want X-Real-IP fallback 198.51.100.7", got)
	}
}

func TestResolver_TrustedProxyNoHeadersReturnsPeer(t *testing.T) {
	rs := NewResolver([]string{"172.16.0.0/12"})

	got := rs.FromRequest(request("172.18.0.5:5000", nil))
	if got != "172.18.0.5" {
		t.Fatalf("got %q; want peer 172.18.0.5", got)
	}
}

func TestResolver_BareIPAndBlankEntries(t *testing.T) {
	rs := NewResolver([]string{"172.16.0.0/12"})

	// RemoteAddr without a port, and a stray blank X-Forwarded-For token.
	got := rs.FromRequest(request("172.18.0.5", map[string]string{
		"X-Forwarded-For": "198.51.100.7, ",
	}))
	if got != "198.51.100.7" {
		t.Fatalf("got %q; want 198.51.100.7", got)
	}
}

func TestConfigureFromEnv(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "172.16.0.0/12")
	ConfigureFromEnv()
	t.Cleanup(func() {
		defaultResolver.Store(NewResolver(loopbackCIDRs))
	})

	// Trusted Docker peer → forwarded client is honored.
	if got := FromRequest(request("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "198.51.100.7",
	})); got != "198.51.100.7" {
		t.Fatalf("configured trust: got %q; want 198.51.100.7", got)
	}
	// Loopback is always trusted in addition to the configured range.
	if got := FromRequest(request("127.0.0.1:5000", map[string]string{
		"X-Forwarded-For": "203.0.113.1",
	})); got != "203.0.113.1" {
		t.Fatalf("loopback baseline: got %q; want 203.0.113.1", got)
	}
	// A public peer that is not a configured proxy is never trusted.
	if got := FromRequest(request("203.0.113.9:5000", map[string]string{
		"X-Forwarded-For": "1.2.3.4",
	})); got != "203.0.113.9" {
		t.Fatalf("untrusted peer: got %q; want 203.0.113.9", got)
	}
}
