package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v4"
)

func ptr[T any](v T) *T {
	return &v
}

func TestHandleNewLoan_Student_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	studentRepo := repository.NewStudentRepository(mock)
	bookRepo := repository.NewBookRepository(mock)
	loanRepo := repository.NewLoanRepository(mock)
	auditRepo := repository.NewAuditRepository(mock)
	svc := NewLoanService(mock, studentRepo, bookRepo, loanRepo, auditRepo).(*defaultLoanService)

	tx := beginTx(t, mock)

	uuidCopy := "123e4567-e89b-12d3-a456-426614174000"
	copy := &repository.BookCopy{ID: uuidCopy, TitelID: "titel1"}
	chkCtx := &checkoutContext{
		borrowerType: "student",
		borrowerID:   "student1",
		student:      &repository.Student{ID: "student1", Vorname: "Max"},
		dueTime:      time.Now().Add(14 * 24 * time.Hour),
	}
	staffID := "staff1"
	resp := &LoanResult{}

	var nilStr *string
	var nilTime *time.Time

	mock.ExpectQuery("INSERT INTO ausleihen").
		WithArgs(uuidCopy, "student1", chkCtx.dueTime, staffID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "exemplar_id", "schueler_id", "ausleiher_benutzer_id", "ausgeliehen_am", "rueckgabe_frist", "rueckgabe_am", "bearbeiter_id", "rueckgabe_bearbeiter_id", "ist_fremdrueckgabe", "ist_handapparat"}).
			AddRow("loan1", ptr(uuidCopy), ptr("student1"), nilStr, time.Now(), chkCtx.dueTime, nilTime, ptr(staffID), nilStr, false, false))

	mock.ExpectQuery("DELETE FROM vormerkungen").
		WithArgs("titel1", "student1").
		WillReturnRows(pgxmock.NewRows([]string{"bereitgestellt_exemplar_id"}).AddRow(nilStr))

	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("ausleihen", "CHECKOUT", uuidCopy, ptr(staffID), "USER", nilStr, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	mock.ExpectRollback()

	result, err := svc.handleNewLoan(context.Background(), tx, copy, chkCtx, staffID, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Type != "ausleihe" {
		t.Errorf("expected type ausleihe, got %s", result.Type)
	}
	if result.Student == nil || result.Student.ID != "student1" {
		t.Errorf("expected student in result")
	}
}

func TestHandleNewLoan_Teacher_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	studentRepo := repository.NewStudentRepository(mock)
	bookRepo := repository.NewBookRepository(mock)
	loanRepo := repository.NewLoanRepository(mock)
	auditRepo := repository.NewAuditRepository(mock)
	svc := NewLoanService(mock, studentRepo, bookRepo, loanRepo, auditRepo).(*defaultLoanService)

	tx := beginTx(t, mock)

	uuidCopy := "123e4567-e89b-12d3-a456-426614174000"
	copy := &repository.BookCopy{ID: uuidCopy, TitelID: "titel1"}
	chkCtx := &checkoutContext{
		borrowerType: "teacher",
		borrowerID:   "teacher1",
		teacher:      &repository.User{ID: "teacher1", Vorname: "Anna"},
		dueTime:      time.Now().Add(365 * 24 * time.Hour),
	}
	staffID := "staff1"
	resp := &LoanResult{}

	var nilStr *string
	var nilTime *time.Time

	mock.ExpectQuery("INSERT INTO ausleihen").
		WithArgs(uuidCopy, "teacher1", chkCtx.dueTime, staffID, true).
		WillReturnRows(pgxmock.NewRows([]string{"id", "exemplar_id", "schueler_id", "ausleiher_benutzer_id", "ausgeliehen_am", "rueckgabe_frist", "rueckgabe_am", "bearbeiter_id", "rueckgabe_bearbeiter_id", "ist_fremdrueckgabe", "ist_handapparat"}).
			AddRow("loan1", ptr(uuidCopy), nilStr, ptr("teacher1"), time.Now(), chkCtx.dueTime, nilTime, ptr(staffID), nilStr, false, true))

	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("ausleihen", "CHECKOUT", uuidCopy, ptr(staffID), "USER", nilStr, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()
	mock.ExpectRollback()

	result, err := svc.handleNewLoan(context.Background(), tx, copy, chkCtx, staffID, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Type != "ausleihe" {
		t.Errorf("expected type ausleihe, got %s", result.Type)
	}
	if result.Teacher == nil || result.Teacher.ID != "teacher1" {
		t.Errorf("expected teacher in result")
	}
}

func TestHandleNewLoan_ErzeugeAusleiheError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	studentRepo := repository.NewStudentRepository(mock)
	bookRepo := repository.NewBookRepository(mock)
	loanRepo := repository.NewLoanRepository(mock)
	auditRepo := repository.NewAuditRepository(mock)
	svc := NewLoanService(mock, studentRepo, bookRepo, loanRepo, auditRepo).(*defaultLoanService)

	tx := beginTx(t, mock)

	uuidCopy := "123e4567-e89b-12d3-a456-426614174000"
	copy := &repository.BookCopy{ID: uuidCopy, TitelID: "titel1"}
	chkCtx := &checkoutContext{
		borrowerType: "student",
		borrowerID:   "student1",
		student:      &repository.Student{ID: "student1", Vorname: "Max"},
		dueTime:      time.Now().Add(14 * 24 * time.Hour),
	}
	staffID := "staff1"
	resp := &LoanResult{}

	dbErr := errors.New("db error")

	mock.ExpectQuery("INSERT INTO ausleihen").
		WithArgs(uuidCopy, "student1", chkCtx.dueTime, staffID).
		WillReturnError(dbErr)

	result, err := svc.handleNewLoan(context.Background(), tx, copy, chkCtx, staffID, resp)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Fatalf("expected nil result, got %v", result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestHandleNewLoan_CommitError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	studentRepo := repository.NewStudentRepository(mock)
	bookRepo := repository.NewBookRepository(mock)
	loanRepo := repository.NewLoanRepository(mock)
	auditRepo := repository.NewAuditRepository(mock)
	svc := NewLoanService(mock, studentRepo, bookRepo, loanRepo, auditRepo).(*defaultLoanService)

	tx := beginTx(t, mock)

	uuidCopy := "123e4567-e89b-12d3-a456-426614174000"
	copy := &repository.BookCopy{ID: uuidCopy, TitelID: "titel1"}
	chkCtx := &checkoutContext{
		borrowerType: "teacher",
		borrowerID:   "teacher1",
		teacher:      &repository.User{ID: "teacher1", Vorname: "Anna"},
		dueTime:      time.Now().Add(365 * 24 * time.Hour),
	}
	staffID := "staff1"
	resp := &LoanResult{}

	var nilStr *string
	var nilTime *time.Time

	mock.ExpectQuery("INSERT INTO ausleihen").
		WithArgs(uuidCopy, "teacher1", chkCtx.dueTime, staffID, true).
		WillReturnRows(pgxmock.NewRows([]string{"id", "exemplar_id", "schueler_id", "ausleiher_benutzer_id", "ausgeliehen_am", "rueckgabe_frist", "rueckgabe_am", "bearbeiter_id", "rueckgabe_bearbeiter_id", "ist_fremdrueckgabe", "ist_handapparat"}).
			AddRow("loan1", ptr(uuidCopy), nilStr, ptr("teacher1"), time.Now(), chkCtx.dueTime, nilTime, ptr(staffID), nilStr, false, true))

	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	result, err := svc.handleNewLoan(context.Background(), tx, copy, chkCtx, staffID, resp)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Fatalf("expected nil result, got %v", result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
