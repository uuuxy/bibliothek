package inventur

import (
	"bibliothek/db"
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// setupTestDB returns a connected pgxpool.Pool and the BookRepository for testing.
// It also creates the necessary tables for the tests.
func setupTestDB(t *testing.T) (db.PgxPoolIface, *BookRepository) {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS buecher_titel (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			titel VARCHAR(255) NOT NULL,
			untertitel VARCHAR(255),
			autor VARCHAR(255),
			isbn VARCHAR(20) UNIQUE,
			verlag VARCHAR(255),
			erscheinungsjahr INTEGER,
			beschreibung TEXT,
			meldebestand INTEGER NOT NULL DEFAULT 5,
			cover_url VARCHAR(512),
			subject VARCHAR(100),
			grade_level SMALLINT,
			track VARCHAR(100),
			stock INTEGER NOT NULL DEFAULT 0,
			last_counted DATE,
			sort_order SERIAL,
			erstellt_am TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			aktualisiert_am TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			medientyp VARCHAR(50) DEFAULT 'Buch',
			erweiterte_eigenschaften JSONB DEFAULT '{}'::jsonb
		)
	`)
	if err != nil {
		t.Fatalf("failed to create buecher_titel table: %v", err)
	}

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS class_books (
			class_name VARCHAR(50) NOT NULL,
			book_id UUID NOT NULL REFERENCES buecher_titel(id) ON DELETE CASCADE,
			PRIMARY KEY (class_name, book_id)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create class_books table: %v", err)
	}

	return pool, &BookRepository{db: pool}
}

func TestBookRepository_DeleteClassGroup(t *testing.T) {
	pool, repo := setupTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	tests := []struct {
		name          string
		className     string
		setupFn       func(t *testing.T)
		expectedError bool
		checkFn       func(t *testing.T)
	}{
		{
			name:      "Delete existing class group",
			className: "5a",
			setupFn: func(t *testing.T) {
				_, err := pool.Exec(ctx, "TRUNCATE class_books, buecher_titel CASCADE")
				if err != nil {
					t.Fatalf("failed to truncate tables: %v", err)
				}
				var book1ID, book2ID string
				err = pool.QueryRow(ctx, "INSERT INTO buecher_titel (titel, isbn) VALUES ('Book 1', '1234567890') RETURNING id").Scan(&book1ID)
				if err != nil {
					t.Fatalf("failed to insert book 1: %v", err)
				}
				err = pool.QueryRow(ctx, "INSERT INTO buecher_titel (titel, isbn) VALUES ('Book 2', '0987654321') RETURNING id").Scan(&book2ID)
				if err != nil {
					t.Fatalf("failed to insert book 2: %v", err)
				}
				_, err = pool.Exec(ctx, "INSERT INTO class_books (class_name, book_id) VALUES ('5a', $1)", book1ID)
				if err != nil {
					t.Fatalf("failed to assign book 1 to 5a: %v", err)
				}
				_, err = pool.Exec(ctx, "INSERT INTO class_books (class_name, book_id) VALUES ('5a', $1)", book2ID)
				if err != nil {
					t.Fatalf("failed to assign book 2 to 5a: %v", err)
				}
				_, err = pool.Exec(ctx, "INSERT INTO class_books (class_name, book_id) VALUES ('5b', $1)", book1ID)
				if err != nil {
					t.Fatalf("failed to assign book 1 to 5b: %v", err)
				}
			},
			expectedError: false,
			checkFn: func(t *testing.T) {
				var count int
				err := pool.QueryRow(ctx, "SELECT count(*) FROM class_books WHERE class_name = '5a'").Scan(&count)
				if err != nil {
					t.Fatalf("failed to count 5a books: %v", err)
				}
				if count != 0 {
					t.Errorf("expected 0 books for 5a, got %d", count)
				}
				err = pool.QueryRow(ctx, "SELECT count(*) FROM class_books WHERE class_name = '5b'").Scan(&count)
				if err != nil {
					t.Fatalf("failed to count 5b books: %v", err)
				}
				if count != 1 {
					t.Errorf("expected 1 book for 5b, got %d", count)
				}
			},
		},
		{
			name:      "Delete non-existent class group",
			className: "99z",
			setupFn: func(t *testing.T) {
				_, err := pool.Exec(ctx, "TRUNCATE class_books, buecher_titel CASCADE")
				if err != nil {
					t.Fatalf("failed to truncate tables: %v", err)
				}
				var book1ID string
				err = pool.QueryRow(ctx, "INSERT INTO buecher_titel (titel, isbn) VALUES ('Book 1', '1234567890') RETURNING id").Scan(&book1ID)
				if err != nil {
					t.Fatalf("failed to insert book 1: %v", err)
				}
				_, err = pool.Exec(ctx, "INSERT INTO class_books (class_name, book_id) VALUES ('5b', $1)", book1ID)
				if err != nil {
					t.Fatalf("failed to assign book 1 to 5b: %v", err)
				}
			},
			expectedError: false,
			checkFn: func(t *testing.T) {
				var count int
				err := pool.QueryRow(ctx, "SELECT count(*) FROM class_books WHERE class_name = '5b'").Scan(&count)
				if err != nil {
					t.Fatalf("failed to count 5b books: %v", err)
				}
				if count != 1 {
					t.Errorf("expected 1 book for 5b, got %d", count)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn(t)
			}
			err := repo.DeleteClassGroup(ctx, tt.className)
			if (err != nil) != tt.expectedError {
				t.Errorf("DeleteClassGroup() error = %v, expectedError %v", err, tt.expectedError)
			}
			if tt.checkFn != nil {
				tt.checkFn(t)
			}
		})
	}
}
