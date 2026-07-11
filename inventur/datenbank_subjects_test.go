package inventur

import (
	"context"
	"fmt"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestGetActiveSubjects_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	query := "(?s)SELECT DISTINCT fach.*FROM buecher.*WHERE fach IS NOT NULL AND fach != ''.*ORDER BY fach ASC"
	rows := pgxmock.NewRows([]string{"fach"}).
		AddRow("Deutsch").
		AddRow("Mathematik")

	mock.ExpectQuery(query).WillReturnRows(rows)

	subjects, err := repo.GetActiveSubjects(ctx)
	if err != nil {
		t.Fatalf("GetActiveSubjects failed: %v", err)
	}

	if len(subjects) != 2 {
		t.Fatalf("expected 2 subjects, got %d", len(subjects))
	}
	if subjects[0].Name != "Deutsch" || subjects[0].ID != 1 || !subjects[0].IsActive {
		t.Errorf("unexpected subjects returned: %+v", subjects[0])
	}
	if subjects[1].Name != "Mathematik" || subjects[1].ID != 2 || !subjects[1].IsActive {
		t.Errorf("unexpected subjects returned: %+v", subjects[1])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetActiveSubjects_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	query := "(?s)SELECT DISTINCT fach.*FROM buecher.*WHERE fach IS NOT NULL AND fach != ''.*ORDER BY fach ASC"
	mock.ExpectQuery(query).WillReturnError(fmt.Errorf("db error"))

	subjects, err := repo.GetActiveSubjects(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if subjects != nil {
		t.Errorf("expected nil subjects on error, got %+v", subjects)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetActiveSubjects_ScanError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	query := "(?s)SELECT DISTINCT fach.*FROM buecher.*WHERE fach IS NOT NULL AND fach != ''.*ORDER BY fach ASC"
	// Returning extra column to cause scan error
	rows := pgxmock.NewRows([]string{"fach", "extra"}).
		AddRow("Deutsch", "extra_val")

	mock.ExpectQuery(query).WillReturnRows(rows)

	subjects, err := repo.GetActiveSubjects(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if subjects != nil {
		t.Errorf("expected nil subjects on error, got %+v", subjects)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetActiveSubjects_RowsError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	query := "(?s)SELECT DISTINCT fach.*FROM buecher.*WHERE fach IS NOT NULL AND fach != ''.*ORDER BY fach ASC"
	rows := pgxmock.NewRows([]string{"fach"}).
		AddRow("Mathematik").
		RowError(0, fmt.Errorf("row error"))

	mock.ExpectQuery(query).WillReturnRows(rows)

	subjects, err := repo.GetActiveSubjects(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if subjects != nil {
		t.Errorf("expected nil subjects on error, got %+v", subjects)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
