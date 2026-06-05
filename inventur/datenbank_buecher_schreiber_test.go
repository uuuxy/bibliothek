package inventur

import (
	"context"
	"os"
	"testing"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, *BookRepository) {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		t.Skip("DATABASE_URL is not set. Skipping database tests.")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	return pool, &BookRepository{db: pool}
}

func TestUpsertBooksBatch(t *testing.T) {
	pool, repo := setupTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	t.Run("EmptyBatch", func(t *testing.T) {
		affected, err := repo.UpsertBooksBatch(ctx, []Book{})
		if err != nil {
			t.Errorf("expected no error for empty batch, got %v", err)
		}
		if affected != 0 {
			t.Errorf("expected 0 rows affected, got %d", affected)
		}
	})

	t.Run("InsertNewBooks", func(t *testing.T) {
		// Clean up database before
		_, err := pool.Exec(ctx, "TRUNCATE buecher_titel CASCADE")
		if err != nil {
			t.Fatalf("failed to truncate table: %v", err)
		}

		books := []Book{
			{ISBN: "978-1-1111-1111-1", Title: "Book 1", Author: "Author 1", Subject: "Math", GradeLevel: 5, Track: "A", Stock: 2, Medientyp: "Buch"},
			{ISBN: "978-2-2222-2222-2", Title: "Book 2", Author: "Author 2", Subject: "Biology", GradeLevel: 8, Track: "B", Stock: 5, Medientyp: "Buch"},
		}

		affected, err := repo.UpsertBooksBatch(ctx, books)
		if err != nil {
			t.Fatalf("UpsertBooksBatch failed: %v", err)
		}

		// Affected rows might be equal to length of batch on insert
		if affected != int64(len(books)) {
			t.Errorf("expected %d rows affected, got %d", len(books), affected)
		}

		// Verify database
		var count int
		err = pool.QueryRow(ctx, "SELECT count(*) FROM buecher_titel WHERE isbn IN ('978-1-1111-1111-1', '978-2-2222-2222-2')").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query database: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 books in database, found %d", count)
		}
	})

	t.Run("UpdateExistingBooks", func(t *testing.T) {
		// Clean up database before
		_, err := pool.Exec(ctx, "TRUNCATE buecher_titel CASCADE")
		if err != nil {
			t.Fatalf("failed to truncate table: %v", err)
		}

		// Setup initial state: Insert Book 1 with stock 2
		initialBooks := []Book{
			{ISBN: "978-1-1111-1111-1", Title: "Book 1", Author: "Author 1", Subject: "Math", GradeLevel: 5, Track: "A", Stock: 2, Medientyp: "Buch"},
		}
		_, err = repo.UpsertBooksBatch(ctx, initialBooks)
		if err != nil {
			t.Fatalf("Initial UpsertBooksBatch failed: %v", err)
		}

		// We insert Book 1 again with stock 3, and new title. Stock should become 5.
		// We also insert a new Book 3.
		books := []Book{
			{ISBN: "978-1-1111-1111-1", Title: "Book 1 - Updated", Author: "Author 1", Subject: "Math", GradeLevel: 5, Track: "A", Stock: 3, Medientyp: "Buch"},
			{ISBN: "978-3-3333-3333-3", Title: "Book 3", Author: "Author 3", Subject: "Physics", GradeLevel: 10, Track: "C", Stock: 1, Medientyp: "Buch"},
		}

		_, err = repo.UpsertBooksBatch(ctx, books)
		if err != nil {
			t.Fatalf("UpsertBooksBatch failed: %v", err)
		}

		// Affected rows could be different on upsert in Postgres (insert + update = 3 rows affected sometimes, but lets just check it didn't fail)

		// Verify Book 1 was updated
		var title string
		var stock int
		err = pool.QueryRow(ctx, "SELECT titel, stock FROM buecher_titel WHERE isbn = '978-1-1111-1111-1'").Scan(&title, &stock)
		if err != nil {
			t.Fatalf("failed to query database for Book 1: %v", err)
		}
		if title != "Book 1 - Updated" {
			t.Errorf("expected title to be updated to 'Book 1 - Updated', got '%s'", title)
		}
		if stock != 5 {
			t.Errorf("expected stock to be added (2 + 3 = 5), got %d", stock)
		}

		// Verify Book 3 was inserted
		var count int
		err = pool.QueryRow(ctx, "SELECT count(*) FROM buecher_titel WHERE isbn = '978-3-3333-3333-3'").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query database for Book 3: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 book for Book 3 in database, found %d", count)
		}
	})

	// Clean up database after
	defer pool.Exec(ctx, "TRUNCATE buecher_titel CASCADE")
}
