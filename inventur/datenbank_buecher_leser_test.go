package inventur

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestListBooks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	queryRegex := `SELECT\s+bt\.id, COALESCE\(bt\.isbn, ''\) AS isbn, bt\.titel AS title, COALESCE\(bt\.autor, ''\) AS author,\s+COALESCE\(bt\.cover_url, ''\) AS cover_url, COALESCE\(bt\.subject, ''\) AS subject,\s+COALESCE\(bt\.grade_level, 0\) AS grade_level, COALESCE\(bt\.track, ''\) AS track,\s+bt\.stock,\s+COUNT\(e\.id\) FILTER \(WHERE e\.ist_ausleihbar = true AND e\.ist_ausgesondert = false AND a\.id IS NULL\) AS verfuegbar,\s+COUNT\(e\.id\) FILTER \(WHERE e\.ist_ausgesondert = false AND coalesce\(e\.zustand_notiz, ''\) NOT LIKE 'Im Zulauf%' AND coalesce\(e\.zustand_notiz, ''\) != 'bestellt' AND coalesce\(e\.zustand_notiz, ''\) NOT LIKE 'Bestellt%'\) AS gesamt,\s+TO_CHAR\(bt\.last_counted, 'YYYY-MM-DD'\) as last_counted, bt\.sort_order, COALESCE\(bt\.medientyp, 'Buch'\) AS medientyp, bt\.erweiterte_eigenschaften\s+FROM buecher_titel bt\s+LEFT JOIN buecher_exemplare e ON e\.titel_id = bt\.id\s+LEFT JOIN ausleihen a ON a\.exemplar_id = e\.id AND a\.rueckgabe_am IS NULL\s+WHERE \(\$1 = '' OR bt\.subject = \$1\)\s+AND \(\$2::smallint IS NULL OR bt\.grade_level = \$2\)\s+AND \(\$3 = '' OR bt\.titel ILIKE '%' \|\| \$3 \|\| '%' OR bt\.autor ILIKE '%' \|\| \$3 \|\| '%' OR bt\.isbn ILIKE '%' \|\| \$3 \|\| '%' OR bt\.subject ILIKE '%' \|\| \$3 \|\| '%' OR CAST\(bt\.id AS TEXT\) ILIKE '%' \|\| \$3 \|\| '%'\)\s+GROUP BY bt\.id, bt\.titel, bt\.autor, bt\.isbn, bt\.cover_url, bt\.subject, bt\.grade_level, bt\.track, bt\.stock, bt\.last_counted, bt\.sort_order, bt\.medientyp, bt\.erweiterte_eigenschaften\s+ORDER BY bt\.sort_order ASC, bt\.titel ASC`

	columns := []string{
		"id", "isbn", "title", "author", "cover_url", "subject", "grade_level", "track", "stock", "verfuegbar", "gesamt", "last_counted", "sort_order", "medientyp", "erweiterte_eigenschaften",
	}

	lastCounted := "2023-01-01"

	t.Run("Success without filters", func(t *testing.T) {
		mock.ExpectQuery(queryRegex).
			WithArgs("", (*int16)(nil), "").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("1", "123", "Title 1", "Author 1", "url1", "Math", int16(5), "A", 10, 5, 8, &lastCounted, 1, "Buch", nil))

		books, err := repo.ListBooks(context.Background(), "", nil, "")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if len(books) != 1 {
			t.Errorf("expected 1 book, got %d", len(books))
		}

		if books[0].Title != "Title 1" {
			t.Errorf("expected title 'Title 1', got '%s'", books[0].Title)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Success with filters", func(t *testing.T) {
		grade := int16(7)
		mock.ExpectQuery(queryRegex).
			WithArgs("Physics", &grade, "Query").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("2", "456", "Title 2", "Author 2", "url2", "Physics", int16(7), "B", 5, 2, 4, &lastCounted, 2, "Buch", nil))

		books, err := repo.ListBooks(context.Background(), "Physics", &grade, "Query")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if len(books) != 1 {
			t.Errorf("expected 1 book, got %d", len(books))
		}

		if books[0].Subject != "Physics" {
			t.Errorf("expected subject 'Physics', got '%s'", books[0].Subject)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Query error", func(t *testing.T) {
		mock.ExpectQuery(queryRegex).
			WithArgs("", (*int16)(nil), "").
			WillReturnError(errors.New("db error"))

		books, err := repo.ListBooks(context.Background(), "", nil, "")
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		if books != nil {
			t.Errorf("expected books to be nil, got %v", books)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Row scan error", func(t *testing.T) {
		mock.ExpectQuery(queryRegex).
			WithArgs("", (*int16)(nil), "").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("invalid_id", "123", "Title", "Author", "url", "Subject", int16(5), "A", "invalid_stock", 5, 8, &lastCounted, 1, "Buch", nil)) // Stock expects int, got string

		books, err := repo.ListBooks(context.Background(), "", nil, "")
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		if books != nil {
			t.Errorf("expected books to be nil, got %v", books)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Row iteration error", func(t *testing.T) {
		mock.ExpectQuery(queryRegex).
			WithArgs("", (*int16)(nil), "").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("1", "123", "Title 1", "Author 1", "url1", "Math", int16(5), "A", 10, 5, 8, &lastCounted, 1, "Buch", nil).
				RowError(0, errors.New("iteration error")))

		books, err := repo.ListBooks(context.Background(), "", nil, "")
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		if books != nil {
			t.Errorf("expected books to be nil, got %v", books)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
