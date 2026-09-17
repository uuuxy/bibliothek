package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

// Wer ein FREIES Buch scannt, ohne dass ein Ausweis vorliegt, bekommt eine Meldung — keine
// Ausleihe auf den eigenen Namen.
//
// Bis zum 16.09.2026 gab es hier einen Sonderweg: Scannte ein angemeldetes
// Kollegiumskonto ein freies Buch, buchte das System es SOFORT auf diese Person, ein Jahr
// Frist. Wer Rückläufer sortiert, sammelte damit still Bücher auf seinem eigenen Namen —
// und merkte es erst, wenn die Mahnung kam oder die Inventur das Buch vermisste. Die
// Vorgeschichte zeigt, wie unauffällig so ein Zweig ist: Er verglich die Rolle wortwörtlich
// mit „LEHRER", Migration 069 benannte sie in „KOLLEGIUM" um, und der Zweig war fast vier
// Monate lang tot, ohne dass es jemandem auffiel.
//
// Ausgeliehen wird über den Ausweis. Dieser Test hält die Tür zu.
type selbstausleiheLoanRepo struct {
	repository.LoanRepository
	mock      pgxmock.PgxPoolIface
	geschrieb bool
}

func (r *selbstausleiheLoanRepo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.mock.Begin(ctx)
}

func (r *selbstausleiheLoanRepo) GetActiveLoanByCopyIDTx(_ context.Context, _ pgx.Tx, _ string) (*repository.Loan, error) {
	return nil, nil // Buch ist frei — genau der Fall, in dem der Sonderweg zuschlug
}

func (r *selbstausleiheLoanRepo) CreateLoanTx(_ context.Context, _ pgx.Tx, _, _, _ string, _ time.Time, _ bool) (*repository.Loan, error) {
	r.geschrieb = true
	return &repository.Loan{}, nil
}

func TestFreiesBuchWirdNichtAufDenScannerGebucht(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectBegin()
	mock.ExpectRollback()

	repo := &selbstausleiheLoanRepo{mock: mock}
	svc := &defaultLoanService{loanRepo: repo}
	copy := &repository.BookCopy{ID: "c1", TitelID: "t1", IstAusleihbar: true}

	_, err = svc.HandleSimpleReturn(context.Background(), copy, "staff1")

	if !errors.Is(err, ErrInvalidState) || !strings.Contains(err.Error(), "nicht ausgeliehen") {
		t.Errorf("ein freies Buch ohne Ausweis muss „nicht ausgeliehen\" melden, bekam: %v", err)
	}
	if repo.geschrieb {
		t.Error("es wurde eine Ausleihe geschrieben — der Scanner hat das Buch auf sich selbst gebucht")
	}
}
