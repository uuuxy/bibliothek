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

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exportSpalten sind die Spalten der Export-Abfrage in ihrer Reihenfolge.
var exportSpalten = []string{"titel", "autor", "verlag", "isbn", "jahr", "subject", "barcode", "zustand", "signatur", "schlagworte", "topf", "standort"}

func TestLadeTitelFuerCSVExport(t *testing.T) {
	t.Run("successful stream", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		zeilen := pgxmock.NewRows(exportSpalten).
			AddRow("Buch 1", "Autor 1", "Verlag 1", "1234567890", 2021, "Kategorie 1", "BC1", "Gut", "Ma 8", "Algebra | Geometrie", "land", "Regal 3B").
			AddRow("Buch 2", "", "", "", 0, "", "", "", "", "", "", "")

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnRows(zeilen)

		kopfGerufen := false
		kopf := func() error {
			kopfGerufen = true
			return nil
		}

		var ergebnisZeilen [][]string
		schreibe := func(zeile []string) error {
			ergebnisZeilen = append(ergebnisZeilen, zeile)
			return nil
		}

		err = abfragen.LadeTitelFuerCSVExport(ctx, kopf, schreibe)
		require.NoError(t, err)
		assert.True(t, kopfGerufen)
		require.Len(t, ergebnisZeilen, 2)
		assert.Equal(t, []string{"Buch 1", "Autor 1", "Verlag 1", "'1234567890", "2021", "Kategorie 1", "BC1", "Gut", "Ma 8", "Algebra | Geometrie", "Land", "Regal 3B"}, ergebnisZeilen[0])
		assert.Equal(t, []string{"Buch 2", "", "", "", "", "", "", "", "", "", "", ""}, ergebnisZeilen[1])

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error on query", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnError(errors.New("db error"))

		kopfGerufen := false
		kopf := func() error {
			kopfGerufen = true
			return nil
		}
		schreibe := func(zeile []string) error { return nil }

		err = abfragen.LadeTitelFuerCSVExport(ctx, kopf, schreibe)
		require.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.False(t, kopfGerufen)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error in kopf", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		zeilen := pgxmock.NewRows(exportSpalten)
		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnRows(zeilen)

		kopf := func() error {
			return errors.New("kopf error")
		}
		schreibe := func(zeile []string) error { return nil }

		err = abfragen.LadeTitelFuerCSVExport(ctx, kopf, schreibe)
		require.Error(t, err)
		assert.Equal(t, "kopf error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error in schreibe breaks stream", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		zeilen := pgxmock.NewRows(exportSpalten).
			AddRow("Buch 1", "Autor 1", "Verlag 1", "1234567890", 2021, "Kategorie 1", "BC1", "Gut", "", "", "schultraeger", "")

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnRows(zeilen)

		kopf := func() error { return nil }
		schreibe := func(zeile []string) error {
			return errors.New("schreibe error")
		}

		err = abfragen.LadeTitelFuerCSVExport(ctx, kopf, schreibe)
		require.Error(t, err)
		assert.Equal(t, "schreibe error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCSVExport(t *testing.T) {
	t.Run("successful export", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		tueren := &Tueren{abfragen: abfragen}

		zeilen := pgxmock.NewRows(exportSpalten).
			AddRow("Buch 1", "Autor 1", "Verlag 1", "1234567890", 2021, "Kategorie 1", "BC1", "Gut", "Ma 8", "Algebra | Geometrie", "schultraeger", "Bibliothek, Regal 3B").
			AddRow("=Buch 2", "", "", "", 0, "", "", "", "", "", "", "-Keller") // Formelschutz: Titel und Standort

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnRows(zeilen)

		anfrage := httptest.NewRequest(http.MethodGet, "/api/admin/books/export", nil)
		w := httptest.NewRecorder()

		tueren.CSVExport(w, anfrage)

		ergebnis := w.Result()
		defer ergebnis.Body.Close() //nolint:errcheck

		assert.Equal(t, http.StatusOK, ergebnis.StatusCode)
		assert.Equal(t, "text/csv; charset=utf-8", ergebnis.Header.Get("Content-Type"))
		assert.Contains(t, ergebnis.Header.Get("Content-Disposition"), "attachment; filename=\"bestand_export_")

		rumpf, err := io.ReadAll(ergebnis.Body)
		require.NoError(t, err)

		// Das BOM prüfen.
		assert.True(t, bytes.HasPrefix(rumpf, []byte{0xEF, 0xBB, 0xBF}))

		// Das BOM für den Vergleich entfernen.
		rumpfText := string(bytes.TrimPrefix(rumpf, []byte{0xEF, 0xBB, 0xBF}))
		textZeilen := strings.Split(strings.TrimSpace(rumpfText), "\n")
		require.Len(t, textZeilen, 3)
		assert.Equal(t, "Titel;Autor;Verlag;ISBN;Jahr;Kategorie;Barcode;Zustand;Signatur;Schlagworte;Eigentum;Standort", textZeilen[0])
		assert.Equal(t, "Buch 1;Autor 1;Verlag 1;'1234567890;2021;Kategorie 1;BC1;Gut;Ma 8;Algebra | Geometrie;Schulträger;Bibliothek, Regal 3B", textZeilen[1])
		assert.Equal(t, "'=Buch 2;;;;;;;;;;;'-Keller", textZeilen[2])

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error before kopf sends 500", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		tueren := &Tueren{abfragen: abfragen}

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnError(errors.New("db error"))

		anfrage := httptest.NewRequest(http.MethodGet, "/api/admin/books/export", nil)
		w := httptest.NewRecorder()

		tueren.CSVExport(w, anfrage)

		ergebnis := w.Result()
		defer ergebnis.Body.Close() //nolint:errcheck

		assert.Equal(t, http.StatusInternalServerError, ergebnis.StatusCode)

		rumpf, leseFehler := io.ReadAll(ergebnis.Body)
		require.NoError(t, leseFehler)
		assert.Contains(t, string(rumpf), "Ein interner Datenbankfehler ist aufgetreten")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error after kopf breaks stream", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		tueren := &Tueren{abfragen: abfragen}

		zeilen := pgxmock.NewRows(exportSpalten).
			AddRow("Buch 1", "Autor 1", "Verlag 1", "1234567890", 2021, "Kategorie 1", "BC1", "Gut", "", "", "land", "").
			RowError(0, errors.New("connection lost"))

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnRows(zeilen)

		anfrage := httptest.NewRequest(http.MethodGet, "/api/admin/books/export", nil)
		w := httptest.NewRecorder()

		tueren.CSVExport(w, anfrage)

		ergebnis := w.Result()
		defer ergebnis.Body.Close() //nolint:errcheck

		// Der Status bleibt 200, weil die Kopfzeilen schon gesendet sind.
		assert.Equal(t, http.StatusOK, ergebnis.StatusCode)

		rumpf, leseFehler := io.ReadAll(ergebnis.Body)
		require.NoError(t, leseFehler)

		// Das BOM prüfen.
		assert.True(t, bytes.HasPrefix(rumpf, []byte{0xEF, 0xBB, 0xBF}))

		// Das BOM für den Vergleich entfernen.
		rumpfText := string(bytes.TrimPrefix(rumpf, []byte{0xEF, 0xBB, 0xBF}))
		textZeilen := strings.Split(strings.TrimSpace(rumpfText), "\n")
		// Ob die erste gültige Zeile schon ganz geschrieben ist, hängt vom Puffer ab; sicher ist, dass
		// nichts abbricht und die Kopfzeilen gesendet sind.
		assert.Equal(t, "text/csv; charset=utf-8", ergebnis.Header.Get("Content-Type"))
		assert.True(t, len(textZeilen) > 0)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
