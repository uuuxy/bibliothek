package api

import (
	"net/http"
	"net/http/httptest"
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

// Das Cookie, das jede Anfrage an die Schnittstelle ohne Cookie bekommt, setzt die Middleware selbst
// (erneuereCSRFCookie). Es folgt der Konfiguration wie das der Tür csrf-token; ohne diesen Test
// blieb ein festes false an dieser Stelle unbemerkt.
func TestCSRFCookieDerMiddlewareFolgtDerKonfiguration(t *testing.T) {
	for _, secure := range []bool{true, false} {
		s := &Server{nurHTTPS: secure}
		antwort := httptest.NewRecorder()
		s.csrfMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
			ServeHTTP(antwort, httptest.NewRequest(http.MethodGet, "/api/books", nil))
		cookies := antwort.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Name != csrfCookieName {
			t.Fatalf("secure=%v: Cookies %v, erwartet das CSRF-Cookie", secure, cookies)
		}
		if cookies[0].Secure != secure {
			t.Errorf("secure=%v: Secure = %v — das Cookie der Middleware folgt der Konfiguration nicht", secure, cookies[0].Secure)
		}
	}
}

// TestCSRFCookieAttribute hält die übrigen Schutzattribute fest. HttpOnly ist hier
// bewusst false: Die Oberfläche muss den Wert lesen, um ihn als X-CSRF-Token
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
	rec := &kopfzeilenMerker{header: http.Header{}}
	http.SetCookie(rec, newCSRFCookie("tok123", true))

	gesetzt := rec.header.Get("Set-Cookie")
	for _, teil := range []string{"tok123", "Secure", "SameSite=Strict", "Path=/"} {
		if !strings.Contains(gesetzt, teil) {
			t.Errorf("Set-Cookie enthält %q nicht: %s", teil, gesetzt)
		}
	}
}

// kopfzeilenMerker ist ein minimaler ResponseWriter, der nur die Kopfzeilen sammelt.
type kopfzeilenMerker struct{ header http.Header }

func (m *kopfzeilenMerker) Header() http.Header         { return m.header }
func (m *kopfzeilenMerker) Write(b []byte) (int, error) { return len(b), nil }
func (m *kopfzeilenMerker) WriteHeader(int)             {}
