package schaden

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/internal/tuertest"
	"bibliothek/pkg/betrag"
	"bibliothek/repository"
)

// meldeAttrappe gibt beim Melden den Fehler zurück, den der Test braucht. Wann die Abfragen
// ihn liefern, prüft repository/schaden_melden_ausleihe_pg_test.go an der Datenbank.
type meldeAttrappe struct {
	err error
}

func (a meldeAttrappe) Melde(context.Context, string, string, string, string, repository.SchadensArt, betrag.Cent) (string, error) {
	return "", a.err
}

// Hat sich die Lage geändert, während die Akte offen stand, antwortet die Tür mit 409 und dem
// Satz, der sagt, was zu tun ist; jede andere Störung ist ein 500 ohne Einzelheiten.
func TestSchadenMelden_Meldungen(t *testing.T) {
	faelle := []struct {
		name    string
		err     error
		status  int
		meldung string
	}{
		{"neu verliehen", repository.ErrExemplarNeuVerliehen, http.StatusConflict,
			"Exemplar wurde zwischenzeitlich neu ausgeliehen. Bitte den Vorgang neu laden."},
		{"inzwischen zurück", repository.ErrAusleiheInzwischenZurueck, http.StatusConflict,
			"Ein gewähltes Buch ist inzwischen zurückgegeben. Bitte den Dialog neu öffnen."},
		{"Störung", errors.New("verbindung abgerissen"), http.StatusInternalServerError,
			"Fehler beim Melden des Schadens"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			srv := &Tueren{Basis: &tuer.Basis{}}
			req := tuertest.JSONPost("/api/damage/report",
				`{"loan_id":"11111111-1111-1111-1111-111111111111","copy_id":"22222222-2222-2222-2222-222222222222","art":"nicht_zurueckgegeben","betrag":12}`)
			req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
				&auth.Claims{KontoID: "k-1", Rolle: auth.RolleMitarbeiter}))
			rec := httptest.NewRecorder()
			srv.SchadenMelden(meldeAttrappe{err: f.err})(rec, req)

			if rec.Code != f.status {
				t.Fatalf("SchadenMelden: Status %d, erwartet %d: %s", rec.Code, f.status, rec.Body.String())
			}
			if got := tuertest.Fehlermeldung(t, rec); got != f.meldung {
				t.Errorf("SchadenMelden: Meldung %q, erwartet %q", got, f.meldung)
			}
		})
	}
}
