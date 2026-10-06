package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"
)

// mockBookRepoForScan implements a minimal BookRepository for testing ErkenneScan.
type mockBookRepoForScan struct {
	repository.BookRepository
	mockGetCopyByBarcode func(ctx context.Context, barcode string) (*repository.BookCopy, error)
}

func (m *mockBookRepoForScan) GetCopyByBarcode(ctx context.Context, barcode string) (*repository.BookCopy, error) {
	if m.mockGetCopyByBarcode != nil {
		return m.mockGetCopyByBarcode(ctx, barcode)
	}
	return nil, nil
}

// mockStudentRepoForScan implements a minimal StudentRepository for testing ErkenneScan.
type mockStudentRepoForScan struct {
	repository.StudentRepository
	mockGetLeserByBarcode func(ctx context.Context, barcode string) (*repository.Student, error)
}

func (m *mockStudentRepoForScan) GetLeserByBarcode(ctx context.Context, barcode string) (*repository.Student, error) {
	if m.mockGetLeserByBarcode != nil {
		return m.mockGetLeserByBarcode(ctx, barcode)
	}
	return nil, nil
}

func TestErkenneScan(t *testing.T) {
	errDbMock := errors.New("db error")

	tests := []struct {
		name                  string
		query                 string
		mockGetCopyByBarcode  func(ctx context.Context, barcode string) (*repository.BookCopy, error)
		mockGetLeserByBarcode func(ctx context.Context, barcode string) (*repository.Student, error)
		wantTyp               string
		wantID                string
		wantErr               error
	}{
		{
			name:  "Exact match for book copy",
			query: "B-12345",
			mockGetCopyByBarcode: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				if barcode == "B-12345" {
					return &repository.BookCopy{ID: "copy-1", BarcodeID: "B-12345", TitelID: "title-1"}, nil
				}
				return nil, nil
			},
			wantTyp: "exemplar",
			wantID:  "copy-1",
			wantErr: nil,
		},
		{
			name:  "Match for book copy via Littera decoded label",
			query: "5896800039556",
			mockGetCopyByBarcode: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				if barcode == "58968" {
					return &repository.BookCopy{ID: "copy-2", BarcodeID: "58968", TitelID: "title-2"}, nil
				}
				return nil, nil
			},
			wantTyp: "exemplar",
			wantID:  "copy-2",
			wantErr: nil,
		},
		{
			name:  "No match for book, match for student",
			query: "student-barcode-1",
			mockGetCopyByBarcode: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				return nil, nil
			},
			mockGetLeserByBarcode: func(ctx context.Context, barcode string) (*repository.Student, error) {
				if barcode == "student-barcode-1" {
					return &repository.Student{ID: "student-1", BarcodeID: "student-barcode-1"}, nil
				}
				return nil, nil
			},
			wantTyp: "schueler",
			wantID:  "student-1",
			wantErr: nil,
		},
		{
			name:  "No match for book, no match for student",
			query: "unknown-barcode",
			mockGetCopyByBarcode: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				return nil, nil
			},
			mockGetLeserByBarcode: func(ctx context.Context, barcode string) (*repository.Student, error) {
				return nil, nil
			},
			wantTyp: "",
			wantID:  "",
			wantErr: nil,
		},
		{
			name:  "Error fetching book copy",
			query: "B-error",
			mockGetCopyByBarcode: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				return nil, errDbMock
			},
			wantErr: errDbMock,
		},
		{
			name:  "Error fetching student",
			query: "student-error",
			mockGetCopyByBarcode: func(ctx context.Context, barcode string) (*repository.BookCopy, error) {
				return nil, nil
			},
			mockGetLeserByBarcode: func(ctx context.Context, barcode string) (*repository.Student, error) {
				return nil, errDbMock
			},
			wantErr: errDbMock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bookRepo := &mockBookRepoForScan{mockGetCopyByBarcode: tt.mockGetCopyByBarcode}
			studentRepo := &mockStudentRepoForScan{mockGetLeserByBarcode: tt.mockGetLeserByBarcode}

			treffer, err := ErkenneScan(context.Background(), bookRepo, studentRepo, tt.query)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr.Error())
				}
				if !errors.Is(err, tt.wantErr) {
					// We use fmt.Errorf("...: %w", err) in ErkenneScan, so errors.Is should work.
					t.Errorf("expected error to wrap %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("did not expect error, got: %v", err)
			}

			if tt.wantTyp == "" {
				if treffer != nil {
					t.Errorf("expected nil treffer, got %+v", treffer)
				}
			} else {
				if treffer == nil {
					t.Fatalf("expected treffer, got nil")
				}
				if treffer.Typ != tt.wantTyp {
					t.Errorf("expected Typ %q, got %q", tt.wantTyp, treffer.Typ)
				}
				if treffer.ID != tt.wantID {
					t.Errorf("expected ID %q, got %q", tt.wantID, treffer.ID)
				}
			}
		})
	}
}
