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

func (a meldeAttrappe) Melde(context.Context, string, string, string, repository.SchadensArt, betrag.Cent) (string, error) {
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
		{"Geräteausleihe", repository.ErrAusleiheOhneBuch, http.StatusConflict,
			"Verlust und Schaden lassen sich hier nur für ein ausgeliehenes Buch buchen, nicht für ein Gerät"},
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

// zaehlendeMeldung zählt die Aufrufe von Melde und nimmt jeden Betrag an.
type zaehlendeMeldung struct{ aufrufe *int }

func (a zaehlendeMeldung) Melde(context.Context, string, string, string, repository.SchadensArt, betrag.Cent) (string, error) {
	*a.aufrufe++
	return "s-1", nil
}

// Ein negativer Betrag scheiterte erst an check_positive_amount, als 500 ohne Grund. 0 € bleibt
// erlaubt: Beim Kollegium fragt der Dialog keinen Betrag.
func TestSchadenMelden_Betrag(t *testing.T) {
	faelle := []struct {
		name     string
		betrag   string
		status   int
		gemeldet int
	}{
		{"negativ", "-25", http.StatusBadRequest, 0},
		{"null", "0", http.StatusOK, 1},
		{"positiv", "12.5", http.StatusOK, 1},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			aufrufe := 0
			srv := &Tueren{Basis: &tuer.Basis{}}
			req := tuertest.JSONPost("/api/damage/report",
				`{"loan_id":"11111111-1111-1111-1111-111111111111","art":"nicht_zurueckgegeben","betrag":`+f.betrag+`}`)
			req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
				&auth.Claims{KontoID: "k-1", Rolle: auth.RolleMitarbeiter}))
			rec := httptest.NewRecorder()
			srv.SchadenMelden(zaehlendeMeldung{aufrufe: &aufrufe})(rec, req)

			if rec.Code != f.status {
				t.Fatalf("SchadenMelden: Status %d, erwartet %d: %s", rec.Code, f.status, rec.Body.String())
			}
			if aufrufe != f.gemeldet {
				t.Errorf("Melde %d-mal gerufen, erwartet %d", aufrufe, f.gemeldet)
			}
			if f.status == http.StatusBadRequest {
				if got := tuertest.Fehlermeldung(t, rec); got != "Der Betrag darf nicht negativ sein" {
					t.Errorf("Meldung %q", got)
				}
			}
		})
	}
}
