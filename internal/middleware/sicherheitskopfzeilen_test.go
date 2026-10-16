package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSicherheitskopfzeilen(t *testing.T) {
	// Der Handler dahinter antwortet mit 200 und „OK".
	dahinter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			t.Fatalf("Antwort schreiben: %v", err)
		}
	})

	handler := Sicherheitskopfzeilen(dahinter)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	erwartet := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		// base-uri fällt nicht auf default-src zurück: Ohne die Direktive böge ein
		// eingeschleustes <base href> jede relative URL der Oberfläche um. img-src steht ohne
		// https:, weil Bilder von fremden Hosts eingeschleustes Markup nach außen trügen;
		// Cover kommen über den eigenen Proxy.
		"Content-Security-Policy":   "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data: blob:; connect-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'self'; object-src 'none';",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Permissions-Policy":        "geolocation=(), microphone=(), camera=(self)",
	}

	for kopf, soll := range erwartet {
		t.Run(kopf, func(t *testing.T) {
			ist := rr.Header().Get(kopf)
			if ist == "" {
				t.Errorf("Kopfzeile %s fehlt", kopf)
			} else if ist != soll {
				t.Errorf("Kopfzeile %s = %q, erwartet %q", kopf, ist, soll)
			}
		})
	}

	// Der Handler dahinter ist gelaufen.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Status = %v, erwartet %v", status, http.StatusOK)
	}
	if inhalt := rr.Body.String(); inhalt != "OK" {
		t.Errorf("Inhalt = %q, erwartet %q", inhalt, "OK")
	}
}

// Antworten unter /api/ legt der Browser nicht ab. Die Programmdateien der Oberfläche bleiben
// unberührt, und ein Handler, der ablegen lassen will, behält seinen eigenen Kopf: Die Vorgabe
// steht vor ihm, nicht nach ihm.
func TestSicherheitskopfzeilen_SchnittstelleWirdNichtAbgelegt(t *testing.T) {
	faelle := []struct {
		name, pfad, eigenerKopf, erwartet string
	}{
		{"Liste", "/api/schueler", "", "no-store"},
		{"Auskunft als PDF", "/api/schueler/5f0c9a1e-2b7d-4c1a-9d3e-7a1b2c3d4e5f/dsgvo-auskunft/pdf", "", "no-store"},
		{"Oberfläche", "/", "", ""},
		{"Programmdatei", "/assets/index-4f2a.js", "", ""},
		{"Pfad beginnt nur mit api", "/apidoc", "", ""},
		{"eigener Kopf des Handlers", "/api/images/cover", "public, max-age=86400", "public, max-age=86400"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			handler := Sicherheitskopfzeilen(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if f.eigenerKopf != "" {
					w.Header().Set("Cache-Control", f.eigenerKopf)
				}
				w.WriteHeader(http.StatusOK)
			}))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://example.com"+f.pfad, nil))
			if got := rr.Header().Get("Cache-Control"); got != f.erwartet {
				t.Errorf("%s: Cache-Control %q, erwartet %q", f.pfad, got, f.erwartet)
			}
		})
	}
}
