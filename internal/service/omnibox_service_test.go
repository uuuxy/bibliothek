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

// mockBookRepo is a partial mock for repository.BookRepository
type mockBookRepo struct {
	repository.BookRepository
	mockGetCopyByBarcode func(ctx context.Context, barcode string) (*repository.BookCopy, error)
}

func (m *mockBookRepo) GetCopyByBarcode(ctx context.Context, barcode string) (*repository.BookCopy, error) {
	if m.mockGetCopyByBarcode != nil {
		return m.mockGetCopyByBarcode(ctx, barcode)
	}
	return nil, nil
}

// mockLoanService is a partial mock for LoanService
type mockLoanService struct {
	LoanService
	mockHandleUnifiedCheckout func(ctx context.Context, copy *repository.BookCopy, activeStudentID *string, activeTeacherID *string, staffID string, overrideBlock bool) (*LoanResult, error)
	mockHandleSimpleReturn    func(ctx context.Context, copy *repository.BookCopy, staffID string, staffRole string) (*LoanResult, error)
}

func (m *mockLoanService) HandleUnifiedCheckout(ctx context.Context, copy *repository.BookCopy, activeStudentID *string, activeTeacherID *string, staffID string, overrideBlock bool) (*LoanResult, error) {
	if m.mockHandleUnifiedCheckout != nil {
		return m.mockHandleUnifiedCheckout(ctx, copy, activeStudentID, activeTeacherID, staffID, overrideBlock)
	}
	return nil, nil
}

func (m *mockLoanService) HandleSimpleReturn(ctx context.Context, copy *repository.BookCopy, staffID string, staffRole string) (*LoanResult, error) {
	if m.mockHandleSimpleReturn != nil {
		return m.mockHandleSimpleReturn(ctx, copy, staffID, staffRole)
	}
	return nil, nil
}

func TestHandleBookAction(t *testing.T) {
	ctx := context.Background()

	t.Run("GetCopyByBarcode returns error", func(t *testing.T) {
		expectedErr := errors.New("db error")
		bookRepo := &mockBookRepo{
			mockGetCopyByBarcode: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				return nil, expectedErr
			},
		}

		svc := &defaultOmniboxService{
			bookRepo: bookRepo,
		}

		q := OmniboxQuery{Query: "123"}
		resp := &OmniboxResult{}
		err := svc.handleBookAction(ctx, q, resp)

		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("GetCopyByBarcode returns nil (not found)", func(t *testing.T) {
		bookRepo := &mockBookRepo{
			mockGetCopyByBarcode: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				return nil, nil
			},
		}

		svc := &defaultOmniboxService{
			bookRepo: bookRepo,
		}

		q := OmniboxQuery{Query: "123"}
		resp := &OmniboxResult{}
		err := svc.handleBookAction(ctx, q, resp)

		if err == nil {
			t.Errorf("Expected an error when copy is not found, got nil")
		} else if !errors.Is(err, ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})

	t.Run("Successful Checkout with active student", func(t *testing.T) {
		studentID := "student-123"
		bookCopy := &repository.BookCopy{ID: "copy-123", IstAusleihbar: true, IstAusgesondert: false}
		expectedLoanResult := &LoanResult{Type: "ausleihe"}

		bookRepo := &mockBookRepo{
			mockGetCopyByBarcode: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				return bookCopy, nil
			},
		}

		loanSvc := &mockLoanService{
			mockHandleUnifiedCheckout: func(ctx context.Context, copy *repository.BookCopy, activeStudentID *string, activeTeacherID *string, staffID string, overrideBlock bool) (*LoanResult, error) {
				if copy != bookCopy || activeStudentID == nil || *activeStudentID != studentID {
					t.Errorf("HandleUnifiedCheckout called with wrong arguments")
				}
				return expectedLoanResult, nil
			},
		}

		svc := &defaultOmniboxService{
			bookRepo: bookRepo,
			loanSvc:  loanSvc,
		}

		q := OmniboxQuery{Query: "123", ActiveStudentID: &studentID}
		resp := &OmniboxResult{}
		err := svc.handleBookAction(ctx, q, resp)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if resp.Type != "ausleihe" {
			t.Errorf("Expected resp.Type to be 'ausleihe', got %s", resp.Type)
		}
	})

	t.Run("Successful Return with no active borrower", func(t *testing.T) {
		bookCopy := &repository.BookCopy{ID: "copy-123", IstAusleihbar: true, IstAusgesondert: false}
		expectedLoanResult := &LoanResult{Type: "rueckgabe"}

		bookRepo := &mockBookRepo{
			mockGetCopyByBarcode: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				return bookCopy, nil
			},
		}

		loanSvc := &mockLoanService{
			mockHandleSimpleReturn: func(ctx context.Context, copy *repository.BookCopy, staffID string, staffRole string) (*LoanResult, error) {
				if copy != bookCopy {
					t.Errorf("HandleSimpleReturn called with wrong arguments")
				}
				return expectedLoanResult, nil
			},
		}

		svc := &defaultOmniboxService{
			bookRepo: bookRepo,
			loanSvc:  loanSvc,
		}

		q := OmniboxQuery{Query: "123"}
		resp := &OmniboxResult{}
		err := svc.handleBookAction(ctx, q, resp)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if resp.Type != "rueckgabe" {
			t.Errorf("Expected resp.Type to be 'rueckgabe', got %s", resp.Type)
		}
	})
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
