package repository

import (
	"github.com/pashagolub/pgxmock/v4"
	"testing"
)

func TestNewLoanRepository(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewLoanRepository(mock)
	if repo == nil {
		t.Fatal("NewLoanRepository returned nil")
	}

	pgRepo, ok := repo.(*pgLoanRepository)
	if !ok {
		t.Fatal("NewLoanRepository did not return *pgLoanRepository")
	}

	if pgRepo.db != mock {
		t.Error("NewLoanRepository did not properly set the db field")
	}
}
