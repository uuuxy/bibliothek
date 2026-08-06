package api

import (
	"bytes"
	"net/http/httptest"
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

func TestIstCoverCacheSchluessel(t *testing.T) {
	gueltig := []string{
		"9783161484100",     // ISBN-13
		"978-3-16-148410-0", // mit Trennstrichen
		"316148410X",        // ISBN-10 mit Prüfziffer X
		"3-16-148410-x",     // kleingeschrieben
		"12345678",          // kurze EAN/Bestellnummer
	}
	for _, s := range gueltig {
		t.Run("gültig/"+s, func(t *testing.T) {
			if !istCoverCacheSchluessel(s) {
				t.Errorf("istCoverCacheSchluessel(%q) = false, want true", s)
			}
		})
	}

	ungueltig := map[string]string{
		"leer":                 "",
		"Pfadanteil":           "../../etc/passwd",
		"Punkt im Namen":       "9783161484100.webp",
		"Buchstaben":           "beliebigername",
		"X nicht am Ende":      "31X148410",
		"zu kurz":              "1234567",
		"zu lang":              "123456789012345",
		"zu lang mit Strichen": "978-3-16-148410-00",
		"Leerzeichen":          "9783161 84100",
	}
	for name, s := range ungueltig {
		t.Run("ungültig/"+name, func(t *testing.T) {
			if istCoverCacheSchluessel(s) {
				t.Errorf("istCoverCacheSchluessel(%q) = true, want false", s)
			}
		})
	}
}

// Diese Fälle enden alle vor dem Dateisystem-Zugriff des Handlers — statt eines
// Fehlers muss das transparente Fallback-GIF kommen (kein Browser-Konsolen-Spam).
func TestServeCoverImage_FallbackOhneDownload(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"ISBN fehlt", "?url=https://covers.openlibrary.org/b/isbn/123-M.jpg"},
		{"URL fehlt", "?isbn=9783161484100"},
		{"Host nicht auf der Allowlist", "?isbn=9783161484100&url=https://evil.example/x.jpg"},
		{"Interne Ziel-URL", "?isbn=9783161484100&url=http://169.254.169.254/latest/meta-data/"},
		// Der Cache-Name wird zum Dateinamen — was keine Bestellnummer ist, kommt gar
		// nicht erst bis zum Dateisystem.
		{"Cache-Name ist keine Nummer", "?isbn=beliebiger-name&url=https://covers.openlibrary.org/b/isbn/1-M.jpg"},
		{"Cache-Name mit Pfadanteil", "?isbn=../../etc/passwd&url=https://covers.openlibrary.org/b/isbn/1-M.jpg"},
	}

	s := &Server{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/cover"+tt.query, nil)

			s.serveCoverImage(rec, req)

			if rec.Code != 200 {
				t.Errorf("Status = %d; want 200", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "image/gif" {
				t.Errorf("Content-Type = %q; want image/gif", ct)
			}
			if !bytes.Equal(rec.Body.Bytes(), coverFallbackGIF) {
				t.Errorf("Body ist nicht das Fallback-GIF (%d Bytes)", rec.Body.Len())
			}
		})
	}
}
