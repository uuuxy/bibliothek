package inventur

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestUpdateBookCategory_Success(t *testing.T) {
	mockDB := &MockDB{
		ExecFunc: func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
			// Verify arguments
			if len(arguments) != 3 {
				t.Errorf("expected 3 arguments, got %d", len(arguments))
			}
			if arguments[0] != "Math" {
				t.Errorf("expected subject 'Math', got '%v'", arguments[0])
			}
			if arguments[1] != int16(10) {
				t.Errorf("expected gradeLevel 10, got '%v'", arguments[1])
			}
			if arguments[2] != "123e4567-e89b-12d3-a456-426614174000" {
				t.Errorf("expected id '123e4567-e89b-12d3-a456-426614174000', got '%v'", arguments[2])
			}

			// Return success with 1 row affected
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}

	repo := NewBookRepository(mockDB)
	err := repo.UpdateBookCategory(context.Background(), "123e4567-e89b-12d3-a456-426614174000", "Math", 10)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestUpdateBookCategory_NotFound(t *testing.T) {
	mockDB := &MockDB{
		ExecFunc: func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
			// Return success but 0 rows affected
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	}

	repo := NewBookRepository(mockDB)
	err := repo.UpdateBookCategory(context.Background(), "non-existent-id", "Math", 10)

	if err != ErrBookNotFound {
		t.Errorf("expected ErrBookNotFound, got %v", err)
	}
}

func TestUpdateBookCategory_DBError(t *testing.T) {
	mockErr := errors.New("database connection failed")
	mockDB := &MockDB{
		ExecFunc: func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, mockErr
		},
	}

	repo := NewBookRepository(mockDB)
	err := repo.UpdateBookCategory(context.Background(), "123e4567-e89b-12d3-a456-426614174000", "Math", 10)

	if err == nil {
		t.Error("expected an error, got nil")
	}
	if !errors.Is(err, mockErr) && err.Error() != "kategorie konnte nicht aktualisiert werden: database connection failed" {
		t.Errorf("expected error wrapping '%v', got '%v'", mockErr, err)
	}
}
