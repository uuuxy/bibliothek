package service

import (
	"context"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

type mockLoanRepo struct {
	repository.AusleihAbfragen // embed to satisfy interface methods we don't mock
	returnLoanTxErr            error
}

func (m *mockLoanRepo) BucheRueckgabe(ctx context.Context, tx pgx.Tx, loanID string, bearbeiterID string, isFremdrueckgabe bool) error {
	return m.returnLoanTxErr
}

func TestVerarbeiteFremdrueckgabe_Student(t *testing.T) {
	svc, _, mockPool := newValidationService(t, &repository.Leser{
		ID: "s1", Vorname: "Max", Nachname: "Mustermann",
	})
	defer mockPool.Close()

	mockLoan := &mockLoanRepo{}
	svc.ausleihAbfragen = mockLoan

	tx := beginTx(t, mockPool)

	activeLoan := &repository.Ausleihe{
		ID:         "loan1",
		SchuelerID: strPtr("s1"),
	}
	copy := &repository.Exemplar{
		ID:        "copy1",
		BarcodeID: "B-C1",
		Titel:     "Testbuch",
	}

	resp := &AusleihErgebnis{}

	// mock bedieneVormerkung internals
	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(pgx.ErrNoRows)
	mockPool.ExpectCommit()

	res, err := svc.verarbeiteFremdrueckgabe(context.Background(), tx, copy, activeLoan, "staff1", resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Art != "rueckgabe" {
		t.Errorf("expected resp.Type to be 'rueckgabe', got %s", res.Art)
	}
	if res.Vorbesitzer == nil || res.Vorbesitzer.ID != "s1" {
		t.Errorf("expected res.Vorbesitzer to be s1")
	}

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Auch das Buch eines Kollegen muss bei der Fremdrückgabe seinen Vorbesitzer NENNEN. Das
// ging vor Migration 125 über eine zweite Spalte und eine zweite Abfrage in die
// Kontentabelle; jetzt ist es derselbe Weg wie beim Schüler — Finde, nicht die
// Sicht `schueler`, sonst bliebe der Vorbesitzer namenlos.
func TestVerarbeiteFremdrueckgabe_Kollege(t *testing.T) {
	svc, _, mockPool := newValidationService(t, &repository.Leser{
		ID: "t1", Vorname: "Hans", Nachname: "Lehrer", Art: "lehrkraft",
	})
	defer mockPool.Close()

	mockLoan := &mockLoanRepo{}
	svc.ausleihAbfragen = mockLoan

	tx := beginTx(t, mockPool)

	activeLoan := &repository.Ausleihe{
		ID:         "loan2",
		SchuelerID: strPtr("t1"),
	}
	copy := &repository.Exemplar{
		ID:        "copy2",
		BarcodeID: "B-C2",
		Titel:     "Lehrerbuch",
	}

	resp := &AusleihErgebnis{}

	// mock bedieneVormerkung internals
	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(pgx.ErrNoRows)
	mockPool.ExpectCommit()

	res, err := svc.verarbeiteFremdrueckgabe(context.Background(), tx, copy, activeLoan, "staff1", resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Art != "rueckgabe" {
		t.Errorf("expected resp.Type to be 'rueckgabe', got %s", res.Art)
	}
	if res.Vorbesitzer == nil || res.Vorbesitzer.ID != "t1" {
		t.Errorf("der Kollege muss als Vorbesitzer in der Antwort stehen, bekam: %+v", res.Vorbesitzer)
	}
	if res.Vorbesitzer.Art != "lehrkraft" {
		t.Errorf("die Art des Vorbesitzers fehlt: %+v", res.Vorbesitzer)
	}

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestVerarbeiteFremdrueckgabe_ReturnLoanError(t *testing.T) {
	svc, _, mockPool := newValidationService(t, &repository.Leser{
		ID: "s1", Vorname: "Max", Nachname: "Mustermann",
	})
	defer mockPool.Close()

	// Return error
	mockLoan := &mockLoanRepo{returnLoanTxErr: pgx.ErrTxClosed}
	svc.ausleihAbfragen = mockLoan

	tx := beginTx(t, mockPool)

	activeLoan := &repository.Ausleihe{
		ID:         "loan3",
		SchuelerID: strPtr("s1"),
	}
	copy := &repository.Exemplar{
		ID: "copy3",
	}

	resp := &AusleihErgebnis{}

	res, err := svc.verarbeiteFremdrueckgabe(context.Background(), tx, copy, activeLoan, "staff1", resp)

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if res != nil {
		t.Errorf("expected res to be nil, got %v", res)
	}
}

func TestVerarbeiteFremdrueckgabe_CommitError(t *testing.T) {
	svc, _, mockPool := newValidationService(t, &repository.Leser{
		ID: "s1", Vorname: "Max", Nachname: "Mustermann",
	})
	defer mockPool.Close()

	mockLoan := &mockLoanRepo{}
	svc.ausleihAbfragen = mockLoan

	tx := beginTx(t, mockPool)

	activeLoan := &repository.Ausleihe{
		ID:         "loan4",
		SchuelerID: strPtr("s1"),
	}
	copy := &repository.Exemplar{
		ID:        "copy4",
		BarcodeID: "B-C4",
		Titel:     "Testbuch",
	}

	resp := &AusleihErgebnis{}

	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(pgx.ErrNoRows)
	mockPool.ExpectCommit().WillReturnError(pgx.ErrTxClosed)

	res, err := svc.verarbeiteFremdrueckgabe(context.Background(), tx, copy, activeLoan, "staff1", resp)

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
