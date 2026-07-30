package api

import (
	"net/http"
	"strings"
	"testing"
)

// CodeQL meldet an jeder Cookie-Stelle `go/cookie-secure-not-set`, weil Secure kein
// literales true ist, sondern aus der Konfiguration kommt. Der Melder kann den
// Laufzeitwert nicht beweisen — wir können ihn prüfen.
//
// Das ist der Beleg, mit dem die Meldungen abgewiesen werden: Secure folgt der
// Konfiguration in beide Richtungen. Die Ausnahme (false) ist gewollt und in
// ermittleCookieSecure begründet — ein Secure-Cookie würde über reines HTTP im
// Schulnetz vom Browser gar nicht erst mitgeschickt, der Login wäre unmöglich.
// Die sichere Vorgabe stellt cookie_secure_test.go sicher.
func TestCSRFCookieFolgtDerKonfiguration(t *testing.T) {
	for _, secure := range []bool{true, false} {
		c := newCSRFCookie("tok123", secure)
		if c.Secure != secure {
			t.Errorf("Secure = %v, erwartet %v", c.Secure, secure)
		}
	}
}

// TestCSRFCookieAttribute hält die übrigen Schutzattribute fest. HttpOnly ist hier
// bewusst false — das Frontend muss den Wert lesen, um ihn als X-CSRF-Token
// zurückzuschicken. Genau deshalb trägt dieses Cookie auch keinen Sitzungszustand.
func TestCSRFCookieAttribute(t *testing.T) {
	c := newCSRFCookie("tok123", true)

	if c.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %v, erwartet Strict", c.SameSite)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, erwartet /", c.Path)
	}
	if c.HttpOnly {
		t.Error("HttpOnly muss false bleiben — das Frontend liest den Wert für den X-CSRF-Token-Header")
	}
	if c.Value != "tok123" {
		t.Errorf("Value = %q", c.Value)
	}
}

// TestCSRFCookieLandetInDerAntwort schließt die Lücke zwischen „das Struct stimmt" und
// „der Browser bekommt es": geprüft wird die tatsächlich gesetzte Set-Cookie-Kopfzeile.
func TestCSRFCookieLandetInDerAntwort(t *testing.T) {
	rec := &kopfzeilenRecorder{header: http.Header{}}
	http.SetCookie(rec, newCSRFCookie("tok123", true))

	gesetzt := rec.header.Get("Set-Cookie")
	for _, teil := range []string{"tok123", "Secure", "SameSite=Strict", "Path=/"} {
		if !strings.Contains(gesetzt, teil) {
			t.Errorf("Set-Cookie enthält %q nicht: %s", teil, gesetzt)
		}
	}
}

// kopfzeilenRecorder ist ein minimaler ResponseWriter, der nur die Kopfzeilen sammelt.
type kopfzeilenRecorder struct{ header http.Header }

func (r *kopfzeilenRecorder) Header() http.Header         { return r.header }
func (r *kopfzeilenRecorder) Write(b []byte) (int, error) { return len(b), nil }
func (r *kopfzeilenRecorder) WriteHeader(int)             {}
