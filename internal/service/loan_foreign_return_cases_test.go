package service

import (
	"context"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

type mockLoanRepo struct {
	repository.LoanRepository // embed to satisfy interface methods we don't mock
	returnLoanTxErr           error
}

func (m *mockLoanRepo) ReturnLoanTx(ctx context.Context, tx pgx.Tx, loanID string, bearbeiterID string, isFremdrueckgabe bool) error {
	return m.returnLoanTxErr
}

func TestHandleForeignReturn_Student(t *testing.T) {
	svc, _, mockPool := newValidationService(t, &repository.Student{
		ID: "s1", Vorname: "Max", Nachname: "Mustermann",
	})
	defer mockPool.Close()

	mockLoan := &mockLoanRepo{}
	svc.loanRepo = mockLoan

	tx := beginTx(t, mockPool)

	activeLoan := &repository.Loan{
		ID:         "loan1",
		SchuelerID: strPtr("s1"),
	}
	copy := &repository.BookCopy{
		ID:        "copy1",
		BarcodeID: "B-C1",
		Titel:     "Testbuch",
	}

	resp := &LoanResult{}

	// mock processReturnVormerkungTx internals
	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(pgx.ErrNoRows)
	mockPool.ExpectCommit()

	res, err := svc.handleForeignReturn(context.Background(), tx, copy, activeLoan, "staff1", resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Type != "rueckgabe" {
		t.Errorf("expected resp.Type to be 'rueckgabe', got %s", res.Type)
	}
	if res.Vorbesitzer == nil || res.Vorbesitzer.ID != "s1" {
		t.Errorf("expected res.Vorbesitzer to be s1")
	}

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestHandleForeignReturn_Teacher(t *testing.T) {
	svc, _, mockPool := newValidationService(t, nil)
	defer mockPool.Close()

	mockLoan := &mockLoanRepo{}
	svc.loanRepo = mockLoan

	tx := beginTx(t, mockPool)

	activeLoan := &repository.Loan{
		ID:                  "loan2",
		AusleiherBenutzerID: strPtr("t1"),
	}
	copy := &repository.BookCopy{
		ID:        "copy2",
		BarcodeID: "B-C2",
		Titel:     "Lehrerbuch",
	}

	resp := &LoanResult{}

	// resolve prevTeacher
	mockPool.ExpectQuery("SELECT b.id, b.vorname, b.nachname, b.rolle::text FROM benutzer b WHERE b.id = \\$1 LIMIT 1").
		WithArgs("t1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "vorname", "nachname", "rolle"}).AddRow("t1", "Hans", "Lehrer", "KOLLEGIUM"))

	// mock processReturnVormerkungTx internals
	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(pgx.ErrNoRows)
	mockPool.ExpectCommit()

	res, err := svc.handleForeignReturn(context.Background(), tx, copy, activeLoan, "staff1", resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Type != "rueckgabe" {
		t.Errorf("expected resp.Type to be 'rueckgabe', got %s", res.Type)
	}
	if res.VorbesitzerUser == nil || res.VorbesitzerUser.ID != "t1" {
		t.Errorf("expected res.VorbesitzerUser to be t1")
	}

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestHandleForeignReturn_ReturnLoanError(t *testing.T) {
	svc, _, mockPool := newValidationService(t, &repository.Student{
		ID: "s1", Vorname: "Max", Nachname: "Mustermann",
	})
	defer mockPool.Close()

	// Return error
	mockLoan := &mockLoanRepo{returnLoanTxErr: pgx.ErrTxClosed}
	svc.loanRepo = mockLoan

	tx := beginTx(t, mockPool)

	activeLoan := &repository.Loan{
		ID:         "loan3",
		SchuelerID: strPtr("s1"),
	}
	copy := &repository.BookCopy{
		ID: "copy3",
	}

	resp := &LoanResult{}

	res, err := svc.handleForeignReturn(context.Background(), tx, copy, activeLoan, "staff1", resp)

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if res != nil {
		t.Errorf("expected res to be nil, got %v", res)
	}
}

func TestHandleForeignReturn_CommitError(t *testing.T) {
	svc, _, mockPool := newValidationService(t, &repository.Student{
		ID: "s1", Vorname: "Max", Nachname: "Mustermann",
	})
	defer mockPool.Close()

	mockLoan := &mockLoanRepo{}
	svc.loanRepo = mockLoan

	tx := beginTx(t, mockPool)

	activeLoan := &repository.Loan{
		ID:         "loan4",
		SchuelerID: strPtr("s1"),
	}
	copy := &repository.BookCopy{
		ID:        "copy4",
		BarcodeID: "B-C4",
		Titel:     "Testbuch",
	}

	resp := &LoanResult{}

	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(pgx.ErrNoRows)
	mockPool.ExpectCommit().WillReturnError(pgx.ErrTxClosed)

	res, err := svc.handleForeignReturn(context.Background(), tx, copy, activeLoan, "staff1", resp)

	if err == nil {
		t.Fatalf("expected commit error, got nil")
	}
	if res != nil {
		t.Errorf("expected res to be nil, got %v", res)
	}

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
