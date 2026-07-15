package api

import (
	"testing"
)

func TestBaueSichereCoverURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		ok       bool
	}{
		{
			"Erlaubter Host bleibt unverändert",
			"https://covers.openlibrary.org/b/isbn/9783161484100-M.jpg",
			"https://covers.openlibrary.org/b/isbn/9783161484100-M.jpg",
			true,
		},
		{
			"HTTP wird auf HTTPS gehoben, Query bleibt erhalten",
			"http://books.google.com/books/content?id=abc&printsec=frontcover",
			"https://books.google.com/books/content?id=abc&printsec=frontcover",
			true,
		},
		{
			"Leerer Pfad wird zu /",
			"https://openlibrary.org",
			"https://openlibrary.org/",
			true,
		},
		{
			"Expliziter Port wird verworfen",
			"https://covers.openlibrary.org:8080/b/isbn/123-M.jpg",
			"https://covers.openlibrary.org/b/isbn/123-M.jpg",
			true,
		},
		{"Fremder Host", "https://evil.example/x.jpg", "", false},
		{"Allowlist-Host als Subdomain eines Angreifers", "https://covers.openlibrary.org.evil.example/x.jpg", "", false},
		{"Allowlist-Host als Userinfo vor Angreifer-Host", "https://covers.openlibrary.org@evil.example/x.jpg", "", false},
		{"IP statt Hostname", "https://127.0.0.1/x.jpg", "", false},
		{"Leerer String", "", "", false},
		{"Kaputte URL", "https://%zz/x.jpg", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := baueSichereCoverURL(tt.input)
			if ok != tt.ok {
				t.Fatalf("baueSichereCoverURL(%q) ok = %v; want %v", tt.input, ok, tt.ok)
			}
			if result != tt.expected {
				t.Errorf("baueSichereCoverURL(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestVerbieteInterneZieladressen(t *testing.T) {
	tests := []struct {
		name    string
		address string
		blocked bool
	}{
		{"Öffentliche IPv4", "93.184.216.34:443", false},
		{"Öffentliche IPv6", "[2606:2800:220:1:248:1893:25c8:1946]:443", false},
		{"Loopback IPv4", "127.0.0.1:443", true},
		{"Loopback IPv6", "[::1]:443", true},
		{"Loopback als IPv4-in-IPv6", "[::ffff:127.0.0.1]:443", true},
		{"Privat 10/8", "10.0.0.5:80", true},
		{"Privat 172.16/12", "172.16.0.1:80", true},
		{"Privat 192.168/16", "192.168.1.10:443", true},
		{"Link-Local (Cloud-Metadaten)", "169.254.169.254:80", true},
		{"IPv6 Unique Local", "[fd00::1]:443", true},
		{"Unspezifiziert", "0.0.0.0:80", true},
		{"Keine IP", "example.com:80", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verbieteInterneZieladressen("tcp4", tt.address, nil)
			if blocked := err != nil; blocked != tt.blocked {
				t.Errorf("verbieteInterneZieladressen(%q) blocked = %v (err: %v); want %v", tt.address, blocked, err, tt.blocked)
			}
		})
	}
}
