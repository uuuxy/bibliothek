package theke

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"
)

// vormerkungsAttrappe gibt genau den Fehler zurück, den der Test braucht — die Frage hier
// ist die Einordnung im Handler, nicht die Abfrage in der Datenbank (die steht in
// repository/vormerkung_pg_test.go am echten Constraint).
type vormerkungsAttrappe struct {
	repository.VormerkungsAbfragen
	err error
}

func (v vormerkungsAttrappe) LegeAn(context.Context, string, string, string) (string, error) {
	return "", v.err
}

// Die zweite Vormerkung desselben Schülers auf denselben Titel ist ein Konflikt (409), kein
// Serverfehler. Liefe der Fehler der Einschränkung in den Sammel-Zweig apierrors.Internal,
// bekäme die Theke 500 und „Fehler beim Erstellen der Vormerkung", dieselbe Meldung wie bei
// einem kaputten Server, und keine Auskunft darüber, ob die erste Vormerkung noch steht.
func TestVormerkungAnlegen_ZweiteIstKonflikt(t *testing.T) {
	faelle := []struct {
		name    string
		err     error
		status  int
		meldung string
	}{
		{"bereits vorgemerkt", repository.ErrVormerkungBereitsVorhanden, http.StatusConflict,
			"Dieser Schüler hat den Titel bereits vorgemerkt"},
		{"Titel selbst ausgeliehen", repository.ErrTitelBereitsAusgeliehen, http.StatusConflict,
			"Buch wird aktuell bereits von diesem Schüler ausgeliehen"},
		{"kein Schüler", repository.ErrVormerkungNurFuerSchueler, http.StatusConflict,
			"Vormerken lässt sich nur für Schüler"},
		{"echte Störung", errors.New("connection reset by peer"), http.StatusInternalServerError,
			"Fehler beim Erstellen der Vormerkung"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			s := &Tueren{Basis: &tuer.Basis{}}
			h := s.VormerkungAnlegen(vormerkungsAttrappe{err: f.err})
			req := httptest.NewRequest(http.MethodPost, "/api/vormerkungen",
				strings.NewReader(`{"titel_id":"11111111-1111-1111-1111-111111111111","schueler_id":"22222222-2222-2222-2222-222222222222"}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h(rec, req)

			if rec.Code != f.status {
				t.Fatalf("Status %d, erwartet %d: %s", rec.Code, f.status, rec.Body.String())
			}
			if got := tuertest.Fehlermeldung(t, rec); got != f.meldung {
				t.Errorf("VormerkungAnlegen: Meldung %q, erwartet %q", got, f.meldung)
			}
		})
	}
}

// Ohne Titel und ohne Schüler gibt es keine Liste aller Vormerkungen: ein Bedienfehler (400) mit
// dem Satz, der sagt, was fehlt. Die echten Abfragen antworten so, bevor sie die Datenbank
// fragen.
func TestVormerkungen_OhneFilterIstBedienfehler(t *testing.T) {
	s := &Tueren{Basis: &tuer.Basis{}}
	h := s.Vormerkungen(repository.NewVormerkungsAbfragen(nil))
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/api/vormerkungen", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}
	const erwartet = "Vormerkungen brauchen einen Titel- oder Schüler-Filter"
	if got := tuertest.Fehlermeldung(t, rec); got != erwartet {
		t.Errorf("Vormerkungen: Meldung %q, erwartet %q", got, erwartet)
	}
}
