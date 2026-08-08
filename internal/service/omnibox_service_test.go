package service

import (
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

func TestNewOmniboxService(t *testing.T) {
	// Setup zero-value dependencies (we just need to verify they get injected correctly)
	var pool db.PgxPoolIface
	var studentRepo repository.StudentRepository
	var bookRepo repository.BookRepository
	var userRepo repository.UserRepository
	var loanRepo repository.LoanRepository
	var loanSvc LoanService
	var deviceSvc DeviceService

	// Act
	svc := NewOmniboxService(
		pool,
		studentRepo,
		bookRepo,
		userRepo,
		loanRepo,
		loanSvc,
		deviceSvc,
	)

	// Assert
	if svc == nil {
		t.Fatalf("Expected NewOmniboxService to return a non-nil instance")
	}

	impl, ok := svc.(*defaultOmniboxService)
	if !ok {
		t.Fatalf("Expected implementation to be of type *defaultOmniboxService, got %T", svc)
	}

	// Verify all dependencies were injected
	if impl.pool != pool {
		t.Errorf("Expected pool to be injected")
	}
	if impl.studentRepo != studentRepo {
		t.Errorf("Expected studentRepo to be injected")
	}
	if impl.bookRepo != bookRepo {
		t.Errorf("Expected bookRepo to be injected")
	}
	if impl.userRepo != userRepo {
		t.Errorf("Expected userRepo to be injected")
	}
	if impl.loanRepo != loanRepo {
		t.Errorf("Expected loanRepo to be injected")
	}
	if impl.loanSvc != loanSvc {
		t.Errorf("Expected loanSvc to be injected")
	}
	if impl.deviceSvc != deviceSvc {
		t.Errorf("Expected deviceSvc to be injected")
	}
}
