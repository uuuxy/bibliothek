package inventur

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBookByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	queryRegex := `SELECT id, COALESCE\(isbn, ''\) AS isbn`

	t.Run("success", func(t *testing.T) {
		lastCounted := "2023-01-01"
		mock.ExpectQuery(queryRegex).
			WithArgs("valid-id").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "isbn", "title", "author", "signatur", "cover_url",
				"subject", "grade_level", "track", "stock", "last_counted",
				"sort_order", "medientyp", "jahrgang_von", "jahrgang_bis",
				"erweiterte_eigenschaften",
			}).AddRow(
				"valid-id", "1234567890", "Test Title", "Test Author", "SIG", "http://cover",
				"Math", int16(5), "A", 10, &lastCounted,
				1, "Buch", 5, 10,
				map[string]any{"key": "value"},
			))

		book, err := repo.GetBookByID(ctx, "valid-id")
		require.NoError(t, err)
		assert.NotNil(t, book)
		assert.Equal(t, "valid-id", book.ID)
		assert.Equal(t, "Test Title", book.Title)
		assert.Equal(t, map[string]any{"key": "value"}, book.ErweiterteEigenschaften)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(queryRegex).
			WithArgs("invalid-id").
			WillReturnError(pgx.ErrNoRows)

		book, err := repo.GetBookByID(ctx, "invalid-id")
		assert.Error(t, err)
		assert.Nil(t, book)
		assert.Equal(t, "buch nicht gefunden", err.Error())
	})
}

func TestUpdateBookMetadata(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	queryRegex := `UPDATE buecher_titel`

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec(queryRegex).
			WithArgs("New Title", "New Author", "http://new-cover", "valid-id").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.UpdateBookMetadata(ctx, "valid-id", "New Title", "New Author", "http://new-cover")
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec(queryRegex).
			WithArgs("New Title", "New Author", "http://new-cover", "invalid-id").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.UpdateBookMetadata(ctx, "invalid-id", "New Title", "New Author", "http://new-cover")
		assert.ErrorIs(t, err, ErrBookNotFound)
	})

	t.Run("error", func(t *testing.T) {
		mockErr := errors.New("db error")
		mock.ExpectExec(queryRegex).
			WithArgs("New Title", "New Author", "http://new-cover", "error-id").
			WillReturnError(mockErr)

		err := repo.UpdateBookMetadata(ctx, "error-id", "New Title", "New Author", "http://new-cover")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metadaten konnten nicht aktualisiert werden: db error")
	})
}

func TestUpdateBookCategory(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	queryRegex := `UPDATE buecher_titel`

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec(queryRegex).
			WithArgs("Science", int16(6), "valid-id").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.UpdateBookCategory(ctx, "valid-id", "Science", int16(6))
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec(queryRegex).
			WithArgs("Science", int16(6), "invalid-id").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.UpdateBookCategory(ctx, "invalid-id", "Science", int16(6))
		assert.ErrorIs(t, err, ErrBookNotFound)
	})

	t.Run("error", func(t *testing.T) {
		mockErr := errors.New("db error")
		mock.ExpectExec(queryRegex).
			WithArgs("Science", int16(6), "error-id").
			WillReturnError(mockErr)

		err := repo.UpdateBookCategory(ctx, "error-id", "Science", int16(6))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "kategorie konnte nicht aktualisiert werden: db error")
	})
}
