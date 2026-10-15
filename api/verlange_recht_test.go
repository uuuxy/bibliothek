package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

const testJWTGeheimnis = "0123456789abcdef0123456789abcdef" // 32 Byte

func baueRechteServer(t *testing.T) (*Server, pgxmock.PgxPoolIface) {
	t.Helper()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock init: %v", err)
	}
	a, err := auth.NewAnmeldung(testJWTGeheimnis, mock, time.Hour)
	if err != nil {
		t.Fatalf("authenticator init: %v", err)
	}
	return &Server{Basis: tuer.Basis{DB: &db.Verbindung{Pool: mock}, Auth: a}}, mock
}

// erwarteSperrlisteFrei erwartet die beiden Prüfungen in PruefeToken bei gültigem Token:
// die Blacklist ("nicht widerrufen") und direkt danach Kontostatus samt aktueller Rolle.
// Die Rolle muss mitgeliefert werden, weil PruefeToken sie über die im Token signierte
// schreibt — die Autorisierung folgt der Datenbank, nicht dem Token.
func erwarteSperrlisteFrei(mock pgxmock.PgxPoolIface, rolle auth.Rolle) {
	mock.ExpectQuery("revoked_tokens").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("SELECT aktiv, rolle FROM benutzer").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"aktiv", "rolle"}).AddRow(true, string(rolle)))
}

func anfrageMitToken(t *testing.T, s *Server, role auth.Rolle) *http.Request {
	t.Helper()
	token, err := s.Auth.ErzeugeToken("u1", "BC1", role, "")
	if err != nil {
		t.Fatalf("token generation: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	return req
}

// laufeDurch führt die Middleware aus und meldet zurück, ob die geschützte Tür erreicht wurde.
func laufeDurch(s *Server, permission string, req *http.Request) (*httptest.ResponseRecorder, bool) {
	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	rr := httptest.NewRecorder()
	s.verlangeRecht(permission)(next).ServeHTTP(rr, req)
	return rr, reached
}

func TestVerlangeRecht_OhneCookieIst401(t *testing.T) {
	s, mock := baueRechteServer(t)
	defer mock.Close()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil) // ohne Cookie
	rr, reached := laufeDurch(s, "buch.loeschen", req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("ohne Cookie erwartet 401, bekam %d", rr.Code)
	}
	if reached {
		t.Error("geschützter Handler darf ohne Auth nicht erreicht werden")
	}
}

func TestVerlangeRecht_AdminUmgehtDiePruefung(t *testing.T) {
	s, mock := baueRechteServer(t)
	defer mock.Close()

	erwarteSperrlisteFrei(mock, auth.RolleAdmin)
	// Keine role_permissions-Abfrage erwartet — Admin hat implizit alle Rechte.

	req := anfrageMitToken(t, s, auth.RolleAdmin)
	rr, reached := laufeDurch(s, "buch.loeschen", req)

	if rr.Code != http.StatusOK || !reached {
		t.Errorf("Admin soll durchgelassen werden: code %d, reached %v", rr.Code, reached)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerfüllte Erwartungen: %v", err)
	}
}

func TestVerlangeRecht_GewaehrtLaesstDurch(t *testing.T) {
	s, mock := baueRechteServer(t)
	defer mock.Close()

	erwarteSperrlisteFrei(mock, auth.RolleKollegium)
	mock.ExpectQuery("role_permissions").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"allowed"}).AddRow(true))

	req := anfrageMitToken(t, s, auth.RolleKollegium)
	rr, reached := laufeDurch(s, "buch.ausleihen", req)

	if rr.Code != http.StatusOK || !reached {
		t.Errorf("gewährte Berechtigung soll durchlassen: code %d, reached %v", rr.Code, reached)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerfüllte Erwartungen: %v", err)
	}
}

func TestVerlangeRecht_DeaktiviertesKontoIst401(t *testing.T) {
	s, mock := baueRechteServer(t)
	defer mock.Close()

	// Blacklist ok, aber das Konto wurde nach Token-Ausstellung deaktiviert → PruefeToken
	// muss ablehnen (Echtzeit-Widerruf), bevor überhaupt Rechte geprüft werden.
	mock.ExpectQuery("revoked_tokens").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("SELECT aktiv, rolle FROM benutzer").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"aktiv", "rolle"}).AddRow(false, string(auth.RolleKollegium)))

	req := anfrageMitToken(t, s, auth.RolleKollegium)
	rr, reached := laufeDurch(s, "buch.ausleihen", req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("deaktiviertes Konto erwartet 401, bekam %d", rr.Code)
	}
	if reached {
		t.Error("deaktiviertes Konto darf den geschützten Handler nicht erreichen")
	}
}

func TestVerlangeRecht_AusdruecklichVerweigertIst403(t *testing.T) {
	s, mock := baueRechteServer(t)
	defer mock.Close()

	erwarteSperrlisteFrei(mock, auth.RolleKollegium)
	mock.ExpectQuery("role_permissions").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"allowed"}).AddRow(false))

	req := anfrageMitToken(t, s, auth.RolleKollegium)
	rr, reached := laufeDurch(s, "buch.loeschen", req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("explizit verweigert erwartet 403, bekam %d", rr.Code)
	}
	if reached {
		t.Error("verweigerter Zugriff darf Handler nicht erreichen")
	}
}

func TestVerlangeRecht_OhneZeileIst403(t *testing.T) {
	s, mock := baueRechteServer(t)
	defer mock.Close()

	erwarteSperrlisteFrei(mock, auth.RolleKollegium)
	mock.ExpectQuery("role_permissions").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	req := anfrageMitToken(t, s, auth.RolleKollegium)
	rr, reached := laufeDurch(s, "unbekanntes.recht", req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("fehlender Eintrag (kein Recht) erwartet 403, bekam %d", rr.Code)
	}
	if reached {
		t.Error("fehlendes Recht darf Handler nicht erreichen")
	}
}

func TestVerlangeRecht_VoruebergehenderDatenbankfehlerIst500(t *testing.T) {
	s, mock := baueRechteServer(t)
	defer mock.Close()

	erwarteSperrlisteFrei(mock, auth.RolleKollegium)
	mock.ExpectQuery("role_permissions").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(errors.New("connection reset"))

	req := anfrageMitToken(t, s, auth.RolleKollegium)
	rr, reached := laufeDurch(s, "buch.ausleihen", req)

	// Transiente DB-Fehler sind 500, nicht 403 — und dürfen nicht als Negativ-Entscheidung
	// gecacht werden.
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("transienter DB-Fehler erwartet 500, bekam %d", rr.Code)
	}
	if reached {
		t.Error("bei DB-Fehler darf Handler nicht erreicht werden")
	}
}

func TestVerlangeRecht_AblehnungWirdZwischengespeichert(t *testing.T) {
	s, mock := baueRechteServer(t)
	defer mock.Close()

	// Erste Anfrage: Blacklist + role_permissions (false) → 403, Entscheidung wird gecacht.
	erwarteSperrlisteFrei(mock, auth.RolleKollegium)
	mock.ExpectQuery("role_permissions").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"allowed"}).AddRow(false))

	// Zweite Anfrage: nur Blacklist erwartet; role_permissions kommt aus dem Cache.
	erwarteSperrlisteFrei(mock, auth.RolleKollegium)

	for i := 0; i < 2; i++ {
		req := anfrageMitToken(t, s, auth.RolleKollegium)
		rr, reached := laufeDurch(s, "buch.loeschen", req)
		if rr.Code != http.StatusForbidden || reached {
			t.Errorf("Request %d: erwartet 403 ohne Handler-Zugriff, bekam %d reached=%v", i+1, rr.Code, reached)
		}
	}

	// Schlägt fehl, falls die zweite Anfrage doch eine Abfrage von role_permissions ausgelöst
	// hätte.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Cache hat zweite DB-Abfrage nicht verhindert: %v", err)
	}
}

func TestVerlangeRecht_SperrlisteOhneDatenbankIst503(t *testing.T) {
	s, mock := baueRechteServer(t)
	defer mock.Close()

	// Die Abfrage der Sperrliste scheitert: istWiderrufen liefert den Fehler, PruefeToken meldet
	// ErrPruefungGestoert, und die Anfrage endet mit 503 statt 401.
	mock.ExpectQuery("revoked_tokens").
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(errors.New("connection reset by peer"))

	req := anfrageMitToken(t, s, auth.RolleKollegium)
	rr, reached := laufeDurch(s, "buch.ausleihen", req)

	// 503, nicht 401: Eine 401 meldet den Arbeitsplatz in der Oberfläche ab (apiFetch →
	// sitzungAbgelaufen). Ein Datenbank-Aussetzer ist keine abgelaufene Sitzung.
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("fail-closed Blacklist erwartet 503, bekam %d", rr.Code)
	}
	pruefeSitzungsfehlerRumpf(t, rr, "connection reset by peer")
	if reached {
		t.Error("bei nicht verifizierbarem Token darf Handler nicht erreicht werden")
	}
}

func TestVerlangeRecht_KontostatusOhneDatenbankIst503(t *testing.T) {
	s, mock := baueRechteServer(t)
	defer mock.Close()

	mock.ExpectQuery("revoked_tokens").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("SELECT aktiv, rolle FROM benutzer").
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(errors.New("context deadline exceeded"))

	req := anfrageMitToken(t, s, auth.RolleKollegium)
	rr, reached := laufeDurch(s, "buch.ausleihen", req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Kontostatus nicht prüfbar: erwartet 503, bekam %d", rr.Code)
	}
	pruefeSitzungsfehlerRumpf(t, rr, "context deadline exceeded")
	if reached {
		t.Error("bei nicht prüfbarem Kontostatus darf der Handler nicht erreicht werden")
	}
}

// pruefeSitzungsfehlerRumpf: Die Antwort 503 der Middleware trägt genau den Satz des Fehlerwerts
// und keine Spur der eingewickelten Ursache.
//
// Dieselbe Prüfung wie auth.pruefeFehlerwertOhneUrsache, aber an der anderen Tür:
// AngemeldetesKonto und SitzungVerlaengern senden selbst, die Middleware der Rechte über
// sendeSitzungsfehler. Beide Türen führen denselben Fehler nach draußen; stünde hinter dem Satz
// die Ursache, läse das Personal etwa „: sperrliste: connection reset by peer".
func pruefeSitzungsfehlerRumpf(t *testing.T, rr *httptest.ResponseRecorder, ursache string) {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("503-Body ist kein JSON: %q (%v)", rr.Body.String(), err)
	}
	msg := body["error"]
	if want := "Die Anmeldung ließ sich nicht prüfen. Bitte erneut versuchen."; msg != want {
		t.Errorf("503-Meldung %q, erwartet genau %q", msg, want)
	}
	for _, leck := range []string{ursache, "sperrliste", "kontostatus"} {
		if strings.Contains(strings.ToLower(msg), leck) {
			t.Errorf("Betriebsinnenleben im 503-Body: %q steht in %q", leck, msg)
		}
	}
}
