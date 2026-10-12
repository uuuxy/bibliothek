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
	"fmt"
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

// RouterAufruf liefert einen Aufruf über einen ganzen Router mit der Sitzung eines Kontos: der
// Rumpf als JSON, die Sitzung als Cookie, dazu der CSRF-Nachweis.
func RouterAufruf(router http.Handler, sitzung string) func(t *testing.T, methode, pfad, rumpf string) *httptest.ResponseRecorder {
	return func(t *testing.T, methode, pfad, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(methode, pfad, strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, MitCSRF(req))
		return rec
	}
}

// Bestelle schickt der Tür der Bestellung einen Warenkorb mit zwei Exemplaren eines Titels aus
// Mitteln des Landes, mit Nummern vorab, und liefert die Antwort.
func Bestelle(t *testing.T, tuer http.HandlerFunc, lieferantID, titelID string) *httptest.ResponseRecorder {
	t.Helper()
	rumpf := fmt.Sprintf(
		`{"supplier_id":%q,"mittel":"land","items":[{"titel_id":%q,"menge":2,"preis":9.5,"generate_barcodes":true}]}`,
		lieferantID, titelID)

	req := httptest.NewRequest(http.MethodPost, "/api/bestellungen", strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	tuer(rec, req)
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

// AntwortOhneDB ruft eine Tür, deren Server keine Datenbank hat, und liefert den Status der
// Antwort. Greift die Tür doch nach der Datenbank, bricht sie ab; der Helfer fängt das und
// meldet es als Fehler des Tests.
func AntwortOhneDB(t *testing.T, h http.HandlerFunc, methode, pfad, rumpf string) (status int) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s %s erreichte die Datenbank statt die Kennung zu prüfen (%v)", methode, pfad, r)
		}
	}()
	var body *strings.Reader
	if rumpf != "" {
		body = strings.NewReader(rumpf)
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(methode, pfad, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec.Code
}
