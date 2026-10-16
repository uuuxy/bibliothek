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

func TestVerarbeiteAusleihe_Schueler(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() = %v", err)
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

	tx := beginneTx(t, mock)

	uuidCopy := "123e4567-e89b-12d3-a456-426614174000"
	exemplar := &repository.Exemplar{ID: uuidCopy, TitelID: "titel1"}
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

	result, err := svc.verarbeiteAusleihe(context.Background(), tx, exemplar, chkCtx, staffID, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("verarbeiteAusleihe(…) = nil, erwartet ein Ergebnis")
	}
	if result.Art != "ausleihe" {
		t.Errorf("result.Art = %s, erwartet type ausleihe", result.Art)
	}
	if result.Leser == nil || result.Leser.ID != "student1" {
		t.Errorf("verarbeiteAusleihe(…).Leser = %v, erwartet student1", result.Leser)
	}
}

func TestVerarbeiteAusleihe_Lehrkraft(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() = %v", err)
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

	tx := beginneTx(t, mock)

	uuidCopy := "123e4567-e89b-12d3-a456-426614174000"
	exemplar := &repository.Exemplar{ID: uuidCopy, TitelID: "titel1"}
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

	result, err := svc.verarbeiteAusleihe(context.Background(), tx, exemplar, chkCtx, staffID, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("verarbeiteAusleihe(…) = nil, erwartet ein Ergebnis")
	}
	if result.Art != "ausleihe" {
		t.Errorf("result.Art = %s, erwartet type ausleihe", result.Art)
	}
	if result.Leser == nil || result.Leser.ID != "teacher1" {
		t.Errorf("die Lehrkraft muss als Leser in der Antwort stehen")
	}
}

func TestVerarbeiteAusleihe_AnlegeFehler(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() = %v", err)
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

	tx := beginneTx(t, mock)

	uuidCopy := "123e4567-e89b-12d3-a456-426614174000"
	exemplar := &repository.Exemplar{ID: uuidCopy, TitelID: "titel1"}
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

	result, err := svc.verarbeiteAusleihe(context.Background(), tx, exemplar, chkCtx, staffID, resp)

	if err == nil {
		t.Fatal("verarbeiteAusleihe(…) = nil, erwartet einen Fehler")
	}
	if result != nil {
		t.Fatalf("verarbeiteAusleihe(…) = %v, erwartet nil", result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestVerarbeiteAusleihe_CommitFehler(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() = %v", err)
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

	tx := beginneTx(t, mock)

	uuidCopy := "123e4567-e89b-12d3-a456-426614174000"
	exemplar := &repository.Exemplar{ID: uuidCopy, TitelID: "titel1"}
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

	// Der Protokolleintrag steht vor dem Commit in derselben Transaktion.
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs("ausleihen", "CHECKOUT", uuidCopy, ptr(staffID), "USER", nilStr, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	result, err := svc.verarbeiteAusleihe(context.Background(), tx, exemplar, chkCtx, staffID, resp)

	if err == nil {
		t.Fatal("verarbeiteAusleihe(…) = nil, erwartet den Fehler des Commits")
	}
	if result != nil {
		t.Fatalf("verarbeiteAusleihe(…) = %v, erwartet nil", result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
