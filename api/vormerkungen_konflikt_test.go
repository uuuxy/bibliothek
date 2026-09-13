package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/repository"
)

// vormerkungRepoStub gibt genau den Fehler zurück, den der Test braucht — die Frage hier
// ist die Einordnung im Handler, nicht die Abfrage in der Datenbank (die steht in
// repository/vormerkung_pg_test.go am echten Constraint).
type vormerkungRepoStub struct {
	repository.VormerkungRepository
	err error
}

func (v vormerkungRepoStub) Create(context.Context, string, string, string) (string, error) {
	return "", v.err
}

// Die zweite Vormerkung desselben Schülers auf denselben Titel ist ein Konflikt (409),
// kein Serverfehler. Bis zum 12.09.2026 lief der Constraint-Fehler in den Sammel-Zweig
// apierrors.Internal: Die Theke bekam 500 und „Fehler beim Erstellen der Vormerkung" —
// dieselbe Meldung wie bei einem kaputten Server, und keine Auskunft darüber, ob die
// erste Vormerkung noch steht.
func TestCreateVormerkung_ZweiteIstKonflikt(t *testing.T) {
	faelle := []struct {
		name   string
		err    error
		status int
	}{
		{"bereits vorgemerkt", repository.ErrVormerkungBereitsVorhanden, http.StatusConflict},
		{"Titel selbst ausgeliehen", repository.ErrTitelBereitsAusgeliehen, http.StatusConflict},
		{"echte Störung", errors.New("connection reset by peer"), http.StatusInternalServerError},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			s := &Server{}
			h := s.CreateVormerkungHandler(vormerkungRepoStub{err: f.err})
			req := httptest.NewRequest(http.MethodPost, "/api/vormerkungen",
				strings.NewReader(`{"titel_id":"11111111-1111-1111-1111-111111111111","schueler_id":"22222222-2222-2222-2222-222222222222"}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h(rec, req)

			if rec.Code != f.status {
				t.Fatalf("Status %d, erwartet %d: %s", rec.Code, f.status, rec.Body.String())
			}
			if f.status != http.StatusConflict {
				return
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("Body ist kein JSON: %q", rec.Body.String())
			}
			if body["error"] != f.err.Error() {
				t.Errorf("Meldung %q, erwartet den fachlichen Satz %q", body["error"], f.err.Error())
			}
		})
	}
}
