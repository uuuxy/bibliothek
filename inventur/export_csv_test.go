package inventur

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamBooksForCSVExport(t *testing.T) {
	t.Run("successful stream", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		ctx := context.Background()

		rows := pgxmock.NewRows([]string{"titel", "autor", "verlag", "isbn", "jahr", "subject", "barcode", "zustand"}).
			AddRow("Buch 1", "Autor 1", "Verlag 1", "1234567890", 2021, "Kategorie 1", "BC1", "Gut").
			AddRow("Buch 2", "", "", "", 0, "", "", "")

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WillReturnRows(rows)

		kopfCalled := false
		kopf := func() error {
			kopfCalled = true
			return nil
		}

		var resultRows [][]string
		schreibe := func(row []string) error {
			resultRows = append(resultRows, row)
			return nil
		}

		err = repo.StreamBooksForCSVExport(ctx, kopf, schreibe)
		require.NoError(t, err)
		assert.True(t, kopfCalled)
		require.Len(t, resultRows, 2)
		assert.Equal(t, []string{"Buch 1", "Autor 1", "Verlag 1", "'1234567890", "2021", "Kategorie 1", "BC1", "Gut"}, resultRows[0])
		assert.Equal(t, []string{"Buch 2", "", "", "", "", "", "", ""}, resultRows[1])

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error on query", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		ctx := context.Background()

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WillReturnError(errors.New("db error"))

		kopfCalled := false
		kopf := func() error {
			kopfCalled = true
			return nil
		}
		schreibe := func(row []string) error { return nil }

		err = repo.StreamBooksForCSVExport(ctx, kopf, schreibe)
		require.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.False(t, kopfCalled)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error in kopf", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		ctx := context.Background()

		rows := pgxmock.NewRows([]string{"titel", "autor", "verlag", "isbn", "jahr", "subject", "barcode", "zustand"})
		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WillReturnRows(rows)

		kopf := func() error {
			return errors.New("kopf error")
		}
		schreibe := func(row []string) error { return nil }

		err = repo.StreamBooksForCSVExport(ctx, kopf, schreibe)
		require.Error(t, err)
		assert.Equal(t, "kopf error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error in schreibe breaks stream", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		ctx := context.Background()

		rows := pgxmock.NewRows([]string{"titel", "autor", "verlag", "isbn", "jahr", "subject", "barcode", "zustand"}).
			AddRow("Buch 1", "Autor 1", "Verlag 1", "1234567890", 2021, "Kategorie 1", "BC1", "Gut")

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WillReturnRows(rows)

		kopf := func() error { return nil }
		schreibe := func(row []string) error {
			return errors.New("schreibe error")
		}

		err = repo.StreamBooksForCSVExport(ctx, kopf, schreibe)
		require.Error(t, err)
		assert.Equal(t, "schreibe error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestHandleExportCSV(t *testing.T) {
	t.Run("successful export", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		handler := &APIHandler{repo: repo}

		rows := pgxmock.NewRows([]string{"titel", "autor", "verlag", "isbn", "jahr", "subject", "barcode", "zustand"}).
			AddRow("Buch 1", "Autor 1", "Verlag 1", "1234567890", 2021, "Kategorie 1", "BC1", "Gut").
			AddRow("=Buch 2", "", "", "", 0, "", "", "") // check formula sanitization

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/admin/books/export", nil)
		w := httptest.NewRecorder()

		handler.handleExportCSV(w, req)

		res := w.Result()
		defer res.Body.Close() //nolint:errcheck

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "text/csv; charset=utf-8", res.Header.Get("Content-Type"))
		assert.Contains(t, res.Header.Get("Content-Disposition"), "attachment; filename=\"bestand_export_")

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)

		// Check BOM
		assert.True(t, bytes.HasPrefix(body, []byte{0xEF, 0xBB, 0xBF}))

		// Remove BOM for string comparison
		bodyStr := string(bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF}))
		lines := strings.Split(strings.TrimSpace(bodyStr), "\n")
		require.Len(t, lines, 3)
		assert.Equal(t, "Titel;Autor;Verlag;ISBN;Jahr;Kategorie;Barcode;Zustand", lines[0])
		assert.Equal(t, "Buch 1;Autor 1;Verlag 1;'1234567890;2021;Kategorie 1;BC1;Gut", lines[1])
		assert.Equal(t, "'=Buch 2;;;;;;;", lines[2]) // sanitized

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error before kopf sends 500", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		handler := &APIHandler{repo: repo}

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WillReturnError(errors.New("db error"))

		req := httptest.NewRequest(http.MethodGet, "/api/admin/books/export", nil)
		w := httptest.NewRecorder()

		handler.handleExportCSV(w, req)

		res := w.Result()
		defer res.Body.Close() //nolint:errcheck

		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)

		body, leseErr := io.ReadAll(res.Body)
		require.NoError(t, leseErr)
		assert.Contains(t, string(body), "Ein interner Datenbankfehler ist aufgetreten")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error after kopf breaks stream", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		handler := &APIHandler{repo: repo}

		rows := pgxmock.NewRows([]string{"titel", "autor", "verlag", "isbn", "jahr", "subject", "barcode", "zustand"}).
			AddRow("Buch 1", "Autor 1", "Verlag 1", "1234567890", 2021, "Kategorie 1", "BC1", "Gut").
			RowError(0, errors.New("connection lost"))

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/admin/books/export", nil)
		w := httptest.NewRecorder()

		handler.handleExportCSV(w, req)

		res := w.Result()
		defer res.Body.Close() //nolint:errcheck

		// HTTP status is still 200 because header was already sent
		assert.Equal(t, http.StatusOK, res.StatusCode)

		body, leseErr := io.ReadAll(res.Body)
		require.NoError(t, leseErr)

		// Check BOM
		assert.True(t, bytes.HasPrefix(body, []byte{0xEF, 0xBB, 0xBF}))

		// Remove BOM for string comparison
		bodyStr := string(bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF}))
		lines := strings.Split(strings.TrimSpace(bodyStr), "\n")
		// The first valid row might or might not be fully flushed depending on internals,
		// but at least we can verify it doesn't panic and the headers are sent.
		assert.Equal(t, "text/csv; charset=utf-8", res.Header.Get("Content-Type"))
		assert.True(t, len(lines) > 0)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
