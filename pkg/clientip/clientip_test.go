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
	// Taking the rightmost untrusted entry must yield the true client, not the
	// attacker-controlled left value — this is the anti-spoofing property.
	rs := NewResolver([]string{"172.16.0.0/12"})

	got := rs.FromRequest(request("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "10.9.9.9, 198.51.100.7",
	}))
	if got != "198.51.100.7" {
		t.Fatalf("got %q; want the proxy-appended client 198.51.100.7", got)
	}
}

func TestResolver_ChainedTrustedProxiesSkipped(t *testing.T) {
	// Two trusted proxies in front (e.g. an edge proxy then Caddy). Both appear
	// in X-Forwarded-For; the resolver skips them and returns the client.
	rs := NewResolver([]string{"172.16.0.0/12", "192.0.2.0/24"})

	got := rs.FromRequest(request("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "203.0.113.9, 192.0.2.10, 172.18.0.9",
	}))
	if got != "203.0.113.9" {
		t.Fatalf("got %q; want the client 203.0.113.9", got)
	}
}

func TestResolver_AllForwardedEntriesTrustedFallsBackToRealIP(t *testing.T) {
	rs := NewResolver([]string{"172.16.0.0/12"})

	got := rs.FromRequest(request("172.18.0.5:5000", map[string]string{
		"X-Forwarded-For": "172.18.0.9",
		"X-Real-IP":       "198.51.100.7",
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
