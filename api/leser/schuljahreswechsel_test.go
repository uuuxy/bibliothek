package leser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func TestSchuljahreswechsel_ProbelaufRolltZurueckOhneSperreUndProtokoll(t *testing.T) {
	s, mock := schuljahreswechselTueren(t)

	// Probelauf: nur Begin → CTE → Rollback. Keine Sperre gegen den Doppellauf, kein
	// Protokolleintrag, kein Commit; sonst wäre die „Vorschau" keine.
	mock.ExpectBegin()
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
	if !strings.Contains(body, `"promoted_count":120`) || !strings.Contains(body, `"dry_run":true`) {
		t.Errorf("Dry-Run-Antwort falsch: %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestSchuljahreswechsel_ZweiterLaufImZeitfensterIst409(t *testing.T) {
	s, mock := schuljahreswechselTueren(t)

	// Ein Lauf innerhalb des Schutzfensters (12 h) existiert → 409, ohne dass das UPDATE läuft.
	mock.ExpectBegin()
	mock.ExpectExec(`pg_advisory_xact_lock`).WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectRollback()

	rec := rufeSchuljahreswechsel(t, s, `{"confirm": true}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("rec.Code = %d, erwartet 409 bei Doppellauf: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestSchuljahreswechsel_EchterLaufSchreibtProtokoll(t *testing.T) {
	s, mock := schuljahreswechselTueren(t)

	mock.ExpectBegin()
	mock.ExpectExec(`pg_advisory_xact_lock`).WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
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
