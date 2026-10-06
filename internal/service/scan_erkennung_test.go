package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"bibliothek/repository"
)

type scanErkennungMockBookRepo struct {
	repository.BookRepository
	mockGetCopyByBarcode func(ctx context.Context, barcode string) (*repository.BookCopy, error)
}

func (m *scanErkennungMockBookRepo) GetCopyByBarcode(ctx context.Context, barcode string) (*repository.BookCopy, error) {
	if m.mockGetCopyByBarcode != nil {
		return m.mockGetCopyByBarcode(ctx, barcode)
	}
	return nil, nil
}

type scanErkennungMockStudentRepo struct {
	repository.StudentRepository
	mockGetLeserByBarcode func(ctx context.Context, barcode string) (*repository.Student, error)
}

func (m *scanErkennungMockStudentRepo) GetLeserByBarcode(ctx context.Context, barcode string) (*repository.Student, error) {
	if m.mockGetLeserByBarcode != nil {
		return m.mockGetLeserByBarcode(ctx, barcode)
	}
	return nil, nil
}

func TestErkenneScan(t *testing.T) {
	ctx := context.Background()
	testErr := errors.New("test error")

	tests := []struct {
		name           string
		q              string
		mockBook       func(ctx context.Context, barcode string) (*repository.BookCopy, error)
		mockStudent    func(ctx context.Context, barcode string) (*repository.Student, error)
		expectedResult *ScanTreffer
		expectErr      bool
	}{
		{
			name: "Exemplar match",
			q:    "buch123",
			mockBook: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				if barcode == "buch123" {
					return &repository.BookCopy{ID: "copy-id", TitelID: "titel-id", BarcodeID: "buch123"}, nil
				}
				return nil, nil
			},
			mockStudent: func(ctx context.Context, barcode string) (*repository.Student, error) {
				return nil, nil // should not be called, but safe to provide
			},
			expectedResult: &ScanTreffer{Typ: "exemplar", ID: "copy-id", TitelID: "titel-id", Barcode: "buch123"},
			expectErr:      false,
		},
		{
			name: "Littera Exemplar match (fallback to decoded)",
			// 58968 -> 5896800039556
			q: "5896800039556",
			mockBook: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				// The first candidate will be "5896800039556"
				if barcode == "5896800039556" {
					return nil, nil
				}
				// The second candidate should be the decoded littera "58968"
				if barcode == "58968" {
					return &repository.BookCopy{ID: "littera-copy-id", TitelID: "littera-titel-id", BarcodeID: "58968"}, nil
				}
				return nil, nil
			},
			expectedResult: &ScanTreffer{Typ: "exemplar", ID: "littera-copy-id", TitelID: "littera-titel-id", Barcode: "58968"},
			expectErr:      false,
		},
		{
			name: "Exemplar repository error",
			q:    "bucherr",
			mockBook: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				return nil, testErr
			},
			expectedResult: nil,
			expectErr:      true,
		},
		{
			name: "Schueler match",
			q:    "schueler123",
			mockBook: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				return nil, nil // not a book
			},
			mockStudent: func(ctx context.Context, barcode string) (*repository.Student, error) {
				if barcode == "schueler123" {
					return &repository.Student{ID: "student-id", BarcodeID: "schueler123"}, nil
				}
				return nil, nil
			},
			expectedResult: &ScanTreffer{Typ: "schueler", ID: "student-id", Barcode: "schueler123"},
			expectErr:      false,
		},
		{
			name: "Schueler repository error",
			q:    "schuelererr",
			mockBook: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				return nil, nil
			},
			mockStudent: func(ctx context.Context, barcode string) (*repository.Student, error) {
				return nil, testErr
			},
			expectedResult: nil,
			expectErr:      true,
		},
		{
			name: "No match",
			q:    "unknown",
			mockBook: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				return nil, nil
			},
			mockStudent: func(ctx context.Context, barcode string) (*repository.Student, error) {
				return nil, nil
			},
			expectedResult: nil,
			expectErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bookRepo := &scanErkennungMockBookRepo{mockGetCopyByBarcode: tt.mockBook}
			studentRepo := &scanErkennungMockStudentRepo{mockGetLeserByBarcode: tt.mockStudent}

			result, err := ErkenneScan(ctx, bookRepo, studentRepo, tt.q)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if !reflect.DeepEqual(result, tt.expectedResult) {
				t.Errorf("expected result %+v, got %+v", tt.expectedResult, result)
			}
		})
	}
}
