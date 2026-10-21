package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v5"
)

// Die Antwort des Abmeldens kommt aus der Wirkung, nicht aus der Eingabe. Meldete die Tür einen
// Fehlschlag von Blacklist.Add nur als Logzeile und antwortete in jedem Fall {"status":"ok"},
// bliebe das Token bei einem Aussetzer der Datenbank bis zu seinem Ablauf gültig, bis zu zwölf
// Stunden, während der Bediener eine gelungene Abmeldung gemeldet bekommt. Nach genau dieser
// Form fragt Frage 5 in docs/invarianten.md: Erfolg anhand der Wirkung oder der Eingabe.
//
// Das Löschcookie bleibt in jedem Fall: Es ist die Hälfte, die wirklich stattgefunden hat.
func abmeldeServer(t *testing.T) (*Server, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock init: %v", err)
	}
	t.Cleanup(mock.Close)
	a, err := auth.NewAnmeldung(testJWTGeheimnis, mock, time.Hour)
	if err != nil {
		t.Fatalf("authenticator init: %v", err)
	}
	return &Server{Basis: tuer.Basis{DB: &db.Verbindung{Pool: mock}, Auth: a}, nurHTTPS: true}, mock
}

func abmeldenMitToken(t *testing.T, s *Server) *httptest.ResponseRecorder {
	t.Helper()
	token, err := s.Auth.ErzeugeToken("u1", auth.RolleMitarbeiter, "")
	if err != nil {
		t.Fatalf("token generation: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rec := httptest.NewRecorder()
	s.abmelden()(rec, req)
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
func TestAbmeldung_WiderrufScheitert_MeldetKeinenErfolg(t *testing.T) {
	s, mock := abmeldeServer(t)
	erwarteSperrlisteFrei(mock, auth.RolleMitarbeiter)
	mock.ExpectExec("INSERT INTO revoked_tokens").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(errors.New("connection reset by peer"))

	rec := abmeldenMitToken(t, s)

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
// Token, bei dem es nichts zu widerrufen gibt.
func TestAbmeldung_PruefungGestoert_MeldetKeinenErfolg(t *testing.T) {
	s, mock := abmeldeServer(t)
	mock.ExpectQuery("revoked_tokens").
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(errors.New("context deadline exceeded"))

	rec := abmeldenMitToken(t, s)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Status %d, erwartet 503 — der Widerruf wurde nie versucht", rec.Code)
	}
	if !loeschcookieGesetzt(rec) {
		t.Error("kein Löschcookie: Die Hälfte, die stattgefunden hat, muss trotzdem wirken")
	}
}

// Gegenprobe 1: Gelingt der Widerruf, bleibt alles beim Alten — 200 und {"status":"ok"}.
func TestAbmeldung_WiderrufGelingt_BleibtOk(t *testing.T) {
	s, mock := abmeldeServer(t)
	erwarteSperrlisteFrei(mock, auth.RolleMitarbeiter)
	mock.ExpectExec("INSERT INTO revoked_tokens").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	rec := abmeldenMitToken(t, s)

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
// (Die Attribute dieses Cookies prüft abmeldung_cookie_test.go.)
func TestAbmeldung_UngueltigesToken_BleibtOk(t *testing.T) {
	s, mock := abmeldeServer(t)
	// Die Sperrliste antwortet (die Datenbank steht), das Token scheitert danach an der
	// Signatur — erst diese Reihenfolge trennt „nichts zu widerrufen" von „Aussetzer".
	mock.ExpectQuery("revoked_tokens").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "kein.gueltiges.token"})
	rec := httptest.NewRecorder()
	s.abmelden()(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status %d, erwartet 200 — ohne widerrufbares Token ist die Abmeldung vollständig", rec.Code)
	}
}
