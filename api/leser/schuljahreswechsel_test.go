package leser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v5"
)

func schuljahreswechselTueren(t *testing.T) (*Tueren, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	t.Cleanup(mock.Close)
	return &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}, mock
}

func rufeSchuljahreswechsel(t *testing.T, s *Tueren, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/students/promote", strings.NewReader(body))
	ctx := context.WithValue(req.Context(), auth.ClaimsSchluessel, &auth.Claims{KontoID: "admin-1", Rolle: auth.RolleAdmin})
	rec := httptest.NewRecorder()
	s.Schuljahreswechsel()(rec, req.WithContext(ctx))
	return rec
}

func TestSchuljahreswechsel_VerlangtBestaetigung(t *testing.T) {
	s, mock := schuljahreswechselTueren(t)

	rec := rufeSchuljahreswechsel(t, s, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("ohne confirm = %d, erwartet 400", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("DB darf ohne Bestätigung nicht berührt werden: %v", err)
	}
}

func TestSchuljahreswechsel_ProbelaufRolltZurueckOhneProtokoll(t *testing.T) {
	s, mock := schuljahreswechselTueren(t)

	// Probelauf: Begin → Sperre → Prüfung auf einen jüngsten Lauf → Lesen → CTE → Rollback. Kein
	// Protokolleintrag, kein Commit; sonst wäre die „Vorschau" keine. Die Sperre lässt sie auf
	// einen laufenden Wechsel warten (fuehreSchuljahreswechselAus).
	mock.ExpectBegin()
	mock.ExpectExec(`pg_advisory_xact_lock`).WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery(`FROM audit_logs`).
		WithArgs("SCHULJAHRESWECHSEL", pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"zeitstempel", "naechster_ab"}))
	mock.ExpectQuery(`SELECT klasse FROM schueler`).
		WillReturnRows(pgxmock.NewRows([]string{"klasse"}).AddRow("14A"))
	mock.ExpectQuery(`WITH parsed AS`).
		WithArgs(pgxmock.AnyArg()). // Zone für abgaenger_jahr (schulzeit)
		WillReturnRows(pgxmock.NewRows([]string{"versetzt", "abgaenger"}).AddRow(120, 25))
	// Die Klassenlehrer-Zuordnung wandert mit; auch die Vorschau rechnet sie
	// durch (und rollt sie zurück).
	mock.ExpectQuery(`FROM klassen_lehrer_mapping`).
		WillReturnRows(pgxmock.NewRows([]string{"klasse", "neue_klasse", "abschluss"}))
	mock.ExpectRollback()

	rec := rufeSchuljahreswechsel(t, s, `{"dry_run": true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"promoted_count":120`) || !strings.Contains(body, `"dry_run":true`) ||
		!strings.Contains(body, `"klassen_ausserhalb":["14A"]`) {
		t.Errorf("Dry-Run-Antwort falsch: %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

// Ein Lauf vor weniger als sechs Monaten sperrt den nächsten, auch dessen Vorschau: Nach einem
// Lauf zeigte sie sonst die Zahlen eines zweiten unter „es wurde noch nichts geändert". Die
// Meldung nennt den Lauf und den Tag, ab dem der nächste geht, in der Uhr der Schule.
func TestSchuljahreswechsel_ZweiterLaufImAbstandIst409(t *testing.T) {
	sommer := time.Date(2026, time.July, 20, 8, 15, 0, 0, time.UTC)    // 10:15 Sommerzeit
	winter := time.Date(2027, time.January, 20, 9, 15, 0, 0, time.UTC) // 10:15 Winterzeit
	const meldung = "Der Schuljahreswechsel lief am 20.07.2026 um 10:15 Uhr. Ein zweiter Lauf würde alle " +
		"Schüler ein weiteres Mal versetzen; der nächste geht ab dem 20.01.2027 um 10:15 Uhr."
	for _, rumpf := range []string{`{"confirm": true}`, `{"dry_run": true}`} {
		t.Run(rumpf, func(t *testing.T) {
			s, mock := schuljahreswechselTueren(t)
			// Kein UPDATE, kein Protokolleintrag: Nach der Prüfung folgt nur das Rollback.
			mock.ExpectBegin()
			mock.ExpectExec(`pg_advisory_xact_lock`).WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("SELECT", 1))
			mock.ExpectQuery(`FROM audit_logs`).
				WithArgs("SCHULJAHRESWECHSEL", pgxmock.AnyArg()).
				WillReturnRows(pgxmock.NewRows([]string{"zeitstempel", "naechster_ab"}).AddRow(sommer, winter))
			mock.ExpectRollback()

			rec := rufeSchuljahreswechsel(t, s, rumpf)
			if rec.Code != http.StatusConflict {
				t.Errorf("rec.Code = %d, erwartet 409 bei Doppellauf: %s", rec.Code, rec.Body.String())
			}
			var antwort struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
				t.Fatalf("Antwort unlesbar: %v: %s", err, rec.Body.String())
			}
			if antwort.Error != meldung {
				t.Errorf("Meldung = %q, erwartet %q", antwort.Error, meldung)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("offene Erwartungen: %v", err)
			}
		})
	}
}

func TestSchuljahreswechsel_EchterLaufSchreibtProtokoll(t *testing.T) {
	s, mock := schuljahreswechselTueren(t)

	mock.ExpectBegin()
	mock.ExpectExec(`pg_advisory_xact_lock`).WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery(`FROM audit_logs`).
		WithArgs("SCHULJAHRESWECHSEL", pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"zeitstempel", "naechster_ab"}))
	mock.ExpectQuery(`SELECT klasse FROM schueler`).
		WillReturnRows(pgxmock.NewRows([]string{"klasse"}))
	mock.ExpectQuery(`WITH parsed AS`).
		WithArgs(pgxmock.AnyArg()). // Zone für abgaenger_jahr (schulzeit)
		WillReturnRows(pgxmock.NewRows([]string{"versetzt", "abgaenger"}).AddRow(300, 42))
	mock.ExpectQuery(`FROM klassen_lehrer_mapping`).
		WillReturnRows(pgxmock.NewRows([]string{"klasse", "neue_klasse", "abschluss"}))
	mock.ExpectExec(`INSERT INTO audit_logs`).
		WithArgs("admin-1", "SCHULJAHRESWECHSEL", `{"versetzt": 300, "abgaenger": 42}`, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	rec := rufeSchuljahreswechsel(t, s, `{"confirm": true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"archived_count":42`) || !strings.Contains(body, `"dry_run":false`) {
		t.Errorf("Antwort falsch: %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}
