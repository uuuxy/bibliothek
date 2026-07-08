package inventur

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestUpdateClassBooks_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM class_books WHERE class_name = \\$1").
		WithArgs("05A").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	mock.ExpectExec("DELETE FROM class_books WHERE class_name = ANY\\(\\$1\\)").
		WithArgs([]string{"05A", "05B"}).
		WillReturnResult(pgxmock.NewResult("DELETE", 2))

	mock.ExpectExec("INSERT INTO class_books").
		WithArgs([]string{"05A", "05A", "05B", "05B"}, []string{"b1", "b2", "b1", "b2"}).
		WillReturnResult(pgxmock.NewResult("INSERT", 4))

	mock.ExpectCommit()
	mock.ExpectRollback()

	err = repo.UpdateClassBooks(context.Background(), "05A", []string{"05A", "05B"}, []string{"b1", "b2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestUpdateClassBooks_TxBeginError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	mock.ExpectBegin().WillReturnError(errors.New("begin error"))

	err = repo.UpdateClassBooks(context.Background(), "05A", []string{"05B"}, []string{"b1"})
	if err == nil || !strings.Contains(err.Error(), "transaktion konnte nicht gestartet werden") {
		t.Errorf("expected begin error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestUpdateClassBooks_DeleteOldError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM class_books WHERE class_name = \\$1").
		WithArgs("05A").
		WillReturnError(errors.New("delete error"))
	mock.ExpectRollback()

	err = repo.UpdateClassBooks(context.Background(), "05A", []string{"05B"}, []string{"b1"})
	if err == nil || !strings.Contains(err.Error(), "alte zuweisungen konnten nicht gelöscht werden") {
		t.Errorf("expected delete error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestUpdateClassBooks_DeleteNewError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM class_books WHERE class_name = \\$1").
		WithArgs("05A").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec("DELETE FROM class_books WHERE class_name = ANY\\(\\$1\\)").
		WithArgs([]string{"05B"}).
		WillReturnError(errors.New("delete error"))
	mock.ExpectRollback()

	err = repo.UpdateClassBooks(context.Background(), "05A", []string{"05B"}, []string{"b1"})
	if err == nil || !strings.Contains(err.Error(), "vorhandene zuweisungen des neuen namens konnten nicht gelöscht werden") {
		t.Errorf("expected delete error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestUpdateClassBooks_InsertError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM class_books WHERE class_name = \\$1").
		WithArgs("05A").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec("DELETE FROM class_books WHERE class_name = ANY\\(\\$1\\)").
		WithArgs([]string{"05B"}).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec("INSERT INTO class_books").
		WithArgs([]string{"05B"}, []string{"b1"}).
		WillReturnError(errors.New("insert error"))
	mock.ExpectRollback()

	err = repo.UpdateClassBooks(context.Background(), "05A", []string{"05B"}, []string{"b1"})
	if err == nil || !strings.Contains(err.Error(), "neue zuweisung konnte nicht gespeichert werden") {
		t.Errorf("expected insert error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestUpdateClassBooks_CommitError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM class_books WHERE class_name = \\$1").
		WithArgs("05A").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec("DELETE FROM class_books WHERE class_name = ANY\\(\\$1\\)").
		WithArgs([]string{"05B"}).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec("INSERT INTO class_books").
		WithArgs([]string{"05B"}, []string{"b1"}).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))
	mock.ExpectRollback()

	err = repo.UpdateClassBooks(context.Background(), "05A", []string{"05B"}, []string{"b1"})
	if err == nil || !strings.Contains(err.Error(), "transaktion konnte nicht abgeschlossen werden") {
		t.Errorf("expected commit error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestUpdateClassBooks_SuccessNoOldClass(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM class_books WHERE class_name = ANY\\(\\$1\\)").
		WithArgs([]string{"05B"}).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec("INSERT INTO class_books").
		WithArgs([]string{"05B"}, []string{"b1"}).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()
	mock.ExpectRollback()

	err = repo.UpdateClassBooks(context.Background(), "", []string{"05B"}, []string{"b1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
