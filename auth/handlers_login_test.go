package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// Bewusst nur der Tabellenteil: pgxmock vergleicht per REGEX, und die Spaltenliste
// enthält seit dem Barcode-aus-der-DB ein coalesce(...) — die Klammern wären dort
// Regex-Gruppen und träfen den echten SQL-Text nicht mehr.
const benutzerSelect = `FROM benutzer`

// aktiviereMockIMAP schaltet den IMAP-Mock für einen Test frei. APP_ENV gehört
// zwingend dazu: Der Mock akzeptiert jedes Passwort und ist deshalb seit dem
// Audit-Fix nur noch in local/development/test erlaubt — ohne APP_ENV würde
// AuthenticateIMAP hier ablehnen und jeder Login-Test liefe ins Leere.
func aktiviereMockIMAP(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "mock")
}

func doLogin(t *testing.T, a *Authenticator, mock pgxmock.PgxPoolIface, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	LoginHandler(mock, a, false)(rec, req)
	return rec
}

func TestLoginHandler_MissingCredentialsReturn400(t *testing.T) {
	aktiviereMockIMAP(t)
	a, mock := newTestAuthenticator(t, 12*time.Hour)

	if rec := doLogin(t, a, mock, `{"password":"x"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("ohne email: erwartet 400, bekam %d", rec.Code)
	}
	if rec := doLogin(t, a, mock, `{"email":"a@b.de"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("ohne password: erwartet 400, bekam %d", rec.Code)
	}
}

func TestLoginHandler_UnknownUserReturns401(t *testing.T) {
	// Mock-IMAP akzeptiert jedes Passwort — die Ablehnung muss aus der DB kommen
	// (IMAP-Konto existiert, aber kein registrierter Bibliotheks-Benutzer).
	aktiviereMockIMAP(t)
	a, mock := newTestAuthenticator(t, 12*time.Hour)

	mock.ExpectQuery(benutzerSelect).
		WithArgs("unbekannt@schule.de").
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv"}))

	rec := doLogin(t, a, mock, `{"email":"unbekannt@schule.de","password":"egal"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("erwartet 401, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("bei fehlgeschlagenem Login darf kein Cookie gesetzt werden")
	}
}

func TestLoginHandler_DeactivatedUserReturns403(t *testing.T) {
	aktiviereMockIMAP(t)
	a, mock := newTestAuthenticator(t, 12*time.Hour)

	mock.ExpectQuery(benutzerSelect).
		WithArgs("inaktiv@schule.de").
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv"}).
			AddRow("u-1", "BC-TEST", "mitarbeiter", "Ex", "Kollege", false))

	rec := doLogin(t, a, mock, `{"email":"inaktiv@schule.de","password":"egal"}`)
	if rec.Code != http.StatusForbidden {
		t.Errorf("erwartet 403, bekam %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLoginHandler_SuccessSetsCookieAndReturnsLoginShape(t *testing.T) {
	aktiviereMockIMAP(t)
	a, mock := newTestAuthenticator(t, 12*time.Hour)

	mock.ExpectQuery(benutzerSelect).
		WithArgs("pflasch@schule.de").
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv"}).
			AddRow("u-admin", "BC-TEST", "admin", "Peter", "Flasch", true))

	rec := doLogin(t, a, mock, `{"email":"pflasch@schule.de","password":"egal"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session_token" || cookies[0].Value == "" {
		t.Fatalf("erwartet session_token-Cookie, bekam %+v", cookies)
	}
	if !cookies[0].HttpOnly {
		t.Errorf("Session-Cookie muss HttpOnly sein")
	}

	var resp LoginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Antwort kein LoginResponse-JSON: %v", err)
	}
	if resp.UserID != "u-admin" || resp.Rolle != RoleAdmin || resp.Vorname != "Peter" {
		t.Errorf("LoginResponse falsch: %+v", resp)
	}
	if len(resp.Permissions) != 1 || resp.Permissions[0] != "*" {
		t.Errorf("Admin muss implizit '*' bekommen: %+v", resp.Permissions)
	}

	// Das ausgestellte Token muss verifizierbar sein und die Identität tragen.
	expectNotBlacklisted(mock)
	expectKontoAktiv(mock, true)
	claims, err := a.VerifyToken(cookies[0].Value)
	if err != nil {
		t.Fatalf("ausgestelltes Token ungültig: %v", err)
	}
	if claims.UserID != "u-admin" || claims.Rolle != RoleAdmin {
		t.Errorf("Claims falsch: %+v", claims)
	}
}

func TestLoginHandler_BruteForceLimiterBlocksSixthAttempt(t *testing.T) {
	// Der Limiter drosselt pro (E-Mail|IP) — 5 Fehlversuche, dann 429.
	// Eindeutige E-Mail, da der Limiter prozessweit global ist.
	aktiviereMockIMAP(t)
	a, mock := newTestAuthenticator(t, 12*time.Hour)

	email := fmt.Sprintf("brute-%d@schule.de", time.Now().UnixNano())
	body := fmt.Sprintf(`{"email":%q,"password":"falsch"}`, email)

	for i := 1; i <= 5; i++ {
		mock.ExpectQuery(benutzerSelect).
			WithArgs(email).
			WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv"}))
		if rec := doLogin(t, a, mock, body); rec.Code != http.StatusUnauthorized {
			t.Fatalf("Versuch %d: erwartet 401, bekam %d", i, rec.Code)
		}
	}

	// 6. Versuch: geblockt, OHNE dass die DB noch gefragt wird
	rec := doLogin(t, a, mock, body)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("erwartet 429 nach 5 Fehlversuchen, bekam %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen (6. Versuch hätte die DB nicht erreichen dürfen): %v", err)
	}
}

// Der Barcode im Session-Token kommt aus der DATENBANK, nicht aus der Anmeldung.
//
// Vorher trug LoginRequest ein Feld barcode_id, und dessen Wert wanderte ungeprüft in die
// Claims: Wer sich anmeldete, bestimmte selbst, welche Ausweisnummer sein signiertes
// Token behauptet. Ausgewertet hat den Wert niemand — ein Loch war es also nicht. Aber
// eine signierte Kennung, die nie jemand geprüft hat, ist genau die Art Zusicherung, auf
// die sich der nächste Aufrufer verlässt, ohne nachzusehen.
//
// Der Test schickt die Kennung eines anderen mit und liest die Claims aus dem Cookie.
func TestLoginHandler_BarcodeImTokenKommtAusDerDatenbank(t *testing.T) {
	aktiviereMockIMAP(t)
	a, mock := newTestAuthenticator(t, 12*time.Hour)

	mock.ExpectQuery(benutzerSelect).
		WithArgs("pflasch@schule.de").
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv"}).
			AddRow("u-admin", "BC-ECHT", "admin", "Peter", "Flasch", true))

	rec := doLogin(t, a, mock,
		`{"email":"pflasch@schule.de","password":"egal","barcode_id":"BC-FREMD","pin":"0000"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("kein Session-Cookie gesetzt")
	}
	// Nutzlast direkt lesen statt über VerifyToken: Das prüft zusätzlich Sperrliste und
	// Kontostatus gegen die DB und bräuchte hier Mock-Erwartungen, die mit der Frage
	// nichts zu tun haben. Interessiert, WAS im signierten Token steht.
	teile := strings.Split(cookies[0].Value, ".")
	if len(teile) != 3 {
		t.Fatalf("kein JWT im Cookie: %q", cookies[0].Value)
	}
	roh, err := base64.RawURLEncoding.DecodeString(teile[1])
	if err != nil {
		t.Fatalf("Nutzlast nicht dekodierbar: %v", err)
	}
	var claims struct {
		BarcodeID string `json:"barcode_id"`
	}
	if err := json.Unmarshal(roh, &claims); err != nil {
		t.Fatalf("Nutzlast nicht lesbar: %v", err)
	}
	if claims.BarcodeID == "BC-FREMD" {
		t.Fatal("die vom Client behauptete Ausweisnummer steht im signierten Token — " +
			"der Barcode gehört aus der benutzer-Tabelle geladen, nicht aus der Anfrage")
	}
	if claims.BarcodeID != "BC-ECHT" {
		t.Errorf("Barcode im Token = %q, erwartet den aus der Datenbank (BC-ECHT)", claims.BarcodeID)
	}
}

// totePortAdresse reserviert einen freien TCP-Port und gibt ihn sofort wieder frei —
// Verbindungen dorthin scheitern mit „connection refused", ohne echtes Netz.
func totePortAdresse(t *testing.T) (host, port string) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listener: %v", err)
	}
	host, port, err = net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatalf("SplitHostPort: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return host, port
}

// TestLoginHandler_MailserverAusfallIstKeinFalschesPasswort schließt die
// Massen-Selbstsperre aus der Ausfallmatrix (20.08.2026): Ein toter Mailserver
// wurde vorher als „invalid email or password" (401) gemeldet UND als Fehlversuch
// gezählt — wer sein richtiges Passwort daraufhin erneut probierte, sperrte sich
// nach fünf Versuchen für 15 Minuten selbst aus. Jetzt: 503 mit ehrlicher Meldung,
// KEIN Fehlversuch — nach Rückkehr des Servers klappt der Login sofort.
func TestLoginHandler_MailserverAusfallIstKeinFalschesPasswort(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	host, port := totePortAdresse(t)
	t.Setenv("IMAP_HOST", host)
	t.Setenv("IMAP_PORT", port)

	a, mock := newTestAuthenticator(t, 12*time.Hour)
	email := fmt.Sprintf("ausfall-%d@schule.de", time.Now().UnixNano())
	body := fmt.Sprintf(`{"email":%q,"password":"richtig"}`, email)

	// 6 Versuche gegen den toten Server: jedes Mal 503, nie 429 — die DB wird nie gefragt.
	for i := 1; i <= 6; i++ {
		rec := doLogin(t, a, mock, body)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("Versuch %d: erwartet 503 bei totem Mailserver, bekam %d: %s", i, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "Mailserver") {
			t.Fatalf("Versuch %d: Meldung muss den Mailserver nennen, nicht das Passwort: %s", i, rec.Body.String())
		}
	}

	// Server „kommt zurück" (Mock): Der Login muss SOFORT gelingen — kein 429,
	// weil die Ausfall-Versuche keinen Fehlversuch gezählt haben dürfen.
	aktiviereMockIMAP(t)
	mock.ExpectQuery(benutzerSelect).
		WithArgs(email).
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv"}).
			AddRow("u-1", "BC-TEST", "admin", "Zurueck", "ImDienst", true))

	rec := doLogin(t, a, mock, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("nach Server-Rückkehr: erwartet 200, bekam %d — die Ausfall-Versuche haben den Nutzer gesperrt: %s", rec.Code, rec.Body.String())
	}
}
