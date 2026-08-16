package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/repository"
)

// Die HTTP-Schicht der Gebühren-Erledigung: Statuscode-Mapping der Sentinels
// (404/409 statt sanitiertem 500), Pflicht-Grund beim Storno, 401 ohne Session.
// Das SQL-Verhalten selbst sichert repository/gebuehr_erledigung_pg_test.go.

type stubGebuehrAuditRepo struct {
	repository.AuditRepository
	stornoErr  error
	bezahltErr error
	grund      string
}

func (s *stubGebuehrAuditRepo) StornierungGebuehr(_ context.Context, _, _, grund string) error {
	s.grund = grund
	return s.stornoErr
}
func (s *stubGebuehrAuditRepo) BezahltGebuehr(_ context.Context, _, _ string) error {
	return s.bezahltErr
}

func gebuehrRequest(t *testing.T, methode, ziel, body string, mitClaims bool) *http.Request {
	t.Helper()
	var leser *strings.Reader
	if body == "" {
		leser = strings.NewReader("")
	} else {
		leser = strings.NewReader(body)
	}
	req := httptest.NewRequest(methode, ziel, leser)
	req.SetPathValue("id", "sf-1")
	req.Header.Set("Content-Type", "application/json")
	if mitClaims {
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: "u-1", Rolle: auth.RoleMitarbeiter}))
	}
	return req
}

func TestStornoGebuehrHandler(t *testing.T) {
	srv := &Server{}

	faelle := []struct {
		name       string
		body       string
		mitClaims  bool
		repoErr    error
		wantStatus int
		wantGrund  string
	}{
		{"ohne Session -> 401", `{"grund":"x"}`, false, nil, http.StatusUnauthorized, ""},
		{"leerer Grund -> 400", `{"grund":"   "}`, true, nil, http.StatusBadRequest, ""},
		{"bereits erledigt -> 409", `{"grund":"Kulanz"}`, true, repository.ErrSchadensfallErledigt, http.StatusConflict, "Kulanz"},
		{"unbekannt -> 404", `{"grund":"Kulanz"}`, true, repository.ErrSchadensfallNichtGefunden, http.StatusNotFound, "Kulanz"},
		{"Erfolg, Grund getrimmt", `{"grund":"  Buch wiedergefunden  "}`, true, nil, http.StatusOK, "Buch wiedergefunden"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			repo := &stubGebuehrAuditRepo{stornoErr: f.repoErr}
			rec := httptest.NewRecorder()
			srv.StornoGebuehrHandler(repo)(rec, gebuehrRequest(t, http.MethodPost, "/api/schadensfaelle/sf-1/storno", f.body, f.mitClaims))
			if rec.Code != f.wantStatus {
				t.Fatalf("Status = %d, erwartet %d (Body: %s)", rec.Code, f.wantStatus, rec.Body.String())
			}
			if repo.grund != f.wantGrund {
				t.Errorf("übergebener Grund = %q, erwartet %q", repo.grund, f.wantGrund)
			}
		})
	}
}

func TestBezahltGebuehrHandler(t *testing.T) {
	srv := &Server{}

	t.Run("bereits erledigt -> 409 mit lesbarer Meldung", func(t *testing.T) {
		repo := &stubGebuehrAuditRepo{bezahltErr: repository.ErrSchadensfallErledigt}
		rec := httptest.NewRecorder()
		srv.BezahltGebuehrHandler(repo)(rec, gebuehrRequest(t, http.MethodPost, "/api/schadensfaelle/sf-1/bezahlt", "", true))
		if rec.Code != http.StatusConflict {
			t.Fatalf("Status = %d, erwartet 409", rec.Code)
		}
		var antwort map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort parsen: %v", err)
		}
		// Die Meldung muss den Konflikt benennen — nicht der sanitierte 500-Text.
		if !strings.Contains(antwort["error"], "bereits bezahlt oder storniert") {
			t.Errorf("Fehlermeldung = %q, erwartet den Konflikt-Text", antwort["error"])
		}
	})

	t.Run("Erfolg -> 200", func(t *testing.T) {
		repo := &stubGebuehrAuditRepo{}
		rec := httptest.NewRecorder()
		srv.BezahltGebuehrHandler(repo)(rec, gebuehrRequest(t, http.MethodPost, "/api/schadensfaelle/sf-1/bezahlt", "", true))
		if rec.Code != http.StatusOK {
			t.Fatalf("Status = %d, erwartet 200 (Body: %s)", rec.Code, rec.Body.String())
		}
	})
}
