package inventur

import (
	"context"
	"fmt"
	"testing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type MockRow struct {
	ScanFunc func(dest ...any) error
}

func (m *MockRow) Scan(dest ...any) error {
	if m.ScanFunc != nil {
		return m.ScanFunc(dest...)
	}
	return nil
}

type MockPool struct {
	QueryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (m *MockPool) Close() {}
func (m *MockPool) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *MockPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *MockPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, sql, args...)
	}
	return &MockRow{}
}
func (m *MockPool) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, nil
}
func (m *MockPool) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return nil, nil
}
func (m *MockPool) Ping(ctx context.Context) error {
	return nil
}

func TestGetBookByID(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockDB := &MockPool{
			QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
				return &MockRow{
					ScanFunc: func(dest ...any) error {
						if len(dest) != 13 {
							return fmt.Errorf("expected 13 destination arguments, got %d", len(dest))
						}
						// Simulate successful scan
						*dest[0].(*string) = "123"
						*dest[1].(*string) = "978-3-16-148410-0"
						*dest[2].(*string) = "Test Title"
						return nil
					},
				}
			},
		}

		repo := &BookRepository{db: mockDB}
		book, err := repo.GetBookByID(context.Background(), "123")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if book.ID != "123" {
			t.Errorf("expected ID 123, got %s", book.ID)
		}
		if book.Title != "Test Title" {
			t.Errorf("expected Title 'Test Title', got %s", book.Title)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		mockDB := &MockPool{
			QueryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
				return &MockRow{
					ScanFunc: func(dest ...any) error {
						return pgx.ErrNoRows
					},
				}
			},
		}

		repo := &BookRepository{db: mockDB}
		book, err := repo.GetBookByID(context.Background(), "123")

		if err == nil {
			t.Fatalf("expected an error, got nil")
		}

		if book != nil {
			t.Errorf("expected book to be nil, got %+v", book)
		}

		expectedErr := "buch nicht gefunden"
		if err.Error() != expectedErr {
			t.Errorf("expected error %q, got %q", expectedErr, err.Error())
		}
	})
}
