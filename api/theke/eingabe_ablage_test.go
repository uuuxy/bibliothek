package theke

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// Eine Antwort mit Serverfehler wird nicht abgelegt, sie gibt den Schlüssel frei: Die Wiederholung
// soll es neu versuchen. Alles darunter gilt als Antwort auf diesen Schlüssel. Die Grenze ist 500:
// Das ist der häufigste Serverfehler.
func TestLegeAntwortAb_GrenzeServerfehler(t *testing.T) {
	const schluessel = "5e1d0c8a-5b0e-4c55-9d0f-3a8b1c2d4e02"
	faelle := []struct {
		status   int
		erwartet func(mock pgxmock.PgxPoolIface)
	}{
		{http.StatusInternalServerError, func(mock pgxmock.PgxPoolIface) {
			mock.ExpectExec("DELETE FROM idempotency_keys").WithArgs(schluessel).
				WillReturnResult(pgxmock.NewResult("DELETE", 1))
		}},
		{499, func(mock pgxmock.PgxPoolIface) {
			mock.ExpectExec("UPDATE idempotency_keys SET response_data").
				WithArgs(schluessel, pgxmock.AnyArg(), 499).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		}},
	}
	for _, f := range faelle {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		f.erwartet(mock)
		s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}
		s.legeAntwortAb(context.Background(), schluessel, map[string]string{"error": "x"}, f.status)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Status %d: %v", f.status, err)
		}
		mock.Close()
	}
}

// Eine abgelegte Fehlerantwort kommt als Fehler zurück, auch bei genau 400; als gewöhnliche Antwort
// gelesen, stünde an der Theke eine leere Buchung statt der Meldung.
func TestAntworteAusAblage_FehlerantwortBleibtFehler(t *testing.T) {
	s := &Tueren{Basis: &tuer.Basis{}}
	w := httptest.NewRecorder()
	if !s.antworteAusAblage(w, &repository.IdempotenzAntwort{Status: http.StatusBadRequest, Daten: []byte(`{"error":"Nummer unlesbar"}`)}) {
		t.Fatal("die Antwort aus der Ablage gilt als unlesbar")
	}
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "Nummer unlesbar") {
		t.Errorf("Status %d, Rumpf %s", w.Code, w.Body.String())
	}
}
