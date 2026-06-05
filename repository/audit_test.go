package repository

import (
	"testing"
)

func TestNewAuditRepository(t *testing.T) {
	// Constructor functions usually only require simple instantiation tests without complex mocks.
	// Passing nil is sufficient to test the assignment logic.
	repo := NewAuditRepository(nil)

	if repo == nil {
		t.Fatalf("NewAuditRepository returned nil")
	}

	pgRepo, ok := repo.(*pgAuditRepository)
	if !ok {
		t.Fatalf("NewAuditRepository did not return *pgAuditRepository")
	}

	if pgRepo.db != nil {
		t.Errorf("NewAuditRepository did not set db correctly. Expected nil, got %v", pgRepo.db)
	}
}
