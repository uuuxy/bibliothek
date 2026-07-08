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
		name      string
		className string
		mockSetup func()
		wantErr   bool
	}{
		{
			name:      "success",
			className: "5A",
			mockSetup: func() {
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = \$1$`).
					WithArgs("5A").
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name:      "db error",
			className: "10B",
			mockSetup: func() {
				mock.ExpectExec(`^DELETE FROM class_books WHERE class_name = \$1$`).
					WithArgs("10B").
					WillReturnError(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := repo.DeleteClassGroup(context.Background(), tt.className)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteClassGroup() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet mock expectations: %v", err)
			}
		})
	}
}
