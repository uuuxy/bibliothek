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

	rec := rufeAnmelden(t, a, mock, `{"password":"x"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("ohne email = %d, erwartet 400", rec.Code)
	}
	if got, want := fehlermeldungAus(t, rec), "Bitte die E-Mail-Adresse eingeben"; got != want {
		t.Errorf("ohne email: Meldung %q, erwartet %q", got, want)
	}
	rec = rufeAnmelden(t, a, mock, `{"email":"a@b.de"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("ohne password = %d, erwartet 400", rec.Code)
	}
	if got, want := fehlermeldungAus(t, rec), "Bitte das Passwort eingeben"; got != want {
		t.Errorf("ohne password: Meldung %q, erwartet %q", got, want)
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
		t.Errorf("rec.Code = %d, erwartet 401: %s", rec.Code, rec.Body.String())
	}
	if got, want := fehlermeldungAus(t, rec), "E-Mail-Adresse oder Passwort stimmt nicht"; got != want {
		t.Errorf("Anmelden: Meldung %q, erwartet %q", got, want)
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
		t.Errorf("rec.Code = %d, erwartet 403: %s", rec.Code, rec.Body.String())
	}
	const erwartet = "Konto deaktiviert. Bitte an die Bibliothek wenden."
	if got := fehlermeldungAus(t, rec); got != erwartet {
		t.Errorf("Anmelden: Meldung %q, erwartet %q", got, erwartet)
	}
}

// fehlermeldungAus liest den Text einer Fehlerantwort.
func fehlermeldungAus(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var antwort struct {
		Fehler string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort ist kein JSON-Fehler (%s): %v", rec.Body.String(), err)
	}
	return antwort.Fehler
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
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session_token" || cookies[0].Value == "" {
		t.Fatalf("rec.Result().Cookies() = %+v, erwartet session_token-Cookie", cookies)
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
	claims, err := a.PruefeToken(t.Context(), cookies[0].Value)
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

// scheitereFuenfMal meldet fünfmal mit falschem Passwort an einem Konto an, das es nicht gibt;
// jeder Versuch zählt als Fehlversuch.
func scheitereFuenfMal(t *testing.T, a *Anmeldung, mock pgxmock.PgxPoolIface, email string) {
	t.Helper()
	rumpf := fmt.Sprintf(`{"email":%q,"password":"falsch"}`, email)
	for i := 1; i <= 5; i++ {
		mock.ExpectQuery(benutzerSelect).
			WithArgs(email).
			WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv", "email", "beantragt"}))
		if rec := rufeAnmelden(t, a, mock, rumpf); rec.Code != http.StatusUnauthorized {
			t.Fatalf("Versuch %d = %d, erwartet 401", i, rec.Code)
		}
	}
}

func TestAnmelden_BremseWeistDenSechstenVersuchAb(t *testing.T) {
	// Die Bremse zählt je E-Mail und Adresse: fünf Fehlversuche, dann 429.
	aktiviereIMAPAttrappe(t)
	a, mock := newTestAnmeldung(t, 12*time.Hour)

	const email = "brute@schule.de"
	rumpf := fmt.Sprintf(`{"email":%q,"password":"falsch"}`, email)
	scheitereFuenfMal(t, a, mock, email)

	// Der sechste Versuch wird abgewiesen, ohne dass die Datenbank noch gefragt wird.
	rec := rufeAnmelden(t, a, mock, rumpf)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("rec.Code = %d, erwartet 429 nach 5 Fehlversuchen", rec.Code)
	}
	if got, want := fehlermeldungAus(t, rec), "Zu viele fehlgeschlagene Anmeldungen. Bitte 15 Minuten warten."; got != want {
		t.Errorf("Anmelden: Meldung %q, erwartet %q", got, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen (6. Versuch hätte die DB nicht erreichen dürfen): %v", err)
	}
}

// Die Bremse gehört der Anmeldung, die der Start einmal anlegt, nicht dem Paket: Eine zweite
// Anmeldung beginnt bei null. Teilten sich beide einen Zähler, hinge das Ergebnis eines Tests
// davon ab, welcher Test vor ihm dieselbe E-Mail gesperrt hat.
func TestAnmelden_BremseGehoertDerAnmeldung(t *testing.T) {
	aktiviereIMAPAttrappe(t)
	const email = "geteilt@schule.de"
	erste, ersterMock := newTestAnmeldung(t, 12*time.Hour)
	scheitereFuenfMal(t, erste, ersterMock, email)

	zweite, zweiterMock := newTestAnmeldung(t, 12*time.Hour)
	zweiterMock.ExpectQuery(benutzerSelect).
		WithArgs(email).
		WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "rolle", "vorname", "nachname", "aktiv", "email", "beantragt"}))
	rec := rufeAnmelden(t, zweite, zweiterMock, fmt.Sprintf(`{"email":%q,"password":"falsch"}`, email))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Anmelden an der zweiten Anmeldung: Status %d, erwartet 401; die Fehlversuche der ersten zählen dort nicht", rec.Code)
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
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
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
			t.Fatalf("Versuch %d = %d, erwartet 503 bei totem Mailserver: %s", i, rec.Code, rec.Body.String())
		}
		const erwartet = "Der Mailserver ist nicht erreichbar. Die Anmeldung geht gerade nicht, bitte später erneut versuchen."
		if got := fehlermeldungAus(t, rec); got != erwartet {
			t.Fatalf("Versuch %d: Meldung %q, erwartet %q", i, got, erwartet)
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
		t.Fatalf("nach Server-Rückkehr: Status = %d, erwartet 200 — die Ausfall-Versuche haben den Nutzer gesperrt: %s", rec.Code, rec.Body.String())
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
		t.Fatalf("Status = %d; erwartet 400 — der Rumpf darf nicht bis zur Anmeldung kommen", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("es darf kein Datenbankzugriff stattfinden: %v", err)
	}
}
