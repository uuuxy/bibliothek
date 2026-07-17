package inventur

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestGetActiveSubjects(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		query := "SELECT id, name, is_active FROM subjects WHERE is_active = true ORDER BY name ASC"
		rows := pgxmock.NewRows([]string{"id", "name", "is_active"}).
			AddRow(1, "Math", true).
			AddRow(2, "Science", true)

		mock.ExpectQuery(query).WillReturnRows(rows)

		subjects, err := repo.GetActiveSubjects(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(subjects) != 2 {
			t.Fatalf("expected 2 subjects, got %d", len(subjects))
		}

		if subjects[0].ID != 1 || subjects[0].Name != "Math" || !subjects[0].IsActive {
			t.Errorf("unexpected subject: %+v", subjects[0])
		}

		if subjects[1].ID != 2 || subjects[1].Name != "Science" || !subjects[1].IsActive {
			t.Errorf("unexpected subject: %+v", subjects[1])
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestGetActiveSubjects_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	t.Run("Query Error", func(t *testing.T) {
		query := "SELECT id, name, is_active FROM subjects WHERE is_active = true ORDER BY name ASC"
		mock.ExpectQuery(query).WillReturnError(errors.New("db connection error"))

		subjects, err := repo.GetActiveSubjects(ctx)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if subjects != nil {
			t.Errorf("expected subjects to be nil, got %v", subjects)
		}

		if err.Error() != "fächer konnten nicht geladen werden: db connection error" {
			t.Errorf("unexpected error message: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestGetActiveSubjects_ScanError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	t.Run("Scan Error", func(t *testing.T) {
		query := "SELECT id, name, is_active FROM subjects WHERE is_active = true ORDER BY name ASC"
		// returning a string instead of an int for ID to cause a scan error
		rows := pgxmock.NewRows([]string{"id", "name", "is_active"}).
			AddRow("not-an-int", "Math", true)

		mock.ExpectQuery(query).WillReturnRows(rows)

		subjects, err := repo.GetActiveSubjects(ctx)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if subjects != nil {
			t.Errorf("expected subjects to be nil, got %v", subjects)
		}

		expectedErrMsgPrefix := "fehler beim lesen der fächer"
		if len(err.Error()) < len(expectedErrMsgPrefix) || err.Error()[:len(expectedErrMsgPrefix)] != expectedErrMsgPrefix {
			t.Errorf("unexpected error message: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestGetActiveSubjects_RowsError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	t.Run("Rows Error", func(t *testing.T) {
		query := "SELECT id, name, is_active FROM subjects WHERE is_active = true ORDER BY name ASC"
		rows := pgxmock.NewRows([]string{"id", "name", "is_active"}).
			AddRow(1, "Math", true).
			RowError(0, errors.New("rows iteration error"))

		mock.ExpectQuery(query).WillReturnRows(rows)

		subjects, err := repo.GetActiveSubjects(ctx)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if subjects != nil {
			t.Errorf("expected subjects to be nil, got %v", subjects)
		}

		expectedErrMsg := "fehler beim lesen der fächer: rows iteration error"
		if err.Error() != expectedErrMsg {
			t.Errorf("unexpected error message: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
