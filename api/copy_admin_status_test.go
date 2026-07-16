package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v4"
)

// Diese Tests halten den achten Aussonderungs-Schreibpfad fest, der bei der Umstellung
// auf chk_aussonderung_grund (Migration 043) übersehen wurde: Der Status-Editor
// schreibt ist_ausgesondert PARAMETRISIERT — als einziger Pfad neben den sieben mit
// festem Grund. Ohne Mitführung von aussonderung_grund lehnte der CHECK jedes
// Aussondern (Grund bliebe NULL) und jedes Reaktivieren (Grund bliebe stehen) mit
// einem 500 ab. Der Constraint selbst ist in db/constraints_aussonderung_pg_test.go
// gegen echtes Postgres abgesichert; hier wird gepinnt, dass der Code-Pfad das
// Grund-Feld tatsächlich mitschreibt.
//
// Das Regex prüft bewusst den CASE-Ausdruck: WHEN $2 (aussondern) muss einen Grund
// setzen, ohne einen vorhandenen (z. B. VERLUST aus der Inventur) zu überschreiben;
// ELSE (reaktivieren) muss ihn löschen.
const updateCopyStatusPattern = `UPDATE buecher_exemplare\s+SET ist_ausleihbar = \$1,\s+ist_ausgesondert = \$2,\s+aussonderung_grund = CASE\s+WHEN \$2 THEN COALESCE\(aussonderung_grund, 'AUSSORTIERT'\)\s+ELSE NULL\s+END`

func neuerCopyStatusAufbau(t *testing.T) (pgxmock.PgxPoolIface, http.HandlerFunc) {
	t.Helper()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Mock-Pool konnte nicht erstellt werden: %v", err)
	}
	t.Cleanup(mock.Close)

	server := &Server{}
	return mock, server.UpdateCopyStatusHandler(repository.NewBookRepository(mock))
}

func sendeStatusUpdate(t *testing.T, handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPut, "/api/buecher/exemplare/ex-1/status", strings.NewReader(body))
	req.SetPathValue("id", "ex-1")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestUpdateCopyStatus_AussondernFuehrtGrundMit(t *testing.T) {
	mock, handler := neuerCopyStatusAufbau(t)

	mock.ExpectExec(updateCopyStatusPattern).
		WithArgs(false, true, "Wasserschaden", "ex-1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	rec := sendeStatusUpdate(t, handler, `{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":"Wasserschaden"}`)

	if rec.Code != http.StatusOK {
		t.Errorf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Aussondern schreibt aussonderung_grund nicht mit: %v", err)
	}
}

func TestUpdateCopyStatus_ReaktivierenLoeschtGrund(t *testing.T) {
	mock, handler := neuerCopyStatusAufbau(t)

	// Der Handler erzwingt bei ist_ausleihbar=true den Weg zurück in den Umlauf
	// (ist_ausgesondert=false, Notiz geleert) — der ELSE-Zweig muss den Grund räumen.
	mock.ExpectExec(updateCopyStatusPattern).
		WithArgs(true, false, "", "ex-1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	rec := sendeStatusUpdate(t, handler, `{"ist_ausleihbar":true,"ist_ausgesondert":true,"zustand_notiz":"war mal Verlust"}`)

	if rec.Code != http.StatusOK {
		t.Errorf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Reaktivieren räumt aussonderung_grund nicht: %v", err)
	}
}
