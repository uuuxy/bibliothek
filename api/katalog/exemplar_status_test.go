package katalog

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// Diese Tests halten einen der acht Wege fest, die ein Exemplar aussondern: Der Status-Editor
// schreibt ist_ausgesondert als Parameter, als einziger neben sieben Wegen mit festem Grund.
// Schriebe er aussonderung_grund nicht mit, lehnte chk_aussonderung_grund (Migration 043) jedes
// Aussondern (der Grund bliebe NULL) und jedes Reaktivieren (der Grund bliebe stehen) mit einem
// 500 ab. Die Bedingung selbst sichert db/constraints_aussonderung_pg_test.go an der Datenbank;
// hier wird festgehalten, dass der Code das Feld tatsächlich mitschreibt.
//
// Der reguläre Ausdruck prüft den CASE-Ausdruck: WHEN $2 (aussondern) muss den Grund
// VERLUST setzen ("Verloren" zählt in der Verlustquote), ohne einen vorhandenen,
// spezifischeren (z. B. BESCHAEDIGUNG aus der Schadensmeldung) zu überschreiben;
// ELSE (reaktivieren) muss ihn löschen.
const exemplarStatusMuster = `UPDATE buecher_exemplare\s+SET ist_ausleihbar = \$1,\s+ist_ausgesondert = \$2,\s+aussonderung_grund = CASE\s+WHEN \$2 THEN COALESCE\(aussonderung_grund, 'VERLUST'\)\s+ELSE NULL\s+END`

func baueExemplarStatusTuer(t *testing.T) (pgxmock.PgxPoolIface, http.HandlerFunc) {
	t.Helper()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Mock-Pool konnte nicht erstellt werden: %v", err)
	}
	t.Cleanup(mock.Close)

	server := &Tueren{Basis: &tuer.Basis{}}
	return mock, server.ExemplarStatusAendern(repository.NewKatalogAbfragen(mock), repository.NewBescheidAbfragen(mock))
}

func sendeExemplarStatus(t *testing.T, handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPut, "/api/buecher/exemplare/ex-1/status", strings.NewReader(body))
	req.SetPathValue("id", "ex-1")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, tuertest.AlsBenutzer(req, statusBearbeiter))
	return rec
}

// statusBearbeiter ist die Person, unter der die Tests speichern.
const statusBearbeiter = "bearbeiter-1"

// erwarteStatusSpeichern erwartet den Ablauf von AendereExemplarStatus bis zur Anweisung: die
// Transaktion, den Stand vor dem Wechsel unter Sperre und die Anweisung selbst.
func erwarteStatusSpeichern(mock pgxmock.PgxPoolIface, muster string, args ...any) {
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT ist_ausgesondert FROM buecher_exemplare WHERE id = \$1 FOR UPDATE`).
		WithArgs("ex-1").
		WillReturnRows(pgxmock.NewRows([]string{"ist_ausgesondert"}).AddRow(false))
	mock.ExpectExec(muster).WithArgs(args...).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
}

func TestExemplarStatusAendern_AussondernFuehrtGrundMit(t *testing.T) {
	mock, handler := baueExemplarStatusTuer(t)

	erwarteStatusSpeichern(mock, exemplarStatusMuster, false, true, "Wasserschaden", "ex-1", (*int)(nil))
	// Der Wechsel in „ausgesondert" steht mit der Person im Protokoll, ohne die Notiz.
	mock.ExpectExec(`INSERT INTO audit_log`).
		WithArgs([]string{"ex-1"}, statusBearbeiter, repository.AussonderungsWegStatus, repository.ProtokollAktionAusgesondert, "{}").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	rec := sendeExemplarStatus(t, handler, `{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":"Wasserschaden"}`)

	if rec.Code != http.StatusOK {
		t.Errorf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Aussondern schreibt aussonderung_grund nicht mit: %v", err)
	}
}

func TestExemplarStatusAendern_ReaktivierenLoeschtGrund(t *testing.T) {
	mock, handler := baueExemplarStatusTuer(t)

	// Die Tür erzwingt bei ist_ausleihbar=true den Weg zurück in den Umlauf
	// (ist_ausgesondert=false, Notiz geleert) — der ELSE-Zweig muss den Grund räumen.
	erwarteStatusSpeichern(mock, exemplarStatusMuster, true, false, "", "ex-1", (*int)(nil))
	mock.ExpectCommit()

	rec := sendeExemplarStatus(t, handler, `{"ist_ausleihbar":true,"ist_ausgesondert":true,"zustand_notiz":"war mal Verlust"}`)

	if rec.Code != http.StatusOK {
		t.Errorf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Reaktivieren räumt aussonderung_grund nicht: %v", err)
	}
}

// Der Beschädigungsgrad (Migration 127) fährt im Status-Editor mit — und das Feld hat
// drei Zustände, nicht zwei.
//
// Das Muster prüft die COALESCE-Zeile selbst: Stünde dort ein nacktes $5, wäre jeder
// Aufruf ohne das Feld eine stille 0 — ein erfasster Wasserschaden verschwände beim
// Sperren oder Freigeben, und die Schule verlangte beim nächsten Verlust wieder den
// vollen Zeitwert (Bugklasse Upsert-Blanking).
const exemplarAbwertungMuster = `zustand_abwertung_prozent = COALESCE\(\$5::smallint, zustand_abwertung_prozent\)`

func TestExemplarStatusAendern_BeschaedigungsgradWirdGeschrieben(t *testing.T) {
	mock, handler := baueExemplarStatusTuer(t)

	zwanzig := 20
	erwarteStatusSpeichern(mock, exemplarAbwertungMuster, false, false, "Wasserrand, lesbar", "ex-1", &zwanzig)
	mock.ExpectCommit()

	rec := sendeExemplarStatus(t, handler,
		`{"ist_ausleihbar":false,"ist_ausgesondert":false,"zustand_notiz":"Wasserrand, lesbar","zustand_abwertung_prozent":20}`)

	if rec.Code != http.StatusOK {
		t.Errorf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("der Beschädigungsgrad kommt nicht an der Spalte an: %v", err)
	}
}

// Fehlt das Feld, muss nil ankommen — nicht 0. Das ist der Fall jedes Aufrufers, der
// den Beschädigungsgrad gar nicht kennt.
func TestExemplarStatusAendern_OhneFeldBleibtDerGradUnangetastet(t *testing.T) {
	mock, handler := baueExemplarStatusTuer(t)

	erwarteStatusSpeichern(mock, exemplarAbwertungMuster, false, false, "gesperrt", "ex-1", (*int)(nil))
	mock.ExpectCommit()

	rec := sendeExemplarStatus(t, handler,
		`{"ist_ausleihbar":false,"ist_ausgesondert":false,"zustand_notiz":"gesperrt"}`)

	if rec.Code != http.StatusOK {
		t.Errorf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("ein fehlendes Feld kommt nicht als nil an — dann ist es eine stille 0: %v", err)
	}
}

// „Verfügbar" räumt die Notiz (bestehende Regel), aber nicht den Beschädigungsgrad: Ein
// Band mit Wasserrand darf ausleihbar sein und trägt seinen Abschlag weiter.
func TestExemplarStatusAendern_VerfuegbarBehaeltDenBeschaedigungsgrad(t *testing.T) {
	mock, handler := baueExemplarStatusTuer(t)

	dreissig := 30
	erwarteStatusSpeichern(mock, exemplarAbwertungMuster, true, false, "", "ex-1", &dreissig)
	mock.ExpectCommit()

	rec := sendeExemplarStatus(t, handler,
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
func TestExemplarStatusAendern_UnmoeglicherGradIstEin400(t *testing.T) {
	for _, koerper := range []string{
		`{"ist_ausleihbar":false,"ist_ausgesondert":false,"zustand_notiz":"x","zustand_abwertung_prozent":140}`,
		`{"ist_ausleihbar":false,"ist_ausgesondert":false,"zustand_notiz":"x","zustand_abwertung_prozent":-5}`,
	} {
		mock, handler := baueExemplarStatusTuer(t)
		rec := sendeExemplarStatus(t, handler, koerper)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s → %d, erwartet 400: %s", koerper, rec.Code, rec.Body.String())
		}
		// Kein Exec erwartet: Der Aufruf darf die Datenbank nicht erreichen.
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unerwarteter DB-Zugriff: %v", err)
		}
	}
}
