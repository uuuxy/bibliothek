package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

func ptr[T any](v T) *T {
	return &v
}

func TestVerarbeiteAusleihe_Student_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	studentRepo := repository.NewLeserAbfragen(mock)
	bookRepo := repository.NewKatalogAbfragen(mock)
	loanRepo := repository.NewAusleihAbfragen(mock)
	auditRepo := repository.NewProtokollAbfragen(mock)
	svc, ok := NewAusleihDienst(mock, studentRepo, bookRepo, loanRepo, auditRepo).(*ausleihDienst)
	if !ok {
		t.Fatal("NewAusleihDienst liefert keinen *ausleihDienst")
	}

	tx := beginTx(t, mock)

	uuidCopy := "123e4567-e89b-12d3-a456-426614174000"
	copy := &repository.Exemplar{ID: uuidCopy, TitelID: "titel1"}
	chkCtx := &ausleihVorgang{
		leserID: "student1",
		leser:   &repository.Leser{ID: "student1", Vorname: "Max", Art: "schueler"},
		frist:   time.Now().Add(14 * 24 * time.Hour),
	}
	staffID := "staff1"
	resp := &AusleihErgebnis{}

	var nilStr *string
	var nilTime *time.Time

	mock.ExpectQuery("INSERT INTO ausleihen").
		WithArgs(uuidCopy, "student1", chkCtx.frist, staffID, false, pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "exemplar_id", "schueler_id", "ausgeliehen_am", "rueckgabe_frist", "rueckgabe_am", "bearbeiter_id", "rueckgabe_bearbeiter_id", "ist_fremdrueckgabe", "ist_handapparat"}).
			AddRow("loan1", ptr(uuidCopy), ptr("student1"), time.Now(), chkCtx.frist, nilTime, ptr(staffID), nilStr, false, false))
	mock.ExpectExec("UPDATE buecher_exemplare SET letzte_bewegung_am").
		WithArgs(uuidCopy, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectQuery("DELETE FROM vormerkungen").
		WithArgs("titel1", "student1").
		WillReturnRows(pgxmock.NewRows([]string{"bereitgestellt_exemplar_id"}).AddRow(nilStr))

	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("ausleihen", "CHECKOUT", uuidCopy, ptr(staffID), "USER", nilStr, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	mock.ExpectRollback()

	result, err := svc.verarbeiteAusleihe(context.Background(), tx, copy, chkCtx, staffID, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Art != "ausleihe" {
		t.Errorf("expected type ausleihe, got %s", result.Art)
	}
	if result.Leser == nil || result.Leser.ID != "student1" {
		t.Errorf("expected student in result")
	}
}

func TestVerarbeiteAusleihe_Teacher_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	studentRepo := repository.NewLeserAbfragen(mock)
	bookRepo := repository.NewKatalogAbfragen(mock)
	loanRepo := repository.NewAusleihAbfragen(mock)
	auditRepo := repository.NewProtokollAbfragen(mock)
	svc, ok := NewAusleihDienst(mock, studentRepo, bookRepo, loanRepo, auditRepo).(*ausleihDienst)
	if !ok {
		t.Fatal("NewAusleihDienst liefert keinen *ausleihDienst")
	}

	tx := beginTx(t, mock)

	uuidCopy := "123e4567-e89b-12d3-a456-426614174000"
	copy := &repository.Exemplar{ID: uuidCopy, TitelID: "titel1"}
	chkCtx := &ausleihVorgang{
		leserID: "teacher1",
		leser:   &repository.Leser{ID: "teacher1", Vorname: "Anna", Art: "lehrkraft"},
		frist:   time.Now().Add(365 * 24 * time.Hour),
	}
	staffID := "staff1"
	resp := &AusleihErgebnis{}

	var nilStr *string
	var nilTime *time.Time

	mock.ExpectQuery("INSERT INTO ausleihen").
		WithArgs(uuidCopy, "teacher1", chkCtx.frist, staffID, true, pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "exemplar_id", "schueler_id", "ausgeliehen_am", "rueckgabe_frist", "rueckgabe_am", "bearbeiter_id", "rueckgabe_bearbeiter_id", "ist_fremdrueckgabe", "ist_handapparat"}).
			AddRow("loan1", ptr(uuidCopy), ptr("teacher1"), time.Now(), chkCtx.frist, nilTime, ptr(staffID), nilStr, false, true))
	mock.ExpectExec("UPDATE buecher_exemplare SET letzte_bewegung_am").
		WithArgs(uuidCopy, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("ausleihen", "CHECKOUT", uuidCopy, ptr(staffID), "USER", nilStr, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()
	mock.ExpectRollback()

	result, err := svc.verarbeiteAusleihe(context.Background(), tx, copy, chkCtx, staffID, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Art != "ausleihe" {
		t.Errorf("expected type ausleihe, got %s", result.Art)
	}
	if result.Leser == nil || result.Leser.ID != "teacher1" {
		t.Errorf("die Lehrkraft muss als Leser in der Antwort stehen")
	}
}

func TestVerarbeiteAusleihe_ErzeugeAusleiheError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	studentRepo := repository.NewLeserAbfragen(mock)
	bookRepo := repository.NewKatalogAbfragen(mock)
	loanRepo := repository.NewAusleihAbfragen(mock)
	auditRepo := repository.NewProtokollAbfragen(mock)
	svc, ok := NewAusleihDienst(mock, studentRepo, bookRepo, loanRepo, auditRepo).(*ausleihDienst)
	if !ok {
		t.Fatal("NewAusleihDienst liefert keinen *ausleihDienst")
	}

	tx := beginTx(t, mock)

	uuidCopy := "123e4567-e89b-12d3-a456-426614174000"
	copy := &repository.Exemplar{ID: uuidCopy, TitelID: "titel1"}
	chkCtx := &ausleihVorgang{
		leserID: "student1",
		leser:   &repository.Leser{ID: "student1", Vorname: "Max", Art: "schueler"},
		frist:   time.Now().Add(14 * 24 * time.Hour),
	}
	staffID := "staff1"
	resp := &AusleihErgebnis{}

	dbErr := errors.New("db error")

	mock.ExpectQuery("INSERT INTO ausleihen").
		WithArgs(uuidCopy, "student1", chkCtx.frist, staffID, false, pgxmock.AnyArg()).
		WillReturnError(dbErr)

	result, err := svc.verarbeiteAusleihe(context.Background(), tx, copy, chkCtx, staffID, resp)

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

func TestVerarbeiteAusleihe_CommitError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	studentRepo := repository.NewLeserAbfragen(mock)
	bookRepo := repository.NewKatalogAbfragen(mock)
	loanRepo := repository.NewAusleihAbfragen(mock)
	auditRepo := repository.NewProtokollAbfragen(mock)
	svc, ok := NewAusleihDienst(mock, studentRepo, bookRepo, loanRepo, auditRepo).(*ausleihDienst)
	if !ok {
		t.Fatal("NewAusleihDienst liefert keinen *ausleihDienst")
	}

	tx := beginTx(t, mock)

	uuidCopy := "123e4567-e89b-12d3-a456-426614174000"
	copy := &repository.Exemplar{ID: uuidCopy, TitelID: "titel1"}
	chkCtx := &ausleihVorgang{
		leserID: "teacher1",
		leser:   &repository.Leser{ID: "teacher1", Vorname: "Anna", Art: "lehrkraft"},
		frist:   time.Now().Add(365 * 24 * time.Hour),
	}
	staffID := "staff1"
	resp := &AusleihErgebnis{}

	var nilStr *string
	var nilTime *time.Time

	mock.ExpectQuery("INSERT INTO ausleihen").
		WithArgs(uuidCopy, "teacher1", chkCtx.frist, staffID, true, pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "exemplar_id", "schueler_id", "ausgeliehen_am", "rueckgabe_frist", "rueckgabe_am", "bearbeiter_id", "rueckgabe_bearbeiter_id", "ist_fremdrueckgabe", "ist_handapparat"}).
			AddRow("loan1", ptr(uuidCopy), ptr("teacher1"), time.Now(), chkCtx.frist, nilTime, ptr(staffID), nilStr, false, true))
	mock.ExpectExec("UPDATE buecher_exemplare SET letzte_bewegung_am").
		WithArgs(uuidCopy, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	// Die Audit-Zeile steht seit 07.09.2026 VOR dem Commit in derselben Transaktion.
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("ausleihen", "CHECKOUT", uuidCopy, ptr(staffID), "USER", nilStr, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	result, err := svc.verarbeiteAusleihe(context.Background(), tx, copy, chkCtx, staffID, resp)

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
