package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Die Middleware setzt das CSRF-Cookie bei jeder Anfrage an die Schnittstelle ohne Cookie
// (erneuereCSRFCookie). Der Test liest es so, wie der Browser es bekommt; baute die Middleware ihr
// Cookie selbst statt über newCSRFCookie, fiele ein fehlendes Secure sonst nicht auf.
func TestCSRFCookieDerMiddlewareNurUeberHTTPS(t *testing.T) {
	antwort := httptest.NewRecorder()
	(&Server{}).csrfMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(antwort, httptest.NewRequest(http.MethodGet, "/api/books", nil))
	cookies := antwort.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != csrfCookieName {
		t.Fatalf("Cookies %v, erwartet das CSRF-Cookie", cookies)
	}
	if !cookies[0].Secure {
		t.Error("Das CSRF-Cookie der Middleware trägt kein Secure")
	}
}

// TestCSRFCookieAttribute hält die Schutzattribute fest. HttpOnly ist hier bewusst false: Die
// Oberfläche muss den Wert lesen, um ihn als X-CSRF-Token zurückzuschicken. Genau deshalb trägt
// dieses Cookie auch keinen Sitzungszustand.
func TestCSRFCookieAttribute(t *testing.T) {
	c := newCSRFCookie("tok123")

	if !c.Secure {
		t.Error("Secure fehlt — der Browser schickte das Cookie auch über HTTP")
	}

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
	http.SetCookie(rec, newCSRFCookie("tok123"))

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
