package inventur

import (
	"context"
	"fmt"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBookRepository_ListBooks(t *testing.T) {
	ctx := context.Background()
	grade5 := int16(5)
	lastCounted := "2023-01-01"

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		mock.ExpectQuery(`SELECT.+FROM buecher_titel bt LEFT JOIN buecher_exemplare e.+`).
			WithArgs("Math", &grade5, "algebra", 50000).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track",
				"verfuegbar", "gesamt", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis",
				"untertitel", "verlag", "erscheinungsjahr", "beschreibung", "erweiterte_eigenschaften",
			}).AddRow(
				"book-1", "123", "Algebra", "Smith", "SIG-1", "url", "Math", int16(5), "A",
				2, 3, &lastCounted, 1, "Buch", 5, 6,
				"", "", 2020, "", map[string]any{},
			))

		books, err := repo.ListBooks(ctx, "Math", &grade5, "algebra")
		assert.NoError(t, err)
		assert.Len(t, books, 1)
		if len(books) > 0 {
			assert.Equal(t, "book-1", books[0].ID)
			assert.Equal(t, 3, books[0].Stock)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		mock.ExpectQuery(`SELECT.+FROM buecher_titel bt LEFT JOIN buecher_exemplare e.+`).
			WithArgs("", (*int16)(nil), "", 50000).
			WillReturnError(fmt.Errorf("db connection failed"))

		books, err := repo.ListBooks(ctx, "", nil, "")
		assert.ErrorContains(t, err, "bücher konnten nicht geladen werden")
		assert.Nil(t, books)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestBookRepository_ListExternalCoverBooks(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url FROM buecher_titel WHERE cover_url LIKE 'http%' ORDER BY id ASC LIMIT \$1`).
			WithArgs(10).
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "cover_url"}).
				AddRow("book-1", "123", "Title 1", "http://example.com/1.jpg"))

		books, err := repo.ListExternalCoverBooks(ctx, 10)
		assert.NoError(t, err)
		assert.Len(t, books, 1)
		if len(books) > 0 {
			assert.Equal(t, "book-1", books[0].ID)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with default limit", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url FROM buecher_titel WHERE cover_url LIKE 'http%' ORDER BY id ASC LIMIT \$1`).
			WithArgs(100).
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "cover_url"}).
				AddRow("book-1", "123", "Title 1", "http://example.com/1.jpg"))

		books, err := repo.ListExternalCoverBooks(ctx, 0)
		assert.NoError(t, err)
		assert.Len(t, books, 1)
		if len(books) > 0 {
			assert.Equal(t, "book-1", books[0].ID)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url FROM buecher_titel WHERE cover_url LIKE 'http%' ORDER BY id ASC LIMIT \$1`).
			WithArgs(100).
			WillReturnError(fmt.Errorf("db failure"))

		books, err := repo.ListExternalCoverBooks(ctx, -1)
		assert.ErrorContains(t, err, "bücher mit externen covern konnten nicht geladen werden")
		assert.Nil(t, books)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestBookRepository_ListBooksByIDs(t *testing.T) {
	ctx := context.Background()
	lastCounted := "2023-01-01"

	t.Run("empty ids", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		books, err := repo.ListBooksByIDs(ctx, []string{})
		assert.NoError(t, err)
		assert.Empty(t, books)
		// No query expected
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		ids := []string{"book-1"}
		mock.ExpectQuery(`SELECT.+FROM buecher_titel bt LEFT JOIN buecher_exemplare e.+`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "grade_level", "track",
				"verfuegbar", "gesamt", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis",
				"untertitel", "verlag", "erscheinungsjahr", "beschreibung", "erweiterte_eigenschaften",
			}).AddRow(
				"book-1", "123", "Algebra", "Smith", "SIG-1", "url", "Math", int16(5), "A",
				2, 3, &lastCounted, 1, "Buch", 5, 6,
				"", "", 2020, "desc", map[string]any{},
			))

		books, err := repo.ListBooksByIDs(ctx, ids)
		assert.NoError(t, err)
		assert.Len(t, books, 1)
		if len(books) > 0 {
			assert.Equal(t, "book-1", books[0].ID)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		ids := []string{"book-1"}
		mock.ExpectQuery(`SELECT.+FROM buecher_titel bt LEFT JOIN buecher_exemplare e.+`).
			WithArgs(ids).
			WillReturnError(fmt.Errorf("db connection failed"))

		books, err := repo.ListBooksByIDs(ctx, ids)
		assert.ErrorContains(t, err, "bücher nach ids konnten nicht geladen werden")
		assert.Nil(t, books)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
