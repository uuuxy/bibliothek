package inventur

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLadeTitel(t *testing.T) {
	ctx := context.Background()
	abfrageMuster := `SELECT id, COALESCE\(isbn, ''\) AS isbn`

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		zuletztGezaehlt := "2023-01-01"
		mock.ExpectQuery(abfrageMuster).
			WithArgs("valid-id").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "isbn", "title", "author", "signatur", "cover_url",
				"subject", "track", "stock", "last_counted",
				"sort_order", "medientyp", "jahrgang_von", "jahrgang_bis",
				"erweiterte_eigenschaften", "auflage",
			}).AddRow(
				"valid-id", "1234567890", "Test Title", "Test Author", "SIG", "http://cover",
				"Math", "A", 10, &zuletztGezaehlt,
				1, "Buch", 5, 10,
				map[string]any{"key": "value"}, "4. Aufl. 2023",
			))

		eintrag, err := abfragen.LadeTitel(ctx, "valid-id")
		require.NoError(t, err)
		assert.NotNil(t, eintrag)
		assert.Equal(t, "valid-id", eintrag.ID)
		assert.Equal(t, "Test Title", eintrag.Titel)
		assert.Equal(t, map[string]any{"key": "value"}, eintrag.ErweiterteEigenschaften)
	})

	t.Run("not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(abfrageMuster).
			WithArgs("invalid-id").
			WillReturnError(pgx.ErrNoRows)

		eintrag, err := abfragen.LadeTitel(ctx, "invalid-id")
		assert.ErrorIs(t, err, ErrTitelNichtGefunden)
		assert.Nil(t, eintrag)
	})

	// Ein Fehler der Datenbank ist kein „nicht gefunden": Ein Fehler beim Scan (NULL in string)
	// endete sonst als 404.
	t.Run("db error ist kein not-found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		attrappenFehler := errors.New("db error")
		mock.ExpectQuery(abfrageMuster).
			WithArgs("error-id").
			WillReturnError(attrappenFehler)

		eintrag, err := abfragen.LadeTitel(ctx, "error-id")
		assert.ErrorIs(t, err, attrappenFehler)
		assert.NotErrorIs(t, err, ErrTitelNichtGefunden)
		assert.Nil(t, eintrag)
	})
}

func TestAendereTitelMetadaten(t *testing.T) {
	ctx := context.Background()
	abfrageMuster := `UPDATE buecher_titel`

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectExec(abfrageMuster).
			WithArgs("New Title", "New Author", "http://new-cover", "valid-id").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err = abfragen.AendereTitelMetadaten(ctx, "valid-id", "New Title", "New Author", "http://new-cover")
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectExec(abfrageMuster).
			WithArgs("New Title", "New Author", "http://new-cover", "invalid-id").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err = abfragen.AendereTitelMetadaten(ctx, "invalid-id", "New Title", "New Author", "http://new-cover")
		assert.ErrorIs(t, err, ErrTitelNichtGefunden)
	})

	t.Run("error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		attrappenFehler := errors.New("db error")
		mock.ExpectExec(abfrageMuster).
			WithArgs("New Title", "New Author", "http://new-cover", "error-id").
			WillReturnError(attrappenFehler)

		err = abfragen.AendereTitelMetadaten(ctx, "error-id", "New Title", "New Author", "http://new-cover")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metadaten konnten nicht aktualisiert werden: db error")
	})
}
