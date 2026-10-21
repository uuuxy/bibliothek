package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/auth"

	"github.com/pashagolub/pgxmock/v5"
)

// Die Sperre nach Inaktivität an der Tür jeder geschützten Route: Eine gesperrte Anmeldung
// erreicht keine Tür. Der Weg durch Anmeldung, Sperren und Entsperren gegen echtes
// Postgres steht in auth/sperre_pg_test.go.

const sperreSitzungID = "5b0f6a2e-8a0c-4a51-9d1c-3f2f6b7c9e10"

func anfrageMitSitzung(t *testing.T, s *Server, role auth.Rolle) *http.Request {
	t.Helper()
	token, err := s.Auth.ErzeugeToken("u1", role, sperreSitzungID)
	if err != nil {
		t.Fatalf("token generation: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	return req
}

func erwarteSperrzustand(mock pgxmock.PgxPoolIface, gesperrt bool) {
	mock.ExpectQuery("SELECT gesperrt_seit IS NOT NULL FROM sitzungen").
		WithArgs(sperreSitzungID).
		WillReturnRows(pgxmock.NewRows([]string{"gesperrt"}).AddRow(gesperrt))
}

func TestGesperrteSitzung_Antwortet423UndErreichtKeineTuer(t *testing.T) {
	tueren := map[string]func(*Server, *http.Request) (*httptest.ResponseRecorder, bool){
		"verlangeRecht": func(s *Server, req *http.Request) (*httptest.ResponseRecorder, bool) {
			return laufeDurch(s, "buch.loeschen", req)
		},
		"VerlangeAnmeldung": func(s *Server, req *http.Request) (*httptest.ResponseRecorder, bool) {
			reached := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })
			rr := httptest.NewRecorder()
			s.VerlangeAnmeldung()(next).ServeHTTP(rr, req)
			return rr, reached
		},
	}
	for name, tuer := range tueren {
		t.Run(name, func(t *testing.T) {
			s, mock := baueRechteServer(t)
			defer mock.Close()

			// Auch ein Admin kommt nicht durch: Die Sperre steht vor jedem Recht.
			erwarteSperrlisteFrei(mock, auth.RolleAdmin)
			erwarteSperrzustand(mock, true)

			rr, reached := tuer(s, anfrageMitSitzung(t, s, auth.RolleAdmin))

			if rr.Code != http.StatusLocked {
				t.Errorf("Status %d, erwartet 423 — 401 meldete den Arbeitsplatz ab, 200 lieferte Daten hinter der Sperre", rr.Code)
			}
			if reached {
				t.Error("der geschützte Handler wurde trotz Sperre erreicht")
			}
			var body map[string]string
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("Body ist kein JSON: %q (%v)", rr.Body.String(), err)
			}
			if want := "Gesperrt wegen Inaktivität. Bitte das Passwort eingeben."; body["error"] != want {
				t.Errorf("Meldung %q, erwartet %q", body["error"], want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unerfüllte Erwartungen: %v", err)
			}
		})
	}
}

func TestNichtGesperrteSitzung_WirdDurchgelassen(t *testing.T) {
	s, mock := baueRechteServer(t)
	defer mock.Close()

	erwarteSperrlisteFrei(mock, auth.RolleAdmin)
	erwarteSperrzustand(mock, false)

	rr, reached := laufeDurch(s, "buch.loeschen", anfrageMitSitzung(t, s, auth.RolleAdmin))
	if rr.Code != http.StatusOK || !reached {
		t.Errorf("nicht gesperrt: code %d, reached %v — erwartet 200 und erreicht", rr.Code, reached)
	}
}

// Lässt sich der Sperrzustand nicht lesen, wird abgelehnt — als 503, nicht als abgelaufene
// Sitzung (401 meldete den Arbeitsplatz ab).
func TestSperrzustandNichtLesbar_Lehnt503Ab(t *testing.T) {
	s, mock := baueRechteServer(t)
	defer mock.Close()

	erwarteSperrlisteFrei(mock, auth.RolleAdmin)
	mock.ExpectQuery("SELECT gesperrt_seit IS NOT NULL FROM sitzungen").
		WithArgs(sperreSitzungID).
		WillReturnError(errors.New("connection reset by peer"))

	rr, reached := laufeDurch(s, "buch.loeschen", anfrageMitSitzung(t, s, auth.RolleAdmin))
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Status %d, erwartet 503", rr.Code)
	}
	if reached {
		t.Error("der geschützte Handler wurde erreicht, obwohl der Sperrzustand unbekannt ist")
	}
}

// Abmelden geht aus der Sperre heraus und nimmt die Zeile der Anmeldung mit.
func TestAbmeldung_AusDerSperre_LoeschtDieZeile(t *testing.T) {
	s, mock := abmeldeServer(t)

	// Keine Abfrage des Sperrzustands: Der Abmelde-Weg prüft ohne sie.
	erwarteSperrlisteFrei(mock, auth.RolleMitarbeiter)
	mock.ExpectExec("INSERT INTO revoked_tokens").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("DELETE FROM sitzungen WHERE id").
		WithArgs(sperreSitzungID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	token, err := s.Auth.ErzeugeToken("u1", auth.RolleMitarbeiter, sperreSitzungID)
	if err != nil {
		t.Fatalf("token generation: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rec := httptest.NewRecorder()
	s.abmelden()(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if !loeschcookieGesetzt(rec) {
		t.Error("kein Löschcookie")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerfüllte Erwartungen (der Prüfwert des Passworts muss mit der Abmeldung gehen): %v", err)
	}
}
