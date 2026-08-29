package inventur

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

var errTest = errors.New("test error")

func TestNormalizeAllClasses(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	mock.ExpectBegin()

	// Step 1: Delete duplicate before space removal
	mock.ExpectExec("DELETE FROM class_books cb1 WHERE class_name LIKE '% %' AND EXISTS").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	// Step 1: Update to remove spaces
	mock.ExpectExec("UPDATE class_books SET class_name = REPLACE").
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))

	// Step 2: Delete duplicate before leading zero addition
	mock.ExpectExec("DELETE FROM class_books cb1 WHERE \\(class_name ~ '\\^\\[1-9\\]\\[\\^0-9\\]' OR class_name ~ '\\^\\[1-9\\]\\$'\\) AND EXISTS").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	// Step 2: Update to add leading zero
	mock.ExpectExec("UPDATE class_books SET class_name = '0' \\|\\| class_name WHERE class_name ~ '\\^\\[1-9\\]\\[\\^0-9\\]' OR class_name ~ '\\^\\[1-9\\]\\$'").
		WillReturnResult(pgxmock.NewResult("UPDATE", 3))

	mock.ExpectCommit()

	err = repo.NormalizeAllClasses(ctx)
	if err != nil {
		t.Errorf("NormalizeAllClasses failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestNormalizeAllClasses_TxError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	mock.ExpectBegin().WillReturnError(errTest)

	err = repo.NormalizeAllClasses(ctx)
	if err == nil || err.Error() != "transaktion konnte nicht gestartet werden: test error" {
		t.Errorf("expected error starting transaction, got: %v", err)
	}
}

func TestNormalizeAllClasses_Step1DeleteError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM class_books cb1 WHERE class_name LIKE '% %' AND EXISTS").
		WillReturnError(errTest)
	mock.ExpectRollback()

	err = repo.NormalizeAllClasses(ctx)
	if err == nil || err.Error() != "fehler beim bereinigen doppelter klassennamen vor leerzeichen-entfernung: test error" {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestNormalizeAllClasses_Step1UpdateError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM class_books cb1 WHERE class_name LIKE '% %' AND EXISTS").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec("UPDATE class_books SET class_name = REPLACE").
		WillReturnError(errTest)
	mock.ExpectRollback()

	err = repo.NormalizeAllClasses(ctx)
	if err == nil || err.Error() != "fehler beim entfernen von leerzeichen in klassennamen: test error" {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestNormalizeAllClasses_Step2DeleteError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM class_books cb1 WHERE class_name LIKE '% %' AND EXISTS").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec("UPDATE class_books SET class_name = REPLACE").
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))
	mock.ExpectExec("DELETE FROM class_books cb1 WHERE \\(class_name ~ '\\^\\[1-9\\]\\[\\^0-9\\]' OR class_name ~ '\\^\\[1-9\\]\\$'\\) AND EXISTS").
		WillReturnError(errTest)
	mock.ExpectRollback()

	err = repo.NormalizeAllClasses(ctx)
	if err == nil || err.Error() != "fehler beim bereinigen doppelter klassennamen: test error" {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestNormalizeAllClasses_Step2UpdateError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM class_books cb1 WHERE class_name LIKE '% %' AND EXISTS").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec("UPDATE class_books SET class_name = REPLACE").
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))
	mock.ExpectExec("DELETE FROM class_books cb1 WHERE \\(class_name ~ '\\^\\[1-9\\]\\[\\^0-9\\]' OR class_name ~ '\\^\\[1-9\\]\\$'\\) AND EXISTS").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec("UPDATE class_books SET class_name = '0' \\|\\| class_name WHERE class_name ~ '\\^\\[1-9\\]\\[\\^0-9\\]' OR class_name ~ '\\^\\[1-9\\]\\$'").
		WillReturnError(errTest)
	mock.ExpectRollback()

	err = repo.NormalizeAllClasses(ctx)
	if err == nil || err.Error() != "fehler beim normalisieren der klassennamen: test error" {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestDeleteClassGroup(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	tests := []struct {
		name       string
		classNames []string
		mockSetup  func()
		wantErr    bool
	}{
		{
			name:       "success single",
			classNames: []string{"5A"},
			mockSetup: func() {
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = ANY\(\$1\)$`).
					WithArgs([]string{"5A"}).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name:       "success multiple",
			classNames: []string{"5A", "5B"},
			mockSetup: func() {
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = ANY\(\$1\)$`).
					WithArgs([]string{"5A", "5B"}).
					WillReturnResult(pgxmock.NewResult("DELETE", 2))
			},
			wantErr: false,
		},
		{
			name:       "success empty",
			classNames: []string{},
			mockSetup: func() {
				// No DB call expected
			},
			wantErr: false,
		},
		{
			name:       "db error",
			classNames: []string{"10B"},
			mockSetup: func() {
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = ANY\(\$1\)$`).
					WithArgs([]string{"10B"}).
					WillReturnError(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := repo.DeleteClassGroup(context.Background(), tt.classNames)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteClassGroup() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet mock expectations: %v", err)
			}
		})
	}
}

func TestGetClassGroups(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	tests := []struct {
		name       string
		branch     string
		sortOrder  string
		mockSetup  func()
		wantErr    bool
		wantGroups int
	}{
		{
			name:      "success default (asc, empty branch)",
			branch:    "",
			sortOrder: "",
			mockSetup: func() {
				rows := pgxmock.NewRows([]string{
					"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt",
				}).
					AddRow("5A", "550e8400-e29b-41d4-a716-446655440000", "Buch 1", "Mathe", "G", "url1", "123", 5, 10).
					AddRow("5A", "550e8400-e29b-41d4-a716-446655440001", "Buch 2", "Deutsch", "G", "url2", "456", 2, 5).
					AddRow("6B", "550e8400-e29b-41d4-a716-446655440002", "Buch 1", "Mathe", "R", "url1", "123", 4, 8)

				mock.ExpectQuery(`(?s)SELECT.*cb\.class_name, b\.id, b\.titel AS title.*FROM class_books cb.*ORDER BY.*CASE.*ASC, b\.titel ASC`).
					WithArgs("").
					WillReturnRows(rows)
			},
			wantErr:    false,
			wantGroups: 2, // 5A and 6B
		},
		{
			name:      "success branch and desc",
			branch:    "F",
			sortOrder: "desc",
			mockSetup: func() {
				rows := pgxmock.NewRows([]string{
					"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt",
				}).
					AddRow("10F", "550e8400-e29b-41d4-a716-446655440003", "Buch 3", "Englisch", "F", "url3", "789", 1, 1)

				mock.ExpectQuery(`(?s)SELECT.*cb\.class_name, b\.id, b\.titel AS title.*FROM class_books cb.*ORDER BY.*CAST.*DESC, cb\.class_name DESC, b\.titel ASC`).
					WithArgs("F").
					WillReturnRows(rows)
			},
			wantErr:    false,
			wantGroups: 1, // 10F
		},
		{
			name:      "query error",
			branch:    "",
			sortOrder: "",
			mockSetup: func() {
				mock.ExpectQuery(`(?s)SELECT.*`).
					WithArgs("").
					WillReturnError(fmt.Errorf("db error"))
			},
			wantErr:    true,
			wantGroups: 0,
		},
		{
			name:      "scan error",
			branch:    "",
			sortOrder: "",
			mockSetup: func() {
				// Invalid type for gesamt (string instead of int) to force scan error
				rows := pgxmock.NewRows([]string{
					"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt",
				}).
					AddRow("5A", "550e8400-e29b-41d4-a716-446655440000", "Buch 1", "Mathe", "G", "url1", "123", 5, "invalid")

				mock.ExpectQuery(`(?s)SELECT.*`).
					WithArgs("").
					WillReturnRows(rows)
			},
			wantErr:    true,
			wantGroups: 0,
		},
		{
			name:      "rows error",
			branch:    "",
			sortOrder: "",
			mockSetup: func() {
				rows := pgxmock.NewRows([]string{
					"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "verfuegbar", "gesamt",
				}).
					AddRow("5A", "550e8400-e29b-41d4-a716-446655440000", "Buch 1", "Mathe", "G", "url1", "123", 5, 10).
					RowError(0, fmt.Errorf("row error"))

				mock.ExpectQuery(`(?s)SELECT.*`).
					WithArgs("").
					WillReturnRows(rows)
			},
			wantErr:    true,
			wantGroups: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			groups, err := repo.GetClassGroups(ctx, tt.branch, tt.sortOrder)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClassGroups() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(groups) != tt.wantGroups {
				t.Errorf("GetClassGroups() got %d groups, want %d", len(groups), tt.wantGroups)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet mock expectations: %v", err)
			}
		})
	}
}
