package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/repository"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

type mockLoanRepoReturn struct {
	returnErr        error
	beginErr         error
	getActiveLoanErr error
	tx               pgx.Tx
}

func (m *mockLoanRepoReturn) GetActiveLoanByCopyID(ctx context.Context, copyID string) (*repository.Loan, error) {
	return nil, m.getActiveLoanErr
}
func (m *mockLoanRepoReturn) GetActiveLoanByCopyIDTx(ctx context.Context, tx pgx.Tx, copyID string) (*repository.Loan, error) {
	return nil, m.getActiveLoanErr
}
func (m *mockLoanRepoReturn) BeginTx(ctx context.Context) (pgx.Tx, error) { return m.tx, m.beginErr }
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
		// Donnerstag vor den Herbstferien (05.10.–17.10.2026): plus drei Tage ist Sonntag, der
		// nächste Schultag der Montag nach den Ferien.
		jetzt: func() time.Time { return time.Date(2026, time.October, 1, 10, 0, 0, 0, schoolLocation()) },
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

	// Die Abholfrist liest die Einstellungen (repository.Abholfrist); keine Sommerferien der
	// Schule, es gilt die Programmtabelle.
	mockPool.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}))

	// Status der Vormerkung auf 'abholbereit' setzen, bis Montag 19.10.2026 abends.
	mockPool.ExpectExec("UPDATE vormerkungen SET status = 'abholbereit'").
		WithArgs("c1", "v1", time.Date(2026, time.October, 19, 23, 59, 59, 0, schoolLocation())).
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

// TestHandleSimpleReturn_ReichtFehlerWeiter: Beginnt die Transaktion nicht oder lässt sich die
// offene Ausleihe nicht lesen, kommt der Fehler an. Verschluckt gälte das Buch als „nicht
// ausgeliehen", und die Theke wiese eine Rückgabe ab, die es gibt.
func TestHandleSimpleReturn_ReichtFehlerWeiter(t *testing.T) {
	dbFehler := errors.New("verbindung weg")
	exemplar := &repository.BookCopy{ID: "ex-1"}

	t.Run("die Transaktion beginnt nicht", func(t *testing.T) {
		svc := &defaultLoanService{loanRepo: &mockLoanRepoReturn{beginErr: dbFehler}}

		ergebnis, err := svc.HandleSimpleReturn(context.Background(), exemplar, "theke")
		if !errors.Is(err, dbFehler) {
			t.Errorf("Fehler %v, erwartet %v", err, dbFehler)
		}
		if ergebnis != nil {
			t.Errorf("Ergebnis %+v, erwartet keines", ergebnis)
		}
	})

	t.Run("die offene Ausleihe lässt sich nicht lesen", func(t *testing.T) {
		pool, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock: %v", err)
		}
		defer pool.Close()
		pool.ExpectBegin()
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatalf("Transaktion der Attrappe: %v", err)
		}
		pool.ExpectRollback()
		svc := &defaultLoanService{loanRepo: &mockLoanRepoReturn{tx: tx, getActiveLoanErr: dbFehler}}

		ergebnis, err := svc.HandleSimpleReturn(context.Background(), exemplar, "theke")
		if !errors.Is(err, dbFehler) {
			t.Errorf("Fehler %v, erwartet %v", err, dbFehler)
		}
		if ergebnis != nil {
			t.Errorf("Ergebnis %+v, erwartet keines", ergebnis)
		}
		if err := pool.ExpectationsWereMet(); err != nil {
			t.Errorf("die Transaktion wurde nicht zurückgerollt: %v", err)
		}
	})
}
