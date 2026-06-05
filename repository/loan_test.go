package repository

import (
	"bibliothek/db"
	"testing"
)

// MockPgxPool is a minimal mock of db.PgxPoolIface for testing constructor and setup functions.
type MockPgxPool struct {
	db.PgxPoolIface
}

func TestNewLoanRepository(t *testing.T) {
	mockDB := &MockPgxPool{}
	repo := NewLoanRepository(mockDB)

	if repo == nil {
		t.Fatal("Expected NewLoanRepository to return a non-nil repository")
	}

	pgRepo, ok := repo.(*pgLoanRepository)
	if !ok {
		t.Fatalf("Expected NewLoanRepository to return type *pgLoanRepository, got %T", repo)
	}

	if pgRepo.db != mockDB {
		t.Errorf("Expected repository to store the provided db interface, got %v", pgRepo.db)
	}
}
