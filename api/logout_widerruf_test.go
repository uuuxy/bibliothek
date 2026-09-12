package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v4"
)

// Gate: Die Antwort des Abmeldens kommt aus der WIRKUNG, nicht aus der Eingabe.
//
// Anlass (Rasterdurchgang 12.09.2026, Register „Abmelden meldet Erfolg, auch wenn nichts
// widerrufen wurde"): Blacklist.Add protokollierte einen Fehlschlag nur als Logzeile, der
// Handler löschte das Cookie und antwortete in jedem Fall {"status":"ok"}. Bei einem
// Datenbank-Aussetzer blieb das Token damit bis zum natürlichen Ablauf gültig — bis zu
// zwölf Stunden —, während der Bediener eine gelungene Abmeldung gemeldet bekam. Genau
// die Form, die docs/sweeps.md unter Frage 5 (Phantom-Erfolg) beschreibt.
//
// Das Löschcookie bleibt in jedem Fall: Es ist die Hälfte, die wirklich stattgefunden hat.
func logoutServer(t *testing.T) (*Server, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock init: %v", err)
	}
	t.Cleanup(mock.Close)
	a, err := auth.NewAuthenticator(testJWTSecret, mock, time.Hour)
	if err != nil {
		t.Fatalf("authenticator init: %v", err)
	}
	t.Cleanup(a.Blacklist.Stop)
	return &Server{DB: &db.Database{Pool: mock}, Auth: a, CookieSecure: true}, mock
}

func logoutMitToken(t *testing.T, s *Server) *httptest.ResponseRecorder {
	t.Helper()
	token, err := s.Auth.GenerateToken("u1", "BC1", auth.RoleMitarbeiter)
	if err != nil {
		t.Fatalf("token generation: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rec := httptest.NewRecorder()
	s.logoutHandler()(rec, req)
	return rec
}

func loeschcookieGesetzt(rec *httptest.ResponseRecorder) bool {
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_token" && c.Value == "" && c.MaxAge < 0 {
			return true
		}
	}
	return false
}

// Der Widerruf selbst scheitert: Das Token bleibt gültig, also darf die Antwort keinen
// Erfolg melden.
func TestLogout_WiderrufScheitert_MeldetKeinenErfolg(t *testing.T) {
	s, mock := logoutServer(t)
	expectBlacklistPass(mock, auth.RoleMitarbeiter)
	mock.ExpectExec("INSERT INTO revoked_tokens").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(errors.New("connection reset by peer"))

	rec := logoutMitToken(t, s)

	if rec.Code == http.StatusOK {
		t.Errorf("Abmeldung meldet 200, obwohl das Token gültig bleibt: %s", rec.Body.String())
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Status %d, erwartet 503 — ein Datenbank-Aussetzer ist kein Serverfehler des Aufrufers", rec.Code)
	}
	if !loeschcookieGesetzt(rec) {
		t.Error("kein Löschcookie: Die Hälfte, die stattgefunden hat, muss trotzdem wirken")
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Body ist kein JSON: %q (%v)", rec.Body.String(), err)
	}
	if body["error"] != auth.ErrWiderrufGestoert.Error() {
		t.Errorf("Meldung %q, erwartet genau %q", body["error"], auth.ErrWiderrufGestoert.Error())
	}
	if body["status"] == "ok" {
		t.Error("Erfolgsfeld in einer Antwort, die keinen Erfolg meldet")
	}
}

// Der Widerruf kommt gar nicht erst zustande, weil schon die Prüfung an der Datenbank
// scheitert. Auch das ist kein Erfolg — und nicht zu verwechseln mit einem ungültigen
// Token, bei dem es nichts zu widerrufen GIBT.
func TestLogout_PruefungGestoert_MeldetKeinenErfolg(t *testing.T) {
	s, mock := logoutServer(t)
	mock.ExpectQuery("revoked_tokens").
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(errors.New("context deadline exceeded"))

	rec := logoutMitToken(t, s)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Status %d, erwartet 503 — der Widerruf wurde nie versucht", rec.Code)
	}
	if !loeschcookieGesetzt(rec) {
		t.Error("kein Löschcookie: Die Hälfte, die stattgefunden hat, muss trotzdem wirken")
	}
}

// Gegenprobe 1: Gelingt der Widerruf, bleibt alles beim Alten — 200 und {"status":"ok"}.
func TestLogout_WiderrufGelingt_BleibtOk(t *testing.T) {
	s, mock := logoutServer(t)
	expectBlacklistPass(mock, auth.RoleMitarbeiter)
	mock.ExpectExec("INSERT INTO revoked_tokens").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	rec := logoutMitToken(t, s)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Body ist kein JSON: %q (%v)", rec.Body.String(), err)
	}
	if body["status"] != "ok" {
		t.Errorf("Body %q, erwartet {\"status\":\"ok\"}", rec.Body.String())
	}
	if !loeschcookieGesetzt(rec) {
		t.Error("kein Löschcookie in der gelungenen Abmeldung")
	}
}

// Gegenprobe 2: Ein unbrauchbares Token ist kein Aussetzer — es gibt nichts zu
// widerrufen, die Abmeldung ist vollständig, und der Bediener bekommt sein 200.
// (Das Attribut-Gate dieses Cookies steht in logout_cookie_secure_test.go.)
func TestLogout_UngueltigesToken_BleibtOk(t *testing.T) {
	s, mock := logoutServer(t)
	// Die Sperrliste antwortet (die Datenbank steht), das Token scheitert danach an der
	// Signatur — erst diese Reihenfolge trennt „nichts zu widerrufen" von „Aussetzer".
	mock.ExpectQuery("revoked_tokens").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "kein.gueltiges.token"})
	rec := httptest.NewRecorder()
	s.logoutHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status %d, erwartet 200 — ohne widerrufbares Token ist die Abmeldung vollständig", rec.Code)
	}
}
