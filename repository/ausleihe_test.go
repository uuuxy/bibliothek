package repository

import (
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

func TestNewAusleihAbfragen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewAusleihAbfragen(mock)
	if repo == nil {
		t.Fatal("NewAusleihAbfragen returned nil")
	}

	pgRepo, ok := repo.(*pgAusleihAbfragen)
	if !ok {
		t.Fatal("NewAusleihAbfragen did not return *pgAusleihAbfragen")
	}

	if pgRepo.db != mock {
		t.Error("NewAusleihAbfragen did not properly set the db field")
	}
}
