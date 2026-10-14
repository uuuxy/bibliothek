package schaden

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/repository"
)

// Die Türen zum Erledigen einer Forderung: die Zuordnung der Fehler zu einem Status (404/409
// statt eines bereinigten 500), der Pflicht-Grund beim Storno, 401 ohne Sitzung.
// Das SQL-Verhalten selbst sichert repository/forderung_erledigen_pg_test.go.

type gebuehrAttrappe struct {
	repository.ProtokollAbfragen
	stornoErr  error
	bezahltErr error
	grund      string
}

func (a *gebuehrAttrappe) StorniereForderung(_ context.Context, _, _, grund string) error {
	a.grund = grund
	return a.stornoErr
}
func (a *gebuehrAttrappe) VerbucheZahlung(_ context.Context, _, _ string) error {
	return a.bezahltErr
}

func gebuehrAnfrage(t *testing.T, methode, ziel, body string, mitClaims bool) *http.Request {
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

func TestForderungStornieren(t *testing.T) {
	srv := &Tueren{Basis: &tuer.Basis{}}

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
			repo := &gebuehrAttrappe{stornoErr: f.repoErr}
			rec := httptest.NewRecorder()
			srv.ForderungStornieren(repo)(rec, gebuehrAnfrage(t, http.MethodPost, "/api/schadensfaelle/sf-1/storno", f.body, f.mitClaims))
			if rec.Code != f.wantStatus {
				t.Fatalf("Status = %d, erwartet %d (Body: %s)", rec.Code, f.wantStatus, rec.Body.String())
			}
			if repo.grund != f.wantGrund {
				t.Errorf("übergebener Grund = %q, erwartet %q", repo.grund, f.wantGrund)
			}
		})
	}
}

func TestZahlungVerbuchen(t *testing.T) {
	srv := &Tueren{Basis: &tuer.Basis{}}

	t.Run("bereits erledigt -> 409 mit lesbarer Meldung", func(t *testing.T) {
		repo := &gebuehrAttrappe{bezahltErr: repository.ErrSchadensfallErledigt}
		rec := httptest.NewRecorder()
		srv.ZahlungVerbuchen(repo)(rec, gebuehrAnfrage(t, http.MethodPost, "/api/schadensfaelle/sf-1/bezahlt", "", true))
		if rec.Code != http.StatusConflict {
			t.Fatalf("Status = %d, erwartet 409", rec.Code)
		}
		var antwort map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort parsen: %v", err)
		}
		// Die Meldung muss den Konflikt benennen, nicht der bereinigte Text einer 500.
		if !strings.Contains(antwort["error"], "bereits bezahlt oder storniert") {
			t.Errorf("Fehlermeldung = %q, erwartet den Konflikt-Text", antwort["error"])
		}
	})

	t.Run("Erfolg -> 200", func(t *testing.T) {
		repo := &gebuehrAttrappe{}
		rec := httptest.NewRecorder()
		srv.ZahlungVerbuchen(repo)(rec, gebuehrAnfrage(t, http.MethodPost, "/api/schadensfaelle/sf-1/bezahlt", "", true))
		if rec.Code != http.StatusOK {
			t.Fatalf("Status = %d, erwartet 200 (Body: %s)", rec.Code, rec.Body.String())
		}
	})
}
