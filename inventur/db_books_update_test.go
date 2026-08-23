package inventur

import (
	"context"
	"fmt"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateBook(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	book := Book{
		ISBN:                    "123",
		Title:                   "Title",
		Author:                  "Author",
		CoverURL:                "URL",
		Subject:                 "Math",
		GradeLevel:              5,
		Track:                   "A",
		Stock:                   10,
		LastCounted:             nil, // handle date logic if needed
		Medientyp:               "Buch",
		ErweiterteEigenschaften: map[string]any{"key": "value"},
		JahrgangVon:             5,
		JahrgangBis:             6,
		Untertitel:              "Subtitle",
		Verlag:                  "Publisher",
		Erscheinungsjahr:        2023,
		Beschreibung:            "Description",
		Signatur:                "SIG-123",
	}

	updateQuery := `UPDATE buecher_titel SET isbn = \$1, titel = \$2, autor = \$3, cover_url = \$4, subject = NULLIF\(\$5, ''\), grade_level = \$6, track = \$7, last_counted = NULLIF\(\$8::text, ''\)::date, medientyp = \$9, erweiterte_eigenschaften = \$10, jahrgang_von = \$11, jahrgang_bis = \$12, untertitel = \$13, verlag = \$14, erscheinungsjahr = \$15, beschreibung = \$16, signatur = COALESCE\(NULLIF\(\$18, ''\), signatur\), aktualisiert_am = NOW\(\) WHERE id = \$17`
	// also note syncBookStock will be called

	t.Run("success", func(t *testing.T) {
		erwarteFachBekannt(mock, book.Subject) // läuft auf dem Pool, vor der Tx
		mock.ExpectBegin()
		mock.ExpectExec(updateQuery).
			WithArgs(
				book.ISBN, book.Title, book.Author, book.CoverURL, book.Subject, book.GradeLevel, book.Track, book.LastCounted, book.Medientyp, book.ErweiterteEigenschaften, book.JahrgangVon, book.JahrgangBis, book.Untertitel, book.Verlag, book.Erscheinungsjahr, book.Beschreibung, "book-123", book.Signatur,
			).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		// syncBookStock query (in der Tx)
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare WHERE titel_id = \$1 AND ist_ausgesondert = false`).
			WithArgs("book-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(10))
		mock.ExpectCommit()

		err := repo.UpdateBook(ctx, "book-123", book, &book.Stock)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		erwarteFachBekannt(mock, book.Subject)
		mock.ExpectBegin()
		mock.ExpectExec(updateQuery).
			WithArgs(
				book.ISBN, book.Title, book.Author, book.CoverURL, book.Subject, book.GradeLevel, book.Track, book.LastCounted, book.Medientyp, book.ErweiterteEigenschaften, book.JahrgangVon, book.JahrgangBis, book.Untertitel, book.Verlag, book.Erscheinungsjahr, book.Beschreibung, "book-123", book.Signatur,
			).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		mock.ExpectRollback()

		err := repo.UpdateBook(ctx, "book-123", book, &book.Stock)
		assert.ErrorIs(t, err, ErrBookNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		erwarteFachBekannt(mock, book.Subject)
		mock.ExpectBegin()
		mock.ExpectExec(updateQuery).
			WithArgs(
				book.ISBN, book.Title, book.Author, book.CoverURL, book.Subject, book.GradeLevel, book.Track, book.LastCounted, book.Medientyp, book.ErweiterteEigenschaften, book.JahrgangVon, book.JahrgangBis, book.Untertitel, book.Verlag, book.Erscheinungsjahr, book.Beschreibung, "book-123", book.Signatur,
			).
			WillReturnError(fmt.Errorf("db connection failed"))
		mock.ExpectRollback()

		err := repo.UpdateBook(ctx, "book-123", book, &book.Stock)
		assert.ErrorContains(t, err, "buch konnte nicht aktualisiert werden")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Transaktionsgrenzen-Sweep A3: Ein Fehlschlag der Bestands-Synchronisierung wird
	// jetzt ZURÜCKGEGEBEN (früher nur geloggt → Titel mit falschem Bestand, aber Erfolg
	// gemeldet). Der Titel-Update rollt mit zurück.
	t.Run("sync-fehler wird zurückgegeben und rollt zurück", func(t *testing.T) {
		erwarteFachBekannt(mock, book.Subject)
		mock.ExpectBegin()
		mock.ExpectExec(updateQuery).
			WithArgs(
				book.ISBN, book.Title, book.Author, book.CoverURL, book.Subject, book.GradeLevel, book.Track, book.LastCounted, book.Medientyp, book.ErweiterteEigenschaften, book.JahrgangVon, book.JahrgangBis, book.Untertitel, book.Verlag, book.Erscheinungsjahr, book.Beschreibung, "book-123", book.Signatur,
			).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare`).
			WithArgs("book-123").
			WillReturnError(fmt.Errorf("bestand nicht lesbar"))
		mock.ExpectRollback()

		err := repo.UpdateBook(ctx, "book-123", book, &book.Stock)
		assert.ErrorContains(t, err, "exemplare konnten nicht synchronisiert werden")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSyncBookStock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	t.Run("increase stock", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare WHERE titel_id = \$1 AND ist_ausgesondert = false`).
			WithArgs("book-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

		mock.ExpectExec(`CREATE SEQUENCE IF NOT EXISTS sys_barcode_seq START 100000`).
			WillReturnResult(pgxmock.NewResult("CREATE", 0))

		mock.ExpectExec(`INSERT INTO buecher_exemplare`).
			WithArgs("book-123", 3). // 5 - 2 = 3
			WillReturnResult(pgxmock.NewResult("INSERT", 3))

		err := repo.syncBookStock(ctx, mock, "book-123", 5)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("decrease stock - only unused", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare WHERE titel_id = \$1 AND ist_ausgesondert = false`).
			WithArgs("book-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(5))

		// Try to retire 2 unused
		mock.ExpectExec(`UPDATE buecher_exemplare SET ist_ausgesondert = true`).
			WithArgs("book-123", 2).
			WillReturnResult(pgxmock.NewResult("UPDATE", 2))

		err := repo.syncBookStock(ctx, mock, "book-123", 3)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("decrease stock - fallback to used", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare WHERE titel_id = \$1 AND ist_ausgesondert = false`).
			WithArgs("book-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(5))

		// Try to retire 2 unused, but only 1 found
		mock.ExpectExec(`UPDATE buecher_exemplare SET ist_ausgesondert = true`).
			WithArgs("book-123", 2).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		// Fallback for remaining 1
		mock.ExpectExec(`UPDATE buecher_exemplare SET ist_ausgesondert = true`).
			WithArgs("book-123", int64(1)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.syncBookStock(ctx, mock, "book-123", 3)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no change in stock", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare WHERE titel_id = \$1 AND ist_ausgesondert = false`).
			WithArgs("book-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(5))

		err := repo.syncBookStock(ctx, mock, "book-123", 5)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
