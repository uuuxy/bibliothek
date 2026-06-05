package inventur

import (
	"context"
	"fmt"
	"testing"
	"github.com/pashagolub/pgxmock/v4"
)

func TestGetActiveSubjects(t *testing.T) {
	t.Run("successful query", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mock.Close()

		repo := &BookRepository{db: mock}

		mock.ExpectQuery("SELECT id, name, is_active FROM subjects WHERE is_active = true ORDER BY name ASC").
			WillReturnRows(pgxmock.NewRows([]string{"id", "name", "is_active"}).
				AddRow(1, "Mathematik", true).
				AddRow(2, "Deutsch", true))

		subjects, err := repo.GetActiveSubjects(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(subjects) != 2 {
			t.Errorf("expected 2 subjects, got %d", len(subjects))
		}

		if len(subjects) > 0 && subjects[0].Name != "Mathematik" {
			t.Errorf("expected Mathematik, got %s", subjects[0].Name)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mock.Close()

		repo := &BookRepository{db: mock}

		mock.ExpectQuery("SELECT id, name, is_active FROM subjects WHERE is_active = true ORDER BY name ASC").
			WillReturnError(fmt.Errorf("db connection error"))

		subjects, err := repo.GetActiveSubjects(context.Background())
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		if subjects != nil {
			t.Errorf("expected nil subjects, got %v", subjects)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("scan error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer mock.Close()

		repo := &BookRepository{db: mock}

		mock.ExpectQuery("SELECT id, name, is_active FROM subjects WHERE is_active = true ORDER BY name ASC").
			WillReturnRows(pgxmock.NewRows([]string{"id", "name", "is_active"}).
				AddRow("invalid_id_type", "Mathematik", true))

		subjects, err := repo.GetActiveSubjects(context.Background())
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		if subjects != nil {
			t.Errorf("expected nil subjects, got %v", subjects)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
