package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/repository"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

type mockLoanRepoReturn struct {
	returnErr error
}

func (m *mockLoanRepoReturn) GetActiveLoanByCopyID(ctx context.Context, copyID string) (*repository.Loan, error) {
	return nil, nil
}
func (m *mockLoanRepoReturn) GetActiveLoanByCopyIDTx(ctx context.Context, tx pgx.Tx, copyID string) (*repository.Loan, error) {
	return nil, nil
}
func (m *mockLoanRepoReturn) BeginTx(ctx context.Context) (pgx.Tx, error) { return nil, nil }
func (m *mockLoanRepoReturn) CreateLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, leserID, bearbeiterID string, rueckgabeFrist time.Time, istDauerleihe bool) (*repository.Loan, error) {
	return nil, nil
}
func (m *mockLoanRepoReturn) ReturnLoanTx(ctx context.Context, tx pgx.Tx, loanID, bearbeiterID string, istVerlust bool) error {
	return m.returnErr
}
func (m *mockLoanRepoReturn) ZaehleAktiveAusleihenVonSchuelerTx(ctx context.Context, tx pgx.Tx, schuelerID string) (int, error) {
	return 0, nil
}
func (m *mockLoanRepoReturn) GetActiveBorrowingsByUserTx(ctx context.Context, tx pgx.Tx, userID string) ([]repository.Loan, error) {
	return nil, nil
}

func TestHandleRueckgabe_ReturnLoanTxError(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to init pgxmock: %v", err)
	}
	defer mockPool.Close()

	expectedErr := errors.New("db error")
	svc := &defaultLoanService{
		pool:        mockPool,
		loanRepo:    &mockLoanRepoReturn{returnErr: expectedErr},
		studentRepo: &mockStudentRepo{student: &repository.Student{ID: "leser1", Art: "lehrkraft"}},
	}

	mockPool.ExpectBegin()
	tx, err := mockPool.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	copy := &repository.BookCopy{ID: "copy1"}
	staffID := "staff1"
	// Der Ausleiher steht seit Migration 125 in EINER Spalte — auch, wenn es der
	// Mitarbeiter selbst ist, der das Buch zurückbringt.
	leserID := "leser1"
	activeLoan := &repository.Loan{ID: "loan1", SchuelerID: &leserID}
	resp := &LoanResult{}

	result, err := svc.handleRueckgabe(context.Background(), tx, copy, activeLoan, staffID, resp)

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
	if result != nil {
		t.Errorf("expected result to be nil, got %v", result)
	}
}

func TestHandleRueckgabe_CommitError(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to init pgxmock: %v", err)
	}
	defer mockPool.Close()

	audit := &mockAuditRepo{}
	svc := &defaultLoanService{
		pool:        mockPool,
		loanRepo:    &mockLoanRepoReturn{},
		auditRepo:   audit,
		studentRepo: &mockStudentRepo{student: &repository.Student{ID: "leser1", Art: "lehrkraft"}},
	}

	mockPool.ExpectBegin()
	tx, err := mockPool.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	leserID := "leser1"
	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE\\(s.klasse, ''\\)").
		WithArgs("t1", &leserID).
		WillReturnError(pgx.ErrNoRows)

	expectedErr := errors.New("commit error")
	mockPool.ExpectCommit().WillReturnError(expectedErr)

	copy := &repository.BookCopy{ID: "copy1", TitelID: "t1"}
	staffID := "staff1"
	// Der Ausleiher steht seit Migration 125 in EINER Spalte — auch, wenn es der
	// Mitarbeiter selbst ist, der das Buch zurückbringt.
	activeLoan := &repository.Loan{ID: "loan1", SchuelerID: &leserID}
	resp := &LoanResult{}

	result, err := svc.handleRueckgabe(context.Background(), tx, copy, activeLoan, staffID, resp)

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
	if result != nil {
		t.Errorf("expected result to be nil, got %v", result)
	}
}

func TestHandleRueckgabe_Success(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to init pgxmock: %v", err)
	}
	defer mockPool.Close()

	audit := &mockAuditRepo{}
	svc := &defaultLoanService{
		pool:        mockPool,
		loanRepo:    &mockLoanRepoReturn{},
		auditRepo:   audit,
		studentRepo: &mockStudentRepo{student: &repository.Student{ID: "leser1", Art: "lehrkraft"}},
	}

	mockPool.ExpectBegin()
	tx, err := mockPool.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	leserID := "leser1"
	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE\\(s.klasse, ''\\)").
		WithArgs("t1", &leserID).
		WillReturnError(pgx.ErrNoRows)

	mockPool.ExpectCommit()

	copy := &repository.BookCopy{ID: "c1", TitelID: "t1", BarcodeID: "b1", Titel: "Test Book"}
	staffID := "staff1"
	// Der Ausleiher steht seit Migration 125 in EINER Spalte — auch, wenn es der
	// Mitarbeiter selbst ist, der das Buch zurückbringt.
	activeLoan := &repository.Loan{ID: "loan1", SchuelerID: &leserID}
	resp := &LoanResult{}

	result, err := svc.handleRueckgabe(context.Background(), tx, copy, activeLoan, staffID, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Type != "rueckgabe" {
		t.Errorf("expected type rueckgabe, got %v", result.Type)
	}
	if result.LoanID == nil || *result.LoanID != "loan1" {
		t.Errorf("expected loan1, got %v", result.LoanID)
	}
}

func TestHandleRueckgabe_VormerkungAktiviert(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to init pgxmock: %v", err)
	}
	defer mockPool.Close()

	audit := &mockAuditRepo{}
	svc := &defaultLoanService{
		pool:        mockPool,
		loanRepo:    &mockLoanRepoReturn{},
		auditRepo:   audit,
		studentRepo: &mockStudentRepo{student: &repository.Student{ID: "leser1", Art: "lehrkraft"}},
	}

	mockPool.ExpectBegin()
	tx, err := mockPool.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	leserID := "leser1"
	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE\\(s.klasse, ''\\)").
		WithArgs("t1", &leserID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "vorname", "nachname", "klasse"}).
			AddRow("v1", "Max", "Mustermann", "10A"))

	// Status der Vormerkung auf 'abholbereit' setzen.
	mockPool.ExpectExec("UPDATE vormerkungen SET status = 'abholbereit'").
		WithArgs("c1", "v1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mockPool.ExpectCommit()

	copy := &repository.BookCopy{ID: "c1", TitelID: "t1", BarcodeID: "b1", Titel: "Test Book"}
	staffID := "staff1"
	// Der Ausleiher steht seit Migration 125 in EINER Spalte — auch, wenn es der
	// Mitarbeiter selbst ist, der das Buch zurückbringt.
	activeLoan := &repository.Loan{ID: "loan1", SchuelerID: &leserID}
	resp := &LoanResult{}

	result, err := svc.handleRueckgabe(context.Background(), tx, copy, activeLoan, staffID, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Type != "rueckgabe" {
		t.Errorf("expected type rueckgabe, got %v", result.Type)
	}
	if !result.HasVormerkung {
		t.Errorf("expected HasVormerkung to be true")
	}
	if result.VormerkungUser != "Max Mustermann, 10A" {
		t.Errorf("expected VormerkungUser to be Max Mustermann, 10A, got %v", result.VormerkungUser)
	}
	if result.VormerkungTitel != "Test Book" {
		t.Errorf("expected VormerkungTitel to be Test Book, got %v", result.VormerkungTitel)
	}
}
