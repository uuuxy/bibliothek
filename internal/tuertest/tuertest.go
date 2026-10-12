// Package tuertest hält, was Tests von Türen brauchen, um eine Anfrage zu stellen und die
// Antwort zu lesen: den CSRF-Nachweis, eine Sitzung an der Anfrage, Attrappen fremder Dienste,
// einen Mailserver, der abfängt, was eine Tür verschickt.
//
// Wie internal/pgtest ist es ein gewöhnliches Paket, das nur Tests einbinden: Eine Testdatei
// lässt sich nicht aus einem anderen Paket einbinden, und die Tests der Türen stehen in mehr
// als einem. Die deadcode-Baseline führt seine Funktionen deshalb mit Begründung.
package tuertest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
)

// MitCSRF rüstet den Double-Submit nach, den der Router bei ändernden Methoden verlangt: Cookie
// und Kopfzeile mit demselben Wert.
func MitCSRF(req *http.Request) *http.Request {
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "pii-gate-csrf"})
	req.Header.Set("X-CSRF-Token", "pii-gate-csrf")
	return req
}

// JSONPost baut eine POST-Anfrage mit JSON-Rumpf und CSRF-Nachweis.
func JSONPost(url, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return MitCSRF(req)
}

// AlsBenutzer hängt die Sitzung des Benutzers an die Anfrage; ohne Benutzer bleibt sie
// unangemeldet.
func AlsBenutzer(req *http.Request, benutzerID string) *http.Request {
	if benutzerID == "" {
		return req
	}
	return req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: benutzerID, Rolle: auth.RoleAdmin}))
}

// RufeAlsAdmin ruft einen Handler mit der Sitzung des Admin-Kontos, ohne den Router davor.
func RufeAlsAdmin(t *testing.T, admin string, h http.HandlerFunc, methode, pfad, body string, pfadwerte map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(methode, pfad, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range pfadwerte {
		req.SetPathValue(k, v)
	}
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: admin, Rolle: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

// Fehlermeldung liest den Text aus einer Fehlerantwort der Türen.
func Fehlermeldung(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Antwort ist kein JSON-Fehler (%s): %v", rec.Body.String(), err)
	}
	return body.Error
}
