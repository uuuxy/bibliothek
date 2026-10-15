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

	"github.com/pashagolub/pgxmock/v5"
)

// Nur der Name der Tabelle: pgxmock vergleicht mit einem regulären Ausdruck, und die Liste der
// Spalten enthält ein coalesce(...), dessen Klammern dort Gruppen wären und den echten SQL-Text
// nicht mehr träfen.
const benutzerSelect = `FROM benutzer`

// aktiviereIMAPAttrappe schaltet die Attrappe des Mailservers für einen Test frei. APP_ENV gehört
// dazu: Die Attrappe nimmt jedes Passwort an und gilt deshalb nur in local, development und
// test. Ohne APP_ENV lehnte pruefeIMAPAnmeldung ab, und jeder Test der Anmeldung liefe ins
// Leere.
func aktiviereIMAPAttrappe(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "mock")
}

func rufeAnmelden(t *testing.T, a *Anmeldung, mock pgxmock.PgxPoolIface, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	anfrage := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(rumpf))
	anfrage.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	Anmelden(mock, a, false)(rec, anfrage)
	return rec
}

func TestAnmelden_FehlendeZugangsdatenSind400(t *testing.T) {
	aktiviereIMAPAttrappe(t)
	a, mock := newTestAnmeldung(t, 12*time.Hour)

	if rec := rufeAnmelden(t, a, mock, `{"password":"x"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("ohne email: erwartet 400, bekam %d", rec.Code)
	}
	if rec := rufeAnmelden(t, a, mock, `{"email":"a@b.de"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("ohne password: erwartet 400, bekam %d", rec.Code)
	}
}

func TestAnmelden_UnbekanntesKontoIst401(t *testing.T) {
	// Die Attrappe nimmt jedes Passwort an; die Ablehnung muss aus der Datenbank kommen (Postfach
	// vorhanden, aber kein Konto in der Bibliothek).
	aktiviereIMAPAttrappe(t)
	a, mock := newTestAnmeldung(t, 12*time.Hour)

	mock.ExpectQuery(benutzerSelect).
		WithArgs("unbekannt@schule.de").
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv", "email", "beantragt"}))

	rec := rufeAnmelden(t, a, mock, `{"email":"unbekannt@schule.de","password":"egal"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("erwartet 401, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("bei fehlgeschlagenem Login darf kein Cookie gesetzt werden")
	}
}

func TestAnmelden_DeaktiviertesKontoIst403(t *testing.T) {
	aktiviereIMAPAttrappe(t)
	a, mock := newTestAnmeldung(t, 12*time.Hour)

	mock.ExpectQuery(benutzerSelect).
		WithArgs("inaktiv@schule.de").
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv", "email", "beantragt"}).
			AddRow("u-1", "BC-TEST", "mitarbeiter", "Ex", "Kollege", false, "ex@example.org", false))

	rec := rufeAnmelden(t, a, mock, `{"email":"inaktiv@schule.de","password":"egal"}`)
	if rec.Code != http.StatusForbidden {
		t.Errorf("erwartet 403, bekam %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAnmelden_ErfolgSetztCookieUndAntwortet(t *testing.T) {
	aktiviereIMAPAttrappe(t)
	a, mock := newTestAnmeldung(t, 12*time.Hour)

	mock.ExpectQuery(benutzerSelect).
		WithArgs("nberger@schule.de").
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv", "email", "beantragt"}).
			AddRow("u-admin", "BC-TEST", "admin", "Nina", "Berger", true, "nina@example.org", false))
	erwarteSitzungAngelegt(mock, "u-admin")

	rec := rufeAnmelden(t, a, mock, `{"email":"nberger@schule.de","password":"egal"}`)
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

	var antwort anmeldeAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort ist kein JSON der Anmeldung: %v", err)
	}
	// Email gehört in die Antwort: Der Sperrbildschirm meldet damit nach dem Wiederherstellen der
	// Sitzung wieder an; ohne sie bliebe er für immer zu.
	if antwort.KontoID != "u-admin" || antwort.Rolle != RolleAdmin || antwort.Vorname != "Nina" || antwort.Email != "nina@example.org" {
		t.Errorf("Antwort der Anmeldung falsch: %+v", antwort)
	}
	if len(antwort.Rechte) != 1 || antwort.Rechte[0] != "*" {
		t.Errorf("Admin muss implizit '*' bekommen: %+v", antwort.Rechte)
	}

	// Das ausgestellte Token muss sich prüfen lassen und das Konto tragen.
	erwarteNichtWiderrufen(mock)
	erwarteKontoAktiv(mock, true)
	erwarteSperrzustand(mock, false)
	claims, err := a.PruefeToken(cookies[0].Value)
	if err != nil {
		t.Fatalf("ausgestelltes Token ungültig: %v", err)
	}
	if claims.KontoID != "u-admin" || claims.Rolle != RolleAdmin {
		t.Errorf("Claims falsch: %+v", claims)
	}
	if claims.SitzungID != testSitzungID {
		t.Errorf("SitzungID = %q, erwartet die Kennung der angelegten Zeile — ohne sie lässt sich die Anmeldung nicht sperren", claims.SitzungID)
	}
}

func TestAnmelden_BremseWeistDenSechstenVersuchAb(t *testing.T) {
	// Die Bremse zählt je E-Mail und Adresse: fünf Fehlversuche, dann 429. Die E-Mail ist eindeutig,
	// weil die Bremse für den ganzen Prozess gilt.
	aktiviereIMAPAttrappe(t)
	a, mock := newTestAnmeldung(t, 12*time.Hour)

	email := fmt.Sprintf("brute-%d@schule.de", time.Now().UnixNano())
	rumpf := fmt.Sprintf(`{"email":%q,"password":"falsch"}`, email)

	for i := 1; i <= 5; i++ {
		mock.ExpectQuery(benutzerSelect).
			WithArgs(email).
			WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv", "email", "beantragt"}))
		if rec := rufeAnmelden(t, a, mock, rumpf); rec.Code != http.StatusUnauthorized {
			t.Fatalf("Versuch %d: erwartet 401, bekam %d", i, rec.Code)
		}
	}

	// Der sechste Versuch wird abgewiesen, ohne dass die Datenbank noch gefragt wird.
	rec := rufeAnmelden(t, a, mock, rumpf)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("erwartet 429 nach 5 Fehlversuchen, bekam %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen (6. Versuch hätte die DB nicht erreichen dürfen): %v", err)
	}
}

// Die Ausweisnummer im Token der Sitzung kommt aus der Datenbank, nicht aus der Anmeldung.
//
// Eine Kennung, die der Browser mitschickt, darf nicht in die Claims wandern: Sonst bestimmte,
// wer sich anmeldet, selbst, welche Ausweisnummer sein signiertes Token behauptet, und der
// nächste Aufrufer verließe sich darauf, ohne nachzusehen.
//
// Der Test schickt die Kennung eines anderen mit und liest die Claims aus dem Cookie.
func TestAnmelden_AusweisnummerImTokenKommtAusDerDatenbank(t *testing.T) {
	aktiviereIMAPAttrappe(t)
	a, mock := newTestAnmeldung(t, 12*time.Hour)

	mock.ExpectQuery(benutzerSelect).
		WithArgs("nberger@schule.de").
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv", "email", "beantragt"}).
			AddRow("u-admin", "BC-ECHT", "admin", "Nina", "Berger", true, "nina@example.org", false))
	erwarteSitzungAngelegt(mock, "u-admin")

	rec := rufeAnmelden(t, a, mock,
		`{"email":"nberger@schule.de","password":"egal","barcode_id":"BC-FREMD","pin":"0000"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("kein Session-Cookie gesetzt")
	}
	// Die Nutzlast direkt lesen statt über PruefeToken: Das prüft zusätzlich Sperrliste und Stand
	// des Kontos in der Datenbank und bräuchte hier Erwartungen an die Attrappe, die mit der Frage
	// nichts zu tun haben. Es zählt, was im signierten Token steht.
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

// totePortAdresse reserviert einen freien TCP-Port und gibt ihn sofort wieder frei. Verbindungen
// dorthin scheitern mit „connection refused", ohne echtes Netz.
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

// Ein toter Mailserver ist kein falsches Passwort: 503 mit ehrlicher Meldung und kein
// Fehlversuch. Sonst sperrte sich, wer sein richtiges Passwort erneut probiert, nach fünf
// Versuchen für 15 Minuten aus. Kommt der Server zurück, gelingt die Anmeldung sofort.
func TestAnmelden_MailserverAusfallIstKeinFalschesPasswort(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	host, port := totePortAdresse(t)
	t.Setenv("IMAP_HOST", host)
	t.Setenv("IMAP_PORT", port)

	a, mock := newTestAnmeldung(t, 12*time.Hour)
	email := fmt.Sprintf("ausfall-%d@schule.de", time.Now().UnixNano())
	rumpf := fmt.Sprintf(`{"email":%q,"password":"richtig"}`, email)

	// Sechs Versuche gegen den toten Server: jedes Mal 503, nie 429, und die Datenbank wird nie
	// gefragt.
	for i := 1; i <= 6; i++ {
		rec := rufeAnmelden(t, a, mock, rumpf)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("Versuch %d: erwartet 503 bei totem Mailserver, bekam %d: %s", i, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "Mailserver") {
			t.Fatalf("Versuch %d: Meldung muss den Mailserver nennen, nicht das Passwort: %s", i, rec.Body.String())
		}
	}

	// Der Server kommt zurück (Attrappe): Die Anmeldung muss sofort gelingen, ohne 429, weil die
	// Versuche während des Ausfalls keinen Fehlversuch zählen durften.
	aktiviereIMAPAttrappe(t)
	mock.ExpectQuery(benutzerSelect).
		WithArgs(email).
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv", "email", "beantragt"}).
			AddRow("u-1", "BC-TEST", "admin", "Zurueck", "ImDienst", true, "zurueck@example.org", false))
	erwarteSitzungAngelegt(mock, "u-1")

	rec := rufeAnmelden(t, a, mock, rumpf)
	if rec.Code != http.StatusOK {
		t.Fatalf("nach Server-Rückkehr: erwartet 200, bekam %d — die Ausfall-Versuche haben den Nutzer gesperrt: %s", rec.Code, rec.Body.String())
	}
}

// Der Rumpf der Anmeldung ist begrenzt (anmeldeRumpfMaxBytes): Ohne eigene Grenze gälte die
// Grenze des Servers von 100 MB für die Importe, und wer nicht angemeldet ist, könnte je Aufruf
// einen Text von 100 MB in den Decoder schieben. Ohne die Grenze liefert der Test 401 (der Rumpf
// wird gelesen und als falsches Passwort behandelt) statt 400.
func TestAnmelden_UebergrosserRumpfWirdAbgewiesen(t *testing.T) {
	aktiviereIMAPAttrappe(t)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	a, err := NewAnmeldung("a-very-long-secret-key-of-at-least-32-bytes", mock, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	rumpf := `{"email":"x@example.test","password":"` + strings.Repeat("a", anmeldeRumpfMaxBytes) + `"}`
	rec := rufeAnmelden(t, a, mock, rumpf)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Status = %d; want 400 — der Rumpf darf nicht bis zur Anmeldung kommen", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("es darf kein Datenbankzugriff stattfinden: %v", err)
	}
}
