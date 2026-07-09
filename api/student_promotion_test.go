package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v4"
)

func promotionTestServer(t *testing.T) (*Server, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	t.Cleanup(mock.Close)
	return &Server{DB: &db.Database{Pool: mock}}, mock
}

func doPromote(t *testing.T, s *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/students/promote", strings.NewReader(body))
	ctx := context.WithValue(req.Context(), auth.ClaimsContextKey, &auth.Claims{UserID: "admin-1", Rolle: auth.RoleAdmin})
	rec := httptest.NewRecorder()
	s.PromoteStudentsHandler()(rec, req.WithContext(ctx))
	return rec
}

func TestPromoteStudents_RequiresConfirm(t *testing.T) {
	s, mock := promotionTestServer(t)

	rec := doPromote(t, s, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("ohne confirm: erwartet 400, bekam %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("DB darf ohne Bestätigung nicht berührt werden: %v", err)
	}
}

func TestPromoteStudents_DryRunRollsBackAndSkipsGuardAndAudit(t *testing.T) {
	s, mock := promotionTestServer(t)

	// Dry-Run: NUR Begin → CTE → Rollback. Kein Doppellauf-Guard, kein
	// Audit-Insert, kein Commit — sonst wäre die „Vorschau" keine.
	mock.ExpectBegin()
	mock.ExpectQuery(`WITH parsed AS`).
		WillReturnRows(pgxmock.NewRows([]string{"versetzt", "abgaenger"}).AddRow(120, 25))
	mock.ExpectRollback()

	rec := doPromote(t, s, `{"dry_run": true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"promoted_count":120`) || !strings.Contains(body, `"dry_run":true`) {
		t.Errorf("Dry-Run-Antwort falsch: %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestPromoteStudents_SecondRunWithinWindowReturns409(t *testing.T) {
	s, mock := promotionTestServer(t)

	// Ein Lauf vor <10 min existiert → 409, ohne dass das UPDATE läuft.
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectRollback()

	rec := doPromote(t, s, `{"confirm": true}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("erwartet 409 bei Doppellauf, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestPromoteStudents_CommitPathWritesAuditLog(t *testing.T) {
	s, mock := promotionTestServer(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`WITH parsed AS`).
		WillReturnRows(pgxmock.NewRows([]string{"versetzt", "abgaenger"}).AddRow(300, 42))
	mock.ExpectExec(`INSERT INTO audit_logs`).
		WithArgs("admin-1", `{"versetzt": 300, "abgaenger": 42}`, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	rec := doPromote(t, s, `{"confirm": true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"archived_count":42`) || !strings.Contains(body, `"dry_run":false`) {
		t.Errorf("Antwort falsch: %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}
