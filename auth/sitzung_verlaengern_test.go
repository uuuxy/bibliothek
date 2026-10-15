package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v5"
)

const testSecret = "test-secret-with-at-least-32-bytes!!"

// newTestAnmeldung baut eine Anmeldung auf einer Attrappe der Datenbank. PruefeToken fragt die
// Sperrliste mit einem SELECT; die Erwartungen dafür setzt jeder Test selbst.
func newTestAnmeldung(t *testing.T, laufzeit time.Duration) (*Anmeldung, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	t.Cleanup(mock.Close)
	a, err := NewAnmeldung(testSecret, mock, laufzeit)
	if err != nil {
		t.Fatalf("NewAnmeldung: %v", err)
	}
	t.Cleanup(a.Sperrliste.Stoppe)
	t.Cleanup(a.Sitzungen.Stoppe)
	return a, mock
}

// testSitzungID ist die Kennung, die die Attrappe für eine neue Zeile in sitzungen liefert.
const testSitzungID = "5b0f6a2e-8a0c-4a51-9d1c-3f2f6b7c9e10"

// erwarteSitzungAngelegt erwartet die Zeile, die jede gelungene Anmeldung anlegt.
func erwarteSitzungAngelegt(mock pgxmock.PgxPoolIface, kontoID string) {
	mock.ExpectQuery(`INSERT INTO sitzungen`).
		WithArgs(kontoID, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(testSitzungID))
}

// erwarteSitzungVerlaengert erwartet, dass die Erneuerung des Tokens das Ende der Zeile
// hinausschiebt; zeilen=0 heißt: Es gibt keine Zeile.
func erwarteSitzungVerlaengert(mock pgxmock.PgxPoolIface, zeilen int64) {
	mock.ExpectExec(`UPDATE sitzungen SET laeuft_ab`).
		WithArgs(testSitzungID, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", zeilen))
}

// erwarteSperrzustand erwartet die Abfrage der Sperre in PruefeToken.
func erwarteSperrzustand(mock pgxmock.PgxPoolIface, gesperrt bool) {
	mock.ExpectQuery(`SELECT gesperrt_seit IS NOT NULL FROM sitzungen`).
		WithArgs(testSitzungID).
		WillReturnRows(pgxmock.NewRows([]string{"gesperrt"}).AddRow(gesperrt))
}

func erwarteNichtWiderrufen(mock pgxmock.PgxPoolIface) {
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
}

// erwarteKontoAktiv erwartet die Prüfung von aktiv in PruefeToken (nach der Sperrliste, nur bei
// gültigem Token) und meldet den gewünschten Stand des Kontos zurück.
func erwarteKontoAktiv(mock pgxmock.PgxPoolIface, aktiv bool) {
	erwarteKontoStatus(mock, aktiv, RolleAdmin)
}

// erwarteKontoStatus erwartet die Abfrage des Kontos in PruefeToken und liefert dazu die Rolle,
// die dort die im Token signierte ersetzt.
func erwarteKontoStatus(mock pgxmock.PgxPoolIface, aktiv bool, rolle Rolle) {
	mock.ExpectQuery(`SELECT aktiv, rolle FROM benutzer`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"aktiv", "rolle"}).AddRow(aktiv, string(rolle)))
}

func rufeSitzungVerlaengern(t *testing.T, a *Anmeldung, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	anfrage := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	if cookie != nil {
		anfrage.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	SitzungVerlaengern(a, false)(rec, anfrage)
	return rec
}

func TestSitzungVerlaengern_OhneCookieIst401(t *testing.T) {
	a, _ := newTestAnmeldung(t, 12*time.Hour)
	rec := rufeSitzungVerlaengern(t, a, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("erwartet 401, bekam %d", rec.Code)
	}
}

func TestSitzungVerlaengern_UngueltigesTokenIst401(t *testing.T) {
	a, mock := newTestAnmeldung(t, 12*time.Hour)
	erwarteNichtWiderrufen(mock)
	rec := rufeSitzungVerlaengern(t, a, &http.Cookie{Name: "session_token", Value: "kein-jwt"})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("erwartet 401, bekam %d", rec.Code)
	}
}

func TestSitzungVerlaengern_FrischesTokenBleibtOhneNeuesCookie(t *testing.T) {
	// Frisch ausgestelltes Token: Es bleibt mehr als die halbe Laufzeit, also keine
	// Erneuerung.
	a, mock := newTestAnmeldung(t, 12*time.Hour)
	token, err := a.ErzeugeToken("user-1", "B-1", RolleAdmin, "")
	if err != nil {
		t.Fatalf("ErzeugeToken: %v", err)
	}
	erwarteNichtWiderrufen(mock)
	erwarteKontoAktiv(mock, true)

	rec := rufeSitzungVerlaengern(t, a, &http.Cookie{Name: "session_token", Value: token})

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); !strings.Contains(got, `"refresh":"skipped"`) {
		t.Errorf("erwartet refresh=skipped, Body: %s", got)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("bei skipped darf kein neues Cookie gesetzt werden")
	}
}

// Ein frisches Token einer Anmeldung mit Zeile in sitzungen wird nicht erneuert und fasst die
// Zeile nicht an. Das Token ohne Zeile im Test darüber käme auch ohne die Regel der halben
// Laufzeit als „skipped" zurück, weil sich ohne Zeile nichts verlängern lässt.
func TestSitzungVerlaengern_FrischesTokenMitSitzungFasstDieZeileNichtAn(t *testing.T) {
	a, mock := newTestAnmeldung(t, 12*time.Hour)
	token, err := a.ErzeugeToken("user-1", "B-1", RolleAdmin, testSitzungID)
	if err != nil {
		t.Fatalf("ErzeugeToken: %v", err)
	}
	erwarteNichtWiderrufen(mock)
	erwarteKontoAktiv(mock, true)

	rec := rufeSitzungVerlaengern(t, a, &http.Cookie{Name: "session_token", Value: token})

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); !strings.Contains(got, `"refresh":"skipped"`) {
		t.Errorf("erwartet refresh=skipped, Body: %s", got)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("ein frisches Token darf kein neues Cookie bekommen")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestSitzungVerlaengern_AltesTokenWirdMitNeuemCookieErneuert(t *testing.T) {
	// Das Token wurde mit kurzer Laufzeit ausgestellt (eine Stunde übrig). Die Tür läuft mit
	// zwölf Stunden: Eine Stunde ist weniger als die Hälfte, also gibt es ein neues Cookie.
	aussteller, _ := newTestAnmeldung(t, 1*time.Hour)
	token, err := aussteller.ErzeugeToken("user-1", "B-1", RolleMitarbeiter, testSitzungID)
	if err != nil {
		t.Fatalf("ErzeugeToken: %v", err)
	}

	a, mock := newTestAnmeldung(t, 12*time.Hour)
	erwarteNichtWiderrufen(mock)
	erwarteKontoStatus(mock, true, RolleMitarbeiter)
	erwarteSitzungVerlaengert(mock, 1)

	rec := rufeSitzungVerlaengern(t, a, &http.Cookie{Name: "session_token", Value: token})

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); !strings.Contains(got, `"refresh":"renewed"`) {
		t.Errorf("erwartet refresh=renewed, Body: %s", got)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session_token" || cookies[0].Value == "" {
		t.Fatalf("erwartet neues session_token-Cookie, bekam %+v", cookies)
	}
	if !cookies[0].HttpOnly {
		t.Errorf("Session-Cookie muss HttpOnly sein")
	}

	// Das neue Token muss gültig sein und die Claims tragen (Rolle unverändert, weil die Datenbank
	// hier dieselbe Rolle meldet).
	erwarteNichtWiderrufen(mock)
	erwarteKontoStatus(mock, true, RolleMitarbeiter)
	erwarteSperrzustand(mock, false)
	claims, err := a.PruefeToken(cookies[0].Value)
	if err != nil {
		t.Fatalf("neues Token ungültig: %v", err)
	}
	if claims.KontoID != "user-1" || claims.Rolle != RolleMitarbeiter {
		t.Errorf("Claims nicht übernommen: %+v", claims)
	}
	if claims.SitzungID != testSitzungID {
		t.Errorf("SitzungID = %q; das erneuerte Token muss die Kennung der Anmeldung behalten, sonst fiele eine Sperre mit der Erneuerung weg", claims.SitzungID)
	}
}

// Eine Herabstufung wirkt sofort. Die Autorisierung liest claims.Rolle; stammte die Rolle aus
// dem bis zu zwölf Stunden gültigen Token, schriebe die Verlängerung sie in jedes folgende Token
// fort, und ein Herabstufen wirkte nie, während ein Deaktivieren sofort wirkt.
func TestSitzungVerlaengern_HerabstufungWirdNichtFortgeschrieben(t *testing.T) {
	// Das Token wurde für ADMIN ausgestellt, eine Stunde übrig: Es wird erneuert.
	aussteller, _ := newTestAnmeldung(t, 1*time.Hour)
	token, err := aussteller.ErzeugeToken("user-1", "B-1", RolleAdmin, testSitzungID)
	if err != nil {
		t.Fatalf("ErzeugeToken: %v", err)
	}

	// In der Datenbank ist dasselbe Konto inzwischen nur noch HELFER.
	a, mock := newTestAnmeldung(t, 12*time.Hour)
	erwarteNichtWiderrufen(mock)
	erwarteKontoStatus(mock, true, RolleHelfer)
	erwarteSitzungVerlaengert(mock, 1)

	rec := rufeSitzungVerlaengern(t, a, &http.Cookie{Name: "session_token", Value: token})
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("erwartet neues session_token-Cookie, bekam %+v", cookies)
	}

	erwarteNichtWiderrufen(mock)
	erwarteKontoStatus(mock, true, RolleHelfer)
	erwarteSperrzustand(mock, false)
	claims, err := a.PruefeToken(cookies[0].Value)
	if err != nil {
		t.Fatalf("neues Token ungültig: %v", err)
	}
	if claims.Rolle != RolleHelfer {
		t.Errorf("Refresh hat Rolle %q fortgeschrieben; want %q (aktuelle DB-Rolle)", claims.Rolle, RolleHelfer)
	}
}

// Die Autorisierung wertet claims.Rolle aus. PruefeToken muss dort deshalb die Rolle aus der
// Datenbank liefern, nicht die im Token signierte, sonst griffe jede Änderung der Rechte erst bei
// der nächsten Anmeldung.
func TestPruefeToken_RolleKommtAusDerDatenbank(t *testing.T) {
	a, mock := newTestAnmeldung(t, 12*time.Hour)
	token, err := a.ErzeugeToken("user-1", "B-1", RolleAdmin, "")
	if err != nil {
		t.Fatalf("ErzeugeToken: %v", err)
	}

	erwarteNichtWiderrufen(mock)
	erwarteKontoStatus(mock, true, RolleHelfer)

	claims, err := a.PruefeToken(token)
	if err != nil {
		t.Fatalf("PruefeToken: %v", err)
	}
	if claims.Rolle != RolleHelfer {
		t.Errorf("claims.Rolle = %q; want %q (Token sagt ADMIN, die DB sagt HELFER)", claims.Rolle, RolleHelfer)
	}
}

func TestSitzungVerlaengern_WiderrufenesTokenIst401(t *testing.T) {
	// Die Abmeldung setzt das Token auf die Sperrliste; danach darf die Verlängerung es nicht
	// wiederbeleben.
	a, mock := newTestAnmeldung(t, 12*time.Hour)
	token, err := a.ErzeugeToken("user-1", "B-1", RolleAdmin, "")
	if err != nil {
		t.Fatalf("ErzeugeToken: %v", err)
	}

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	rec := rufeSitzungVerlaengern(t, a, &http.Cookie{Name: "session_token", Value: token})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("erwartet 401 für widerrufenes Token, bekam %d", rec.Code)
	}
}
