// Package clientip resolves the real client IP address of an HTTP request in a
// way that is safe behind a reverse proxy (e.g. Caddy in Docker).
//
// The rule: X-Forwarded-For and X-Real-IP are only believed when the request
// actually arrived from a configured trusted proxy. For requests that reach the
// backend directly, these headers are attacker-controlled and are ignored. When
// walking X-Forwarded-For we start at the right (the hop closest to us) and
// return the first address that is not itself a trusted proxy — that is the real
// client as our proxy saw it, and it cannot be spoofed by prepending fake
// left-hand entries.
package clientip

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
)

// loopbackCIDRs is the always-trusted baseline: only a loopback peer may set
// forwarding headers. It keeps local development and health checks working and
// is the safe default when no proxies are configured.
var loopbackCIDRs = []string{"127.0.0.0/8", "::1/128"}

// Resolver maps an incoming request to a client IP given a set of trusted proxy
// networks. It is safe for concurrent use.
type Resolver struct {
	trusted []*net.IPNet
}

// NewResolver builds a Resolver from CIDR strings or bare IPs (a bare IP is
// treated as a /32 or /128). Blank or unparsable entries are skipped.
func NewResolver(cidrs []string) *Resolver {
	rs := &Resolver{}
	for _, raw := range cidrs {
		c := strings.TrimSpace(raw)
		if c == "" {
			continue
		}
		if !strings.Contains(c, "/") {
			if strings.Contains(c, ":") {
				c += "/128"
			} else {
				c += "/32"
			}
		}
		if _, network, err := net.ParseCIDR(c); err == nil {
			rs.trusted = append(rs.trusted, network)
		}
	}
	return rs
}

// FromRequest returns the best-effort real client IP as a bare address (no port).
func (rs *Resolver) FromRequest(r *http.Request) string {
	peer := hostOnly(r.RemoteAddr)
	if !rs.isTrusted(peer) {
		// Direct connection from a non-proxy: forwarding headers are untrusted.
		return peer
	}
	// The immediate peer is a trusted proxy. Walk X-Forwarded-For from the
	// right (closest hop) to the left (original client) and return the first
	// entry that is not itself a trusted proxy.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(parts[i])
			if ip == "" {
				continue
			}
			if !rs.isTrusted(ip) {
				return ip
			}
		}
	}
	// No untrusted X-Forwarded-For entry: fall back to the single-valued
	// X-Real-IP set by the immediate proxy, otherwise the peer itself.
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	return peer
}

// isTrusted reports whether ipStr is a configured trusted proxy address.
func (rs *Resolver) isTrusted(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, network := range rs.trusted {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// hostOnly strips the port from a RemoteAddr, tolerating values that carry no
// port (e.g. a bare IP in tests or over a Unix socket).
func hostOnly(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

// defaultResolver is the process-wide resolver used by the package-level
// FromRequest. It starts trusting loopback only and is replaced at startup via
// Configure/ConfigureFromEnv.
var defaultResolver atomic.Pointer[Resolver]

func init() {
	defaultResolver.Store(NewResolver(loopbackCIDRs))
}

// Configure replaces the process-wide resolver. Loopback is always trusted in
// addition to the supplied networks. Call once during startup.
func Configure(cidrs []string) {
	combined := make([]string, 0, len(cidrs)+len(loopbackCIDRs))
	combined = append(combined, cidrs...)
	combined = append(combined, loopbackCIDRs...)
	defaultResolver.Store(NewResolver(combined))
}

// ConfigureFromEnv configures the process-wide resolver from the TRUSTED_PROXIES
// environment variable (comma-separated CIDRs or bare IPs). Unset or empty
// leaves loopback as the only trusted network.
func ConfigureFromEnv() {
	raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if raw == "" {
		Configure(nil)
		return
	}
	Configure(strings.Split(raw, ","))
}

// FromRequest resolves the client IP using the process-wide resolver.
func FromRequest(r *http.Request) string {
	return defaultResolver.Load().FromRequest(r)
}
