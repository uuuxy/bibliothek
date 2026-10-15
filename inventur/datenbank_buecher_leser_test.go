package inventur

import (
	"context"
	"fmt"
	"testing"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTitelAbfragen_LadeTitelListe(t *testing.T) {
	ctx := context.Background()
	zuletztGezaehlt := "2023-01-01"

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT.+FROM buecher_titel bt LEFT JOIN buecher_exemplare e.+`).
			WithArgs("Math", "algebra", 50000).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "ist_lernmittel",
				"verfuegbar", "gesamt", "im_zulauf", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis",
				"untertitel", "verlag", "erscheinungsjahr", "erweiterte_eigenschaften", "auflage",
				"listenpreis", "mehrjahresband", "werk_id", "werk_rang",
			}).AddRow(
				"book-1", "123", "Algebra", "Smith", "SIG-1", "url", "Math", "A", false,
				2, 3, 0, &zuletztGezaehlt, 1, "Buch", 5, 6,
				"", "", 2020, map[string]any{}, "4. Aufl. 2023", nil, false, "werk-1", 2,
			))
		mock.ExpectQuery(`SELECT tsw.titel_id.+FROM schlagworte sw`).
			WithArgs([]string{"book-1"}).
			WillReturnRows(pgxmock.NewRows([]string{"titel_id", "woerter"}))
		mock.ExpectQuery(`SELECT e.titel_id::text, e.standort.+FROM buecher_exemplare e`).
			WithArgs([]string{"book-1"}).
			WillReturnRows(pgxmock.NewRows([]string{"titel_id", "standort", "anzahl"}).
				AddRow("book-1", "Lehrerschrank", 2))

		eintraege, err := abfragen.LadeTitelListe(ctx, "Math", "algebra", false)
		assert.NoError(t, err)
		assert.Len(t, eintraege, 1)
		if len(eintraege) > 0 {
			assert.Equal(t, "book-1", eintraege[0].ID)
			assert.Equal(t, 3, eintraege[0].Bestand)
			// Die Standorte der Exemplare kommen mit der Liste.
			assert.Equal(t, []repository.StandortZahl{{Standort: "Lehrerschrank", Anzahl: 2}}, eintraege[0].Standorte)
			// Buch und Rang der Auflage stehen am Ende der Zeile.
			assert.Equal(t, "werk-1", eintraege[0].WerkID)
			assert.Equal(t, 2, eintraege[0].WerkRang)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT.+FROM buecher_titel bt LEFT JOIN buecher_exemplare e.+`).
			WithArgs("", "", 50000).
			WillReturnError(fmt.Errorf("db connection failed"))

		eintraege, err := abfragen.LadeTitelListe(ctx, "", "", false)
		assert.ErrorContains(t, err, "bücher konnten nicht geladen werden")
		assert.Nil(t, eintraege)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTitelAbfragen_LadeTitelMitFremdemCover(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url FROM buecher_titel WHERE cover_url LIKE 'http%' ORDER BY id ASC LIMIT \$1`).
			WithArgs(10).
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "cover_url"}).
				AddRow("book-1", "123", "Title 1", "http://example.com/1.jpg"))

		eintraege, err := abfragen.LadeTitelMitFremdemCover(ctx, 10)
		assert.NoError(t, err)
		assert.Len(t, eintraege, 1)
		if len(eintraege) > 0 {
			assert.Equal(t, "book-1", eintraege[0].ID)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with default limit", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url FROM buecher_titel WHERE cover_url LIKE 'http%' ORDER BY id ASC LIMIT \$1`).
			WithArgs(100).
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "cover_url"}).
				AddRow("book-1", "123", "Title 1", "http://example.com/1.jpg"))

		eintraege, err := abfragen.LadeTitelMitFremdemCover(ctx, 0)
		assert.NoError(t, err)
		assert.Len(t, eintraege, 1)
		if len(eintraege) > 0 {
			assert.Equal(t, "book-1", eintraege[0].ID)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url FROM buecher_titel WHERE cover_url LIKE 'http%' ORDER BY id ASC LIMIT \$1`).
			WithArgs(100).
			WillReturnError(fmt.Errorf("db failure"))

		eintraege, err := abfragen.LadeTitelMitFremdemCover(ctx, -1)
		assert.ErrorContains(t, err, "bücher mit externen covern konnten nicht geladen werden")
		assert.Nil(t, eintraege)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url FROM buecher_titel WHERE cover_url LIKE 'http%' ORDER BY id ASC LIMIT \$1`).
			WithArgs(10).
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title"}). // Missing column to trigger scan error
												AddRow("book-1", "123", "Title 1"))

		eintraege, err := abfragen.LadeTitelMitFremdemCover(ctx, 10)
		assert.ErrorContains(t, err, "bücher mit externen covern konnten nicht gelesen werden")
		assert.Nil(t, eintraege)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("iteration error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url FROM buecher_titel WHERE cover_url LIKE 'http%' ORDER BY id ASC LIMIT \$1`).
			WithArgs(10).
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "cover_url"}).
				AddRow("book-1", "123", "Title 1", "http://example.com/1.jpg").
				RowError(1, fmt.Errorf("row error")))

		eintraege, err := abfragen.LadeTitelMitFremdemCover(ctx, 10)
		assert.ErrorContains(t, err, "fehler beim iterieren externer cover-bücher")
		assert.Nil(t, eintraege)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTitelAbfragen_LadeTitelNachIDs(t *testing.T) {
	ctx := context.Background()
	zuletztGezaehlt := "2023-01-01"

	t.Run("empty ids", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		eintraege, err := abfragen.LadeTitelNachIDs(ctx, []string{})
		assert.NoError(t, err)
		assert.Empty(t, eintraege)
		// No query expected
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		ids := []string{"book-1"}
		mock.ExpectQuery(`SELECT.+FROM buecher_titel bt LEFT JOIN buecher_exemplare e.+`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "ist_lernmittel",
				"verfuegbar", "gesamt", "im_zulauf", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis",
				"untertitel", "verlag", "erscheinungsjahr", "erweiterte_eigenschaften", "auflage",
				"listenpreis", "mehrjahresband", "werk_id", "werk_rang",
			}).AddRow(
				"book-1", "123", "Algebra", "Smith", "SIG-1", "url", "Math", "A", false,
				2, 3, 0, &zuletztGezaehlt, 1, "Buch", 5, 6,
				"", "", 2020, map[string]any{}, "4. Aufl. 2023", nil, false, "", 0,
			))

		eintraege, err := abfragen.LadeTitelNachIDs(ctx, ids)
		assert.NoError(t, err)
		assert.Len(t, eintraege, 1)
		if len(eintraege) > 0 {
			assert.Equal(t, "book-1", eintraege[0].ID)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		ids := []string{"book-1"}
		mock.ExpectQuery(`SELECT.+FROM buecher_titel bt LEFT JOIN buecher_exemplare e.+`).
			WithArgs(ids).
			WillReturnError(fmt.Errorf("db connection failed"))

		eintraege, err := abfragen.LadeTitelNachIDs(ctx, ids)
		assert.ErrorContains(t, err, "bücher nach ids konnten nicht geladen werden")
		assert.Nil(t, eintraege)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
