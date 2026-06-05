package inventur

import (
	"bibliothek/db"
	"context"
	"fmt"
	"os"
	"testing"
)

// setupBenchDB returns a connected pgxpool.Pool and the BookRepository.
func setupBenchDB(b *testing.B) (db.PgxPoolIface, *BookRepository) {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		b.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		b.Fatalf("failed to connect: %v", err)
	}

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS books (
			id BIGSERIAL PRIMARY KEY,
			isbn TEXT UNIQUE,
			title TEXT,
			author TEXT,
			cover_url TEXT,
			subject TEXT,
			grade_level SMALLINT,
			track TEXT,
			stock INT,
			last_counted DATE,
			sort_order SERIAL
		)
	`)
	if err != nil {
		b.Fatalf("failed to create table: %v", err)
	}

	_, err = pool.Exec(ctx, "TRUNCATE books")
	if err != nil {
		b.Fatalf("failed to truncate table: %v", err)
	}

	return pool, &BookRepository{db: pool}
}

func BenchmarkImportSequential(b *testing.B) {
	pool, repo := setupBenchDB(b)
	defer pool.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		book := Book{
			ISBN:       fmt.Sprintf("978-3-16-%d", i),
			Title:      "Test",
			Author:     "Test",
			Subject:    "Test",
			GradeLevel: 5,
			Stock:      1,
		}
		repo.UpsertBook(ctx, book)
	}
}

func BenchmarkImportBatch(b *testing.B) {
	pool, repo := setupBenchDB(b)
	defer pool.Close()
	ctx := context.Background()

	// Pre-allocate the books to isolate just the db performance
	books := make([]Book, b.N)
	for i := 0; i < b.N; i++ {
		books[i] = Book{
			ISBN:       fmt.Sprintf("978-3-16-%d", i),
			Title:      "Test",
			Author:     "Test",
			Subject:    "Test",
			GradeLevel: 5,
			Stock:      1,
		}
	}

	b.ResetTimer()
	if len(books) > 0 {
		_, err := repo.UpsertBooksBatch(ctx, books)
		if err != nil {
			b.Fatalf("Batch failed: %v", err)
		}
	}
}
