package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

type mockUserRepoForOmnibox struct {
	repository.UserRepository
	lehrer *repository.User
	err    error
}

func (m *mockUserRepoForOmnibox) GetLehrerByBarcode(ctx context.Context, barcode string) (*repository.User, error) {
	return m.lehrer, m.err
}

func TestHandleTeacherAction(t *testing.T) {
	tests := []struct {
		name        string
		lehrer      *repository.User
		err         error
		expectError bool
		expectType  string
	}{
		{
			name:        "Happy Path",
			lehrer:      &repository.User{ID: "teacher-123"},
			err:         nil,
			expectError: false,
			expectType:  "teacher",
		},
		{
			name:        "Not Found",
			lehrer:      nil,
			err:         nil,
			expectError: true,
			expectType:  "",
		},
		{
			name:        "Database Error",
			lehrer:      nil,
			err:         errors.New("db connection failed"),
			expectError: true,
			expectType:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockUserRepoForOmnibox{
				lehrer: tt.lehrer,
				err:    tt.err,
			}

			svc := &defaultOmniboxService{
				userRepo: mockRepo,
			}

			resp := &OmniboxResult{}
			err := svc.handleTeacherAction(context.Background(), "L-123", resp)

			if tt.expectError && err == nil {
				t.Fatalf("expected an error but got none")
			}

			if !tt.expectError && err != nil {
				t.Fatalf("did not expect an error, but got: %v", err)
			}

			if !tt.expectError && resp.Type != tt.expectType {
				t.Errorf("expected resp.Type to be %q, got %q", tt.expectType, resp.Type)
			}

			if !tt.expectError && resp.Teacher != tt.lehrer {
				t.Errorf("expected resp.Teacher to be %v, got %v", tt.lehrer, resp.Teacher)
			}
		})
	}
}

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
