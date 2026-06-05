package inventur

import (
	"context"
	"testing"
	"errors"
	"bibliothek/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type MockPool struct {
	db.PgxPoolIface
	ExecFunc     func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (m *MockPool) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, sql, arguments...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *MockPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, sql, args...)
	}
	return nil
}

type MockRow struct {
	ScanFunc func(dest ...any) error
}

func (m *MockRow) Scan(dest ...any) error {
	if m.ScanFunc != nil {
		return m.ScanFunc(dest...)
	}
	return nil
}

func TestUpsertBook_Success(t *testing.T) {
	var queryArgs []any
	mockPool := &MockPool{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			queryArgs = args
			return &MockRow{
				ScanFunc: func(dest ...any) error {
					if id, ok := dest[0].(*string); ok {
						*id = "test-id"
					}
					return nil
				},
			}
		},
	}

	repo := NewBookRepository(mockPool)

	book := Book{
		ISBN:       "978-3-16-148410-0",
		Title:      "Test Title",
		Author:     "Test Author",
		CoverURL:   "http://example.com/cover.jpg",
		Subject:    "Test Subject",
		GradeLevel: 5,
		Track:      "Test Track",
		Stock:      10,
	}

	id, err := repo.UpsertBook(context.Background(), book)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if id != "test-id" {
		t.Fatalf("expected ID to be 'test-id', got: %s", id)
	}

	if len(queryArgs) != 10 {
		t.Fatalf("expected 10 arguments, got %d", len(queryArgs))
	}

	if queryArgs[0] != book.ISBN || queryArgs[1] != book.Title || queryArgs[2] != book.Author || queryArgs[3] != book.CoverURL || queryArgs[4] != book.Subject || queryArgs[5] != book.GradeLevel || queryArgs[6] != book.Track || queryArgs[7] != book.Stock || queryArgs[8] != book.LastCounted {
		t.Fatalf("unexpected query args: %v", queryArgs)
	}

	if queryArgs[9] != "Buch" {
		t.Fatalf("expected medientyp to default to 'Buch', got: %v", queryArgs[9])
	}
}

func TestUpsertBook_WithMedientyp(t *testing.T) {
	var queryArgs []any
	mockPool := &MockPool{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			queryArgs = args
			return &MockRow{
				ScanFunc: func(dest ...any) error {
					if id, ok := dest[0].(*string); ok {
						*id = "test-id"
					}
					return nil
				},
			}
		},
	}

	repo := NewBookRepository(mockPool)

	book := Book{
		ISBN:      "978-3-16-148410-0",
		Medientyp: "Zeitschrift",
	}

	id, err := repo.UpsertBook(context.Background(), book)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if id != "test-id" {
		t.Fatalf("expected ID to be 'test-id', got: %s", id)
	}

	if queryArgs[9] != "Zeitschrift" {
		t.Fatalf("expected medientyp to be 'Zeitschrift', got: %v", queryArgs[9])
	}
}

func TestUpsertBook_DBError(t *testing.T) {
	mockPool := &MockPool{
		QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &MockRow{
				ScanFunc: func(dest ...any) error {
					return errors.New("db connection failed")
				},
			}
		},
	}

	repo := NewBookRepository(mockPool)

	book := Book{
		ISBN: "978-3-16-148410-0",
	}

	_, err := repo.UpsertBook(context.Background(), book)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedErr := "buch konnte nicht importiert werden: db connection failed"
	if err.Error() != expectedErr {
		t.Fatalf("expected error message '%s', got: '%v'", expectedErr, err)
	}
}
