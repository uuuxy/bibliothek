package inventur

import (
	"context"
	"fmt"
	"testing"

	"bibliothek/db"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetClassGroups(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	database := &db.Database{Pool: mock}
	repo := NewBookRepository(database.Pool)

	// Test case 1: Successful retrieval without branch
	t.Run("SuccessWithoutBranch", func(t *testing.T) {
		mock.ExpectQuery(`SELECT(.*)FROM class_books cb(.*)JOIN buecher_titel b ON cb.book_id = b.id`).
			WithArgs(""). // branch is empty string
			WillReturnRows(pgxmock.NewRows([]string{"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "stock", "verfuegbar", "gesamt"}).
				AddRow("10A", "book1", "Math 10", "Math", "A", "cover1.jpg", "1234567890", 10, 8, 10).
				AddRow("10A", "book2", "History 10", "History", "", "cover2.jpg", "0987654321", 5, 5, 5).
				AddRow("10B", "book1", "Math 10", "Math", "B", "cover1.jpg", "1234567890", 10, 8, 10))

		groups, err := repo.GetClassGroups(context.Background(), "", "asc")

		assert.NoError(t, err)
		assert.Len(t, groups, 2)

		assert.Equal(t, "10A", groups[0].ClassName)
		assert.Len(t, groups[0].Books, 2)
		assert.Equal(t, "Math 10", groups[0].Books[0].Title)
		assert.Equal(t, "History 10", groups[0].Books[1].Title)

		assert.Equal(t, "10B", groups[1].ClassName)
		assert.Len(t, groups[1].Books, 1)
		assert.Equal(t, "Math 10", groups[1].Books[0].Title)

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	// Test case 2: Successful retrieval with branch and descending order
	t.Run("SuccessWithBranchDesc", func(t *testing.T) {
		mock.ExpectQuery(`SELECT(.*)FROM class_books cb(.*)WHERE(.*)ILIKE(.*)ORDER BY(.*)DESC`).
			WithArgs("MainBranch"). // branch is "MainBranch"
			WillReturnRows(pgxmock.NewRows([]string{"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "stock", "verfuegbar", "gesamt"}).
				AddRow("11A", "book3", "Science 11", "Science", "", "cover3.jpg", "1111111111", 15, 10, 15))

		groups, err := repo.GetClassGroups(context.Background(), "MainBranch", "desc")

		assert.NoError(t, err)
		assert.Len(t, groups, 1)

		assert.Equal(t, "11A", groups[0].ClassName)
		assert.Len(t, groups[0].Books, 1)
		assert.Equal(t, "Science 11", groups[0].Books[0].Title)

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	// Test case 3: Query error
	t.Run("QueryError", func(t *testing.T) {
		mock.ExpectQuery(`SELECT(.*)FROM class_books cb(.*)`).
			WithArgs("").
			WillReturnError(fmt.Errorf("db error"))

		groups, err := repo.GetClassGroups(context.Background(), "", "asc")

		assert.Error(t, err)
		assert.Nil(t, groups)
		assert.Contains(t, err.Error(), "klassen-bücher konnten nicht geladen werden")

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	// Test case 4: Scan error (mocking row structure mismatch)
	t.Run("ScanError", func(t *testing.T) {
		mock.ExpectQuery(`SELECT(.*)FROM class_books cb(.*)`).
			WithArgs("").
			WillReturnRows(pgxmock.NewRows([]string{"class_name"}).
				AddRow("10A")) // Only one column instead of all 10

		groups, err := repo.GetClassGroups(context.Background(), "", "asc")

		assert.Error(t, err)
		assert.Nil(t, groups)
		assert.Contains(t, err.Error(), "daten konnten nicht gelesen werden")

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	// Test case 5: Empty result
	t.Run("EmptyResult", func(t *testing.T) {
		mock.ExpectQuery(`SELECT(.*)FROM class_books cb(.*)`).
			WithArgs("EmptyBranch").
			WillReturnRows(pgxmock.NewRows([]string{"class_name", "id", "title", "subject", "track", "cover_url", "isbn", "stock", "verfuegbar", "gesamt"}))

		groups, err := repo.GetClassGroups(context.Background(), "EmptyBranch", "asc")

		assert.NoError(t, err)
		assert.Len(t, groups, 0) // Expect 0

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
