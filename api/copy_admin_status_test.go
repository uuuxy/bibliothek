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
// Das Regex prüft bewusst den CASE-Ausdruck: WHEN $2 (aussondern) muss den Default-Grund
// VERLUST setzen ("Verloren" zählt in der Verlustquote), ohne einen vorhandenen,
// spezifischeren (z. B. BESCHAEDIGUNG aus der Schadensmeldung) zu überschreiben;
// ELSE (reaktivieren) muss ihn löschen.
const updateCopyStatusPattern = `UPDATE buecher_exemplare\s+SET ist_ausleihbar = \$1,\s+ist_ausgesondert = \$2,\s+aussonderung_grund = CASE\s+WHEN \$2 THEN COALESCE\(aussonderung_grund, 'VERLUST'\)\s+ELSE NULL\s+END`

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
		WithArgs(false, true, "Wasserschaden", "ex-1", (*int)(nil)).
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
		WithArgs(true, false, "", "ex-1", (*int)(nil)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	rec := sendeStatusUpdate(t, handler, `{"ist_ausleihbar":true,"ist_ausgesondert":true,"zustand_notiz":"war mal Verlust"}`)

	if rec.Code != http.StatusOK {
		t.Errorf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Reaktivieren räumt aussonderung_grund nicht: %v", err)
	}
}

// Der Beschädigungsgrad (Migration 127) fährt im Status-Editor mit — und das Feld hat
// DREI Zustände, nicht zwei.
//
// Das Muster prüft die COALESCE-Zeile selbst: Stünde dort ein nacktes $5, wäre jeder
// Aufruf ohne das Feld eine stille 0 — ein erfasster Wasserschaden verschwände beim
// Sperren oder Freigeben, und die Schule verlangte beim nächsten Verlust wieder den
// vollen Zeitwert (Bugklasse Upsert-Blanking).
const updateCopyAbwertungPattern = `zustand_abwertung_prozent = COALESCE\(\$5::smallint, zustand_abwertung_prozent\)`

func TestUpdateCopyStatus_BeschaedigungsgradWirdGeschrieben(t *testing.T) {
	mock, handler := neuerCopyStatusAufbau(t)

	zwanzig := 20
	mock.ExpectExec(updateCopyAbwertungPattern).
		WithArgs(false, false, "Wasserrand, lesbar", "ex-1", &zwanzig).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	rec := sendeStatusUpdate(t, handler,
		`{"ist_ausleihbar":false,"ist_ausgesondert":false,"zustand_notiz":"Wasserrand, lesbar","zustand_abwertung_prozent":20}`)

	if rec.Code != http.StatusOK {
		t.Errorf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("der Beschädigungsgrad kommt nicht an der Spalte an: %v", err)
	}
}

// Fehlt das Feld, muss nil ankommen — nicht 0. Das ist der Fall JEDES Aufrufers, der
// den Beschädigungsgrad gar nicht kennt.
func TestUpdateCopyStatus_OhneFeldBleibtDerGradUnangetastet(t *testing.T) {
	mock, handler := neuerCopyStatusAufbau(t)

	mock.ExpectExec(updateCopyAbwertungPattern).
		WithArgs(false, false, "gesperrt", "ex-1", (*int)(nil)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	rec := sendeStatusUpdate(t, handler,
		`{"ist_ausleihbar":false,"ist_ausgesondert":false,"zustand_notiz":"gesperrt"}`)

	if rec.Code != http.StatusOK {
		t.Errorf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("ein fehlendes Feld kommt nicht als nil an — dann ist es eine stille 0: %v", err)
	}
}

// „Verfügbar" räumt die Notiz (bestehende Regel), aber NICHT den Beschädigungsgrad: Ein
// Band mit Wasserrand darf ausleihbar sein und trägt seinen Abschlag weiter.
func TestUpdateCopyStatus_VerfuegbarBehaeltDenBeschaedigungsgrad(t *testing.T) {
	mock, handler := neuerCopyStatusAufbau(t)

	dreissig := 30
	mock.ExpectExec(updateCopyAbwertungPattern).
		WithArgs(true, false, "", "ex-1", &dreissig).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	rec := sendeStatusUpdate(t, handler,
		`{"ist_ausleihbar":true,"ist_ausgesondert":false,"zustand_notiz":"Wasserrand","zustand_abwertung_prozent":30}`)

	if rec.Code != http.StatusOK {
		t.Errorf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Freigeben räumt den Beschädigungsgrad — er ist eine Eigenschaft des "+
			"Buchs, kein Status: %v", err)
	}
}

// Ein unmöglicher Grad ist eine Auskunft, kein 500. Die Spalte lehnt ihn ohnehin ab
// (chk_zustand_abwertung_bereich); ohne Prüfung hier käme der CHECK-Fehler als
// „Serverfehler" zurück und niemand wüsste, was erlaubt ist.
func TestUpdateCopyStatus_UnmoeglicherGradIstEin400(t *testing.T) {
	for _, koerper := range []string{
		`{"ist_ausleihbar":false,"ist_ausgesondert":false,"zustand_notiz":"x","zustand_abwertung_prozent":140}`,
		`{"ist_ausleihbar":false,"ist_ausgesondert":false,"zustand_notiz":"x","zustand_abwertung_prozent":-5}`,
	} {
		mock, handler := neuerCopyStatusAufbau(t)
		rec := sendeStatusUpdate(t, handler, koerper)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s → %d, erwartet 400: %s", koerper, rec.Code, rec.Body.String())
		}
		// Kein Exec erwartet: Der Aufruf darf die Datenbank nicht erreichen.
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unerwarteter DB-Zugriff: %v", err)
		}
	}
}
