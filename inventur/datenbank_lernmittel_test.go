package inventur

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBookRepository_GetLernmittelTitel(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &BookRepository{db: mock}
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		mock.ExpectQuery("WITH titel AS").
			WithArgs(0, "", "", true, "Mathematik").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "titel", "autor", "subject", "cover_url", "isbn",
				"jahrgang_von", "jahrgang_bis", "track", "gezaehlt",
				"gesamt", "verliehen", "verfuegbar", "auflagen",
			}).
				AddRow("uuid-1", "Mathe 1", "Autor 1", "Mathematik", "url1", "123", 5, 6, "Gymnasium", "01.01.2023", 10, 2, 8, nil).
				AddRow("uuid-2", "Mathe 2", "Autor 2", "Mathematik", "url2", "456", 7, 8, "Realschule", "02.01.2023", 5, 5, 0, []byte(`[{"id":"uuid-2","auflage":"1.","erscheinungsjahr":2020,"gesamt_bestand":5}]`)))

		filter := LernmittelFilter{}
		result, err := repo.GetLernmittelTitel(ctx, "Mathematik", true, filter)

		assert.NoError(t, err)
		assert.Len(t, result, 2)

		assert.Equal(t, "Mathe 1", result[0].Title)
		assert.Nil(t, result[0].AuflagenBestand)

		assert.Equal(t, "Mathe 2", result[1].Title)
		assert.NotNil(t, result[1].AuflagenBestand)
		assert.Len(t, result[1].AuflagenBestand, 1)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		mock.ExpectQuery("WITH titel AS").
			WithArgs(0, "", "", false, "Biologie").
			WillReturnError(errors.New("db error"))

		filter := LernmittelFilter{}
		result, err := repo.GetLernmittelTitel(ctx, "Biologie", false, filter)

		assert.ErrorContains(t, err, "lernmittel eines fachs: db error")
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan error", func(t *testing.T) {
		mock.ExpectQuery("WITH titel AS").
			WithArgs(0, "", "", false, "Biologie").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "titel", "autor", "subject", "cover_url", "isbn",
				"jahrgang_von", "jahrgang_bis", "track", "gezaehlt",
				"gesamt", "verliehen", "verfuegbar", "auflagen",
			}).
				AddRow("uuid-1", "Bio 1", "Autor 1", "Biologie", "url1", "123", "wrong_type_for_int", 6, "Gymnasium", "01.01.2023", 10, 2, 8, nil))

		filter := LernmittelFilter{}
		result, err := repo.GetLernmittelTitel(ctx, "Biologie", false, filter)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("json unmarshal error", func(t *testing.T) {
		mock.ExpectQuery("WITH titel AS").
			WithArgs(0, "", "", false, "Physik").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "titel", "autor", "subject", "cover_url", "isbn",
				"jahrgang_von", "jahrgang_bis", "track", "gezaehlt",
				"gesamt", "verliehen", "verfuegbar", "auflagen",
			}).
				AddRow("uuid-1", "Physik 1", "Autor 1", "Physik", "url1", "123", 5, 6, "Gymnasium", "01.01.2023", 10, 2, 8, []byte(`invalid_json`)))

		filter := LernmittelFilter{}
		result, err := repo.GetLernmittelTitel(ctx, "Physik", false, filter)

		assert.ErrorContains(t, err, "auflagen von uuid-1:")
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
