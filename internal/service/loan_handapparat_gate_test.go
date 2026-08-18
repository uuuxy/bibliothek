package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// gateLoanRepo stellt genau die drei Methoden, die HandleSimpleReturn bis zur
// Weichenstellung braucht — alles danach beantwortet der Handapparat selbst.
type gateLoanRepo struct {
	repository.LoanRepository
	mock pgxmock.PgxPoolIface
}

func (r *gateLoanRepo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.mock.Begin(ctx)
}

func (r *gateLoanRepo) GetActiveLoanByCopyIDTx(_ context.Context, _ pgx.Tx, _ string) (*repository.Loan, error) {
	return nil, nil // Buch ist frei — genau der Fall, in dem die Weiche entscheidet
}

func (r *gateLoanRepo) CreateUserLoanTx(_ context.Context, _ pgx.Tx, _ string, _ string, _ string, _ time.Time, _ bool) (*repository.Loan, error) {
	return &repository.Loan{}, nil
}

// TestHandapparatGate_RollennameNachMigration069: Die Weiche zum Handapparat
// verglich wortwörtlich mit "LEHRER" — Migration 069 benannte die Rolle in
// KOLLEGIUM um, der Zweig war seither UNERREICHBAR (Lehrkraft scannt freies
// Buch → "nicht ausgeliehen" statt Jahres-Ausleihe). Die bestehenden Tests
// riefen handleLehrerHandapparat DIREKT auf und sahen das tote Tor nie —
// dieser Test geht durch die Vordertür HandleSimpleReturn.
//
// Beweisidee ohne tiefes Mock-Geflecht: Das Exemplar ist nicht ausleihbar.
// Erreicht der Aufruf den Handapparat, lautet der Fehler "nicht ausleihbar";
// bleibt das Tor zu, lautet er "nicht ausgeliehen". Die Botschaft verrät den Weg.
func TestHandapparatGate_RollennameNachMigration069(t *testing.T) {
	for _, rolle := range []string{"KOLLEGIUM", "kollegium", "Kollegium"} {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := &defaultLoanService{loanRepo: &gateLoanRepo{mock: mock}}
		copy := &repository.BookCopy{ID: "c1", TitelID: "t1", IstAusleihbar: false}

		_, err = svc.HandleSimpleReturn(context.Background(), copy, "staff1", rolle)
		if err == nil || !strings.Contains(err.Error(), "nicht ausleihbar") {
			t.Errorf("Rolle %q: Handapparat-Weiche blieb zu (err=%v) — der Scan einer Lehrkraft erreicht den Handapparat nicht", rolle, err)
		}
		mock.Close()
	}

	// Gegenprobe: Ein Mitarbeiter darf die Weiche NICHT nehmen.
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectBegin()
	mock.ExpectRollback()
	svc := &defaultLoanService{loanRepo: &gateLoanRepo{mock: mock}}
	copy := &repository.BookCopy{ID: "c1", TitelID: "t1", IstAusleihbar: false}
	_, err = svc.HandleSimpleReturn(context.Background(), copy, "staff1", "MITARBEITER")
	if err == nil || !strings.Contains(err.Error(), "nicht ausgeliehen") {
		t.Errorf("Mitarbeiter-Scan eines freien Buchs muss 'nicht ausgeliehen' melden, got %v", err)
	}
}
