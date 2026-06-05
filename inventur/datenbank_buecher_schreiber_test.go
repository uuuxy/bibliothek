package inventur

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func TestCreateBook_Mock(t *testing.T) {
	ctx := context.Background()

	t.Run("Create valid book", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mock.Close()

		repo := NewBookRepository(mock)

		lastCounted := "2023-10-27"
		book := Book{
			ISBN:                    "978-3-16-148410-0",
			Title:                   "Test Book",
			Author:                  "Test Author",
			CoverURL:                "http://example.com/cover.jpg",
			Subject:                 "Math",
			GradeLevel:              10,
			Track:                   "A",
			Stock:                   5,
			LastCounted:             &lastCounted,
			Medientyp:               "Buch",
			ErweiterteEigenschaften: map[string]any{"color": "red"},
		}

		mock.ExpectQuery(`INSERT INTO buecher_titel`).
			WithArgs(
				"978-3-16-148410-0", // ISBN
				"Test Book",         // Title
				"Test Author",       // Author
				"http://example.com/cover.jpg", // CoverURL
				"Math",              // Subject
				int16(10),           // GradeLevel
				"A",                 // Track
				5,                   // Stock
				&lastCounted,        // LastCounted
				"Buch",              // Medientyp
				map[string]any{"color": "red"}, // ErweiterteEigenschaften
			).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("test-uuid-123"))

		id, err := repo.CreateBook(ctx, book)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if id != "test-uuid-123" {
			t.Errorf("expected id 'test-uuid-123', got '%s'", id)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Create duplicate ISBN", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mock.Close()

		repo := NewBookRepository(mock)

		book := Book{ISBN: "1234567890", Title: "Book 1"}

		pgErr := &pgconn.PgError{Code: "23505", ConstraintName: "books_isbn_key"}

		mock.ExpectQuery(`INSERT INTO buecher_titel`).
			WithArgs(
				"1234567890",
				"Book 1",
				"", // Author default from struct is empty string
				"", // CoverURL
				"", // Subject
				int16(0), // GradeLevel
				"", // Track
				0, // Stock
				(*string)(nil), // LastCounted
				"Buch", // Medientyp
				map[string]any{}, // ErweiterteEigenschaften
			).
			WillReturnError(pgErr)

		_, err = repo.CreateBook(ctx, book)
		if !errors.Is(err, ErrDuplicateISBN) {
			t.Fatalf("expected ErrDuplicateISBN, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Create generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mock.Close()

		repo := NewBookRepository(mock)

		book := Book{ISBN: "0987654321", Title: "Minimal Book"}

		mock.ExpectQuery(`INSERT INTO buecher_titel`).
			WithArgs(
				"0987654321",
				"Minimal Book",
				"", // Author
				"", // CoverURL
				"", // Subject
				int16(0), // GradeLevel
				"", // Track
				0, // Stock
				(*string)(nil), // LastCounted
				"Buch", // Medientyp
				map[string]any{}, // ErweiterteEigenschaften
			).
			WillReturnError(errors.New("some db error"))

		_, err = repo.CreateBook(ctx, book)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Create book checks correct defaults", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mock.Close()

		repo := NewBookRepository(mock)

		book := Book{
			ISBN:  "1111111111",
			Title: "Test Defaults",
		}

		mock.ExpectQuery(`INSERT INTO buecher_titel`).
			WithArgs(
				"1111111111",
				"Test Defaults",
				"",
				"",
				"",
				int16(0),
				"",
				0,
				(*string)(nil),
				"Buch", // The default medientyp that is assigned
				map[string]any{}, // The default properties assigned
			).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("test-uuid-456"))

		id, err := repo.CreateBook(ctx, book)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if id != "test-uuid-456" {
			t.Errorf("expected id 'test-uuid-456', got '%s'", id)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
