package inventur

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestListExternalCoverBooks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	// Use a regex pattern to match the query, escaping parentheses and dollar sign.
	queryPattern := "SELECT id, COALESCE\\(isbn, ''\\) AS isbn, titel AS title, COALESCE\\(cover_url, ''\\) AS cover_url"

	t.Run("Happy path with valid limit", func(t *testing.T) {
		mock.ExpectQuery(queryPattern).
			WithArgs(50).
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "cover_url"}).
				AddRow("1", "123", "Title 1", "http://cover1").
				AddRow("2", "456", "Title 2", "https://cover2"))

		books, err := repo.ListExternalCoverBooks(ctx, 50)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(books) != 2 {
			t.Errorf("expected 2 books, got %d", len(books))
		}
		if books[0].ID != "1" || books[1].ID != "2" {
			t.Errorf("unexpected books returned: %+v", books)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Default limit mapping", func(t *testing.T) {
		mock.ExpectQuery(queryPattern).
			WithArgs(100). // 0 maps to 100
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "cover_url"}).
				AddRow("3", "789", "Title 3", "http://cover3"))

		books, err := repo.ListExternalCoverBooks(ctx, 0)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(books) != 1 {
			t.Errorf("expected 1 book, got %d", len(books))
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Database query error", func(t *testing.T) {
		mock.ExpectQuery(queryPattern).
			WithArgs(100).
			WillReturnError(errors.New("db error"))

		books, err := repo.ListExternalCoverBooks(ctx, -5) // -5 maps to 100
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if books != nil {
			t.Errorf("expected nil books, got %+v", books)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Row scan error", func(t *testing.T) {
		// Return a row with incompatible type or missing column
		mock.ExpectQuery(queryPattern).
			WithArgs(100).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).
				AddRow("1")) // Missing columns will cause scan error

		books, err := repo.ListExternalCoverBooks(ctx, 100)
		if err == nil {
			t.Errorf("expected scan error, got nil")
		}
		if books != nil {
			t.Errorf("expected nil books, got %+v", books)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Rows iteration error", func(t *testing.T) {
		mock.ExpectQuery(queryPattern).
			WithArgs(100).
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "cover_url"}).
				AddRow("1", "123", "Title 1", "http://cover1").
				RowError(0, errors.New("iteration error")))

		books, err := repo.ListExternalCoverBooks(ctx, 100)
		if err == nil {
			t.Errorf("expected iteration error, got nil")
		}
		if books != nil {
			t.Errorf("expected nil books, got %+v", books)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
