package service

import (
	"context"
	"errors"
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

type searchBookRepo struct {
	repository.BookRepository
	titles []repository.BookTitle
	err    error
}

func (r *searchBookRepo) SearchTitles(_ context.Context, _ string) ([]repository.BookTitle, error) {
	return r.titles, r.err
}

func TestHandleSearchAction(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		mockTitles    []repository.BookTitle
		mockErr       error
		expectedType  string
		expectedError bool
	}{
		{
			name:          "Success - returns titles",
			query:         "hobbit",
			mockTitles:    []repository.BookTitle{{Titel: "The Hobbit"}},
			mockErr:       nil,
			expectedType:  "search_results",
			expectedError: false,
		},
		{
			name:          "Success - returns empty list",
			query:         "unknown",
			mockTitles:    []repository.BookTitle{},
			mockErr:       nil,
			expectedType:  "search_results",
			expectedError: false,
		},
		{
			name:          "Error - repository error",
			query:         "error",
			mockTitles:    nil,
			mockErr:       errors.New("db error"),
			expectedType:  "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &searchBookRepo{
				titles: tt.mockTitles,
				err:    tt.mockErr,
			}
			svc := &defaultOmniboxService{
				bookRepo: repo,
			}

			resp := &OmniboxResult{}
			err := svc.handleSearchAction(context.Background(), tt.query, resp)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected an error but got nil")
				}
				if err != nil && err.Error() != tt.mockErr.Error() {
					t.Errorf("Expected error %v, got %v", tt.mockErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if resp.Type != tt.expectedType {
					t.Errorf("Expected response type %q, got %q", tt.expectedType, resp.Type)
				}
				// Verify the exact content
				if len(resp.SearchResults) != len(tt.mockTitles) {
					t.Errorf("Expected %d search results, got %d", len(tt.mockTitles), len(resp.SearchResults))
				}
			}
		})
	}
}
