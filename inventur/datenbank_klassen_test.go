package inventur

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestAddBooksToClasses(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	t.Run("empty classNames", func(t *testing.T) {
		err := repo.AddBooksToClasses(ctx, []string{}, []string{"b1"})
		if err != nil {
			t.Errorf("expected no error for empty classes, got %v", err)
		}
	})

	t.Run("empty bookIDs", func(t *testing.T) {
		err := repo.AddBooksToClasses(ctx, []string{"c1"}, []string{})
		if err != nil {
			t.Errorf("expected no error for empty books, got %v", err)
		}
	})

	t.Run("successful insert", func(t *testing.T) {
		classes := []string{"05A", "05B"}
		books := []string{"book1", "book2"}

		expectedClasses := []string{"05A", "05A", "05B", "05B"}
		expectedBooks := []string{"book1", "book2", "book1", "book2"}

		mock.ExpectExec("INSERT INTO class_books").
			WithArgs(expectedClasses, expectedBooks).
			WillReturnResult(pgxmock.NewResult("INSERT", 4))

		err := repo.AddBooksToClasses(ctx, classes, books)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		classes := []string{"05A"}
		books := []string{"book1"}

		expectedClasses := []string{"05A"}
		expectedBooks := []string{"book1"}

		mock.ExpectExec("INSERT INTO class_books").
			WithArgs(expectedClasses, expectedBooks).
			WillReturnError(errors.New("db insert error"))

		err := repo.AddBooksToClasses(ctx, classes, books)
		if err == nil {
			t.Errorf("expected error, got nil")
		} else if err.Error() != "fehler beim hinzufügen der bücher zu den klassen: db insert error" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
