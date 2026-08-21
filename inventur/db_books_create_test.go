package inventur

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBook(t *testing.T) {
	book := Book{
		ISBN:                    "123",
		Title:                   "Title",
		Author:                  "Author",
		CoverURL:                "URL",
		Subject:                 "Math",
		GradeLevel:              5,
		Track:                   "A",
		Stock:                   10,
		LastCounted:             nil,
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

	insertQuery := `INSERT INTO buecher_titel \(isbn, titel, autor, cover_url, subject, grade_level, track, last_counted, medientyp, erweiterte_eigenschaften, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, beschreibung, signatur\) VALUES \(\$1, \$2, \$3, \$4, NULLIF\(\$5, ''\), \$6, \$7, NULLIF\(\$8::text, ''\)::date, \$9, \$10, \$11, \$12, \$13, \$14, \$15, \$16, NULLIF\(\$17, ''\)\) RETURNING id`

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, book.Subject)
		mock.ExpectBegin()
		mock.ExpectQuery(insertQuery).
			WithArgs(
				book.ISBN, book.Title, book.Author, book.CoverURL, book.Subject, book.GradeLevel, book.Track, book.LastCounted, book.Medientyp, book.ErweiterteEigenschaften, book.JahrgangVon, book.JahrgangBis, book.Untertitel, book.Verlag, book.Erscheinungsjahr, book.Beschreibung, book.Signatur,
			).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("book-123"))

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare WHERE titel_id = \$1 AND ist_ausgesondert = false`).
			WithArgs("book-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectExec(`CREATE SEQUENCE IF NOT EXISTS sys_barcode_seq START 100000`).
			WillReturnResult(pgxmock.NewResult("CREATE", 0))
		mock.ExpectExec(`INSERT INTO buecher_exemplare`).
			WithArgs("book-123", 10).
			WillReturnResult(pgxmock.NewResult("INSERT", 10))

		mock.ExpectCommit()

		id, err := repo.CreateBook(ctx, book)
		assert.NoError(t, err)
		assert.Equal(t, "book-123", id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

    t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, book.Subject)
		mock.ExpectBegin()
		mock.ExpectQuery(insertQuery).
			WithArgs(
				book.ISBN, book.Title, book.Author, book.CoverURL, book.Subject, book.GradeLevel, book.Track, book.LastCounted, book.Medientyp, book.ErweiterteEigenschaften, book.JahrgangVon, book.JahrgangBis, book.Untertitel, book.Verlag, book.Erscheinungsjahr, book.Beschreibung, book.Signatur,
			).
			WillReturnError(fmt.Errorf("db connection failed"))
		mock.ExpectRollback()

		_, err = repo.CreateBook(ctx, book)
		assert.ErrorContains(t, err, "buch konnte nicht erstellt werden")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

    t.Run("sync error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, book.Subject)
		mock.ExpectBegin()
		mock.ExpectQuery(insertQuery).
			WithArgs(
				book.ISBN, book.Title, book.Author, book.CoverURL, book.Subject, book.GradeLevel, book.Track, book.LastCounted, book.Medientyp, book.ErweiterteEigenschaften, book.JahrgangVon, book.JahrgangBis, book.Untertitel, book.Verlag, book.Erscheinungsjahr, book.Beschreibung, book.Signatur,
			).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("book-123"))

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM buecher_exemplare`).
			WithArgs("book-123").
			WillReturnError(fmt.Errorf("bestand nicht lesbar"))
		mock.ExpectRollback()

		_, err = repo.CreateBook(ctx, book)
		assert.ErrorContains(t, err, "exemplare konnten nicht angelegt werden")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUpsertBook(t *testing.T) {
	book := Book{
		ISBN:                    "123",
		Title:                   "Title",
		Author:                  "Author",
		CoverURL:                "URL",
		Subject:                 "Math",
		GradeLevel:              5,
		Track:                   "A",
		Stock:                   2,
		LastCounted:             nil,
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

	upsertQuery := `INSERT INTO buecher_titel \(isbn, titel, autor, cover_url, subject, grade_level, track, last_counted, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, beschreibung, erweiterte_eigenschaften, signatur\) VALUES \(\$1, \$2, \$3, \$4, NULLIF\(\$5, ''\), \$6, \$7, NULLIF\(\$8::text, ''\)::date, \$9, \$10, \$11, \$12, \$13, \$14, \$15, \$16, NULLIF\(\$17, ''\)\) ON CONFLICT \(isbn\) DO UPDATE SET titel = EXCLUDED.titel, autor = EXCLUDED.autor, cover_url = EXCLUDED.cover_url, subject = EXCLUDED.subject, grade_level = EXCLUDED.grade_level, track = EXCLUDED.track, last_counted = EXCLUDED.last_counted, medientyp = EXCLUDED.medientyp, jahrgang_von = EXCLUDED.jahrgang_von, jahrgang_bis = EXCLUDED.jahrgang_bis, untertitel = EXCLUDED.untertitel, verlag = EXCLUDED.verlag, erscheinungsjahr = EXCLUDED.erscheinungsjahr, beschreibung = EXCLUDED.beschreibung, erweiterte_eigenschaften = EXCLUDED.erweiterte_eigenschaften, signatur = COALESCE\(NULLIF\(EXCLUDED.signatur, ''\), buecher_titel.signatur\) RETURNING id`

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, book.Subject)
		mock.ExpectBegin()
		mock.ExpectQuery(upsertQuery).
			WithArgs(
				book.ISBN, book.Title, book.Author, book.CoverURL, book.Subject, book.GradeLevel, book.Track, book.LastCounted, book.Medientyp, book.JahrgangVon, book.JahrgangBis, book.Untertitel, book.Verlag, book.Erscheinungsjahr, book.Beschreibung, book.ErweiterteEigenschaften, book.Signatur,
			).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("book-123"))

		// Upsert calls legeImportExemplareAn
		mock.ExpectExec(`CREATE SEQUENCE IF NOT EXISTS sys_barcode_seq START 100000`).
			WillReturnResult(pgxmock.NewResult("CREATE", 0))
		mock.ExpectExec(`INSERT INTO buecher_exemplare`).
			WithArgs([]string{book.ISBN}, []int32{int32(book.Stock)}).
			WillReturnResult(pgxmock.NewResult("INSERT", 2))

		mock.ExpectCommit()

		id, err := repo.UpsertBook(ctx, book)
		assert.NoError(t, err)
		assert.Equal(t, "book-123", id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, book.Subject)
		mock.ExpectBegin()
		mock.ExpectQuery(upsertQuery).
			WithArgs(
				book.ISBN, book.Title, book.Author, book.CoverURL, book.Subject, book.GradeLevel, book.Track, book.LastCounted, book.Medientyp, book.JahrgangVon, book.JahrgangBis, book.Untertitel, book.Verlag, book.Erscheinungsjahr, book.Beschreibung, book.ErweiterteEigenschaften, book.Signatur,
			).
			WillReturnError(fmt.Errorf("db connection failed"))
		mock.ExpectRollback()

		_, err = repo.UpsertBook(ctx, book)
		assert.ErrorContains(t, err, "buch konnte nicht importiert werden")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUpsertBooksBatch(t *testing.T) {
	books := []Book{
		{
			ISBN:                    "123",
			Title:                   "Title",
			Author:                  "Author",
			CoverURL:                "URL",
			Subject:                 "Math",
			GradeLevel:              5,
			Track:                   "A",
			Stock:                   2,
			LastCounted:             nil,
			Medientyp:               "Buch",
			ErweiterteEigenschaften: map[string]any{"key": "value"},
			JahrgangVon:             5,
			JahrgangBis:             6,
			Untertitel:              "Subtitle",
			Verlag:                  "Publisher",
			Erscheinungsjahr:        2023,
			Beschreibung:            "Description",
			Signatur:                "SIG-123",
		},
	}

	batchQuery := `INSERT INTO buecher_titel \(isbn, titel, autor, cover_url, subject, grade_level, track, last_counted, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, beschreibung, erweiterte_eigenschaften, signatur\) SELECT t.isbn, t.titel, t.autor, t.cover_url, NULLIF\(t.subject, ''\), t.grade_level, t.track, NULLIF\(t.last_counted_text, ''\)::date, t.medientyp, t.jahrgang_von, t.jahrgang_bis, t.untertitel, t.verlag, t.erscheinungsjahr, t.beschreibung, t.erweiterte_eigenschaften, NULLIF\(t.signatur, ''\) FROM UNNEST\(\$1::text\[\], \$2::text\[\], \$3::text\[\], \$4::text\[\], \$5::text\[\], \$6::smallint\[\], \$7::text\[\], \$8::text\[\], \$9::text\[\], \$10::int\[\], \$11::int\[\], \$12::text\[\], \$13::text\[\], \$14::int\[\], \$15::text\[\], \$16::jsonb\[\], \$17::text\[\]\) AS t\(isbn, titel, autor, cover_url, subject, grade_level, track, last_counted_text, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, beschreibung, erweiterte_eigenschaften, signatur\) ON CONFLICT \(isbn\) DO UPDATE SET titel = EXCLUDED.titel, autor = EXCLUDED.autor, cover_url = EXCLUDED.cover_url, subject = EXCLUDED.subject, grade_level = EXCLUDED.grade_level, track = EXCLUDED.track, last_counted = EXCLUDED.last_counted, medientyp = EXCLUDED.medientyp, jahrgang_von = EXCLUDED.jahrgang_von, jahrgang_bis = EXCLUDED.jahrgang_bis, untertitel = EXCLUDED.untertitel, verlag = EXCLUDED.verlag, erscheinungsjahr = EXCLUDED.erscheinungsjahr, beschreibung = EXCLUDED.beschreibung, erweiterte_eigenschaften = EXCLUDED.erweiterte_eigenschaften, signatur = COALESCE\(NULLIF\(EXCLUDED.signatur, ''\), buecher_titel.signatur\)`

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, books[0].Subject)
		mock.ExpectBegin()

		jsonProps, _ := json.Marshal(books[0].ErweiterteEigenschaften)
		mock.ExpectExec(batchQuery).
            WithArgs(
                []string{books[0].ISBN},
                []string{books[0].Title},
                []string{books[0].Author},
                []string{books[0].CoverURL},
                []string{books[0].Subject},
                []int16{books[0].GradeLevel},
                []string{books[0].Track},
                []*string{books[0].LastCounted},
                []string{books[0].Medientyp},
                []int{books[0].JahrgangVon},
                []int{books[0].JahrgangBis},
                []string{books[0].Untertitel},
                []string{books[0].Verlag},
                []int{books[0].Erscheinungsjahr},
                []string{books[0].Beschreibung},
                [][]byte{jsonProps},
                []string{books[0].Signatur},
            ).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		// Batch Upsert calls legeImportExemplareAn
		mock.ExpectExec(`CREATE SEQUENCE IF NOT EXISTS sys_barcode_seq START 100000`).
			WillReturnResult(pgxmock.NewResult("CREATE", 0))
		mock.ExpectExec(`INSERT INTO buecher_exemplare`).
			WithArgs([]string{books[0].ISBN}, []int32{int32(books[0].Stock)}).
			WillReturnResult(pgxmock.NewResult("INSERT", 2))

		mock.ExpectCommit()

		rowsAffected, err := repo.UpsertBooksBatch(ctx, books)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rowsAffected)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

    t.Run("empty books", func(t *testing.T) {
        mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

        repo := NewBookRepository(mock)
		ctx := context.Background()

        rowsAffected, err := repo.UpsertBooksBatch(ctx, []Book{})
        assert.NoError(t, err)
        assert.Equal(t, int64(0), rowsAffected)
    })

	t.Run("db error on batch upsert", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewBookRepository(mock)
		ctx := context.Background()

		erwarteFachBekannt(mock, books[0].Subject)
		mock.ExpectBegin()

        jsonProps, _ := json.Marshal(books[0].ErweiterteEigenschaften)
		mock.ExpectExec(batchQuery).
            WithArgs(
                []string{books[0].ISBN},
                []string{books[0].Title},
                []string{books[0].Author},
                []string{books[0].CoverURL},
                []string{books[0].Subject},
                []int16{books[0].GradeLevel},
                []string{books[0].Track},
                []*string{books[0].LastCounted},
                []string{books[0].Medientyp},
                []int{books[0].JahrgangVon},
                []int{books[0].JahrgangBis},
                []string{books[0].Untertitel},
                []string{books[0].Verlag},
                []int{books[0].Erscheinungsjahr},
                []string{books[0].Beschreibung},
                [][]byte{jsonProps},
                []string{books[0].Signatur},
            ).
			WillReturnError(fmt.Errorf("db batch failed"))
		mock.ExpectRollback()

		_, err = repo.UpsertBooksBatch(ctx, books)
		assert.ErrorContains(t, err, "bücher konnten nicht im batch importiert werden")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
