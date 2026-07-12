package inventur

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAllClasses_Regex(t *testing.T) {
	dbURL := os.Getenv("TEST_DB")
	if dbURL == "" {
		t.Skip("TEST_DB not set. Skipping real DB queries test.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	_, err = tx.Exec(ctx, `
		CREATE TEMP TABLE class_books (
			class_name TEXT NOT NULL,
			book_id UUID NOT NULL
		);
	`)
	require.NoError(t, err)

	bookID := "123e4567-e89b-12d3-a456-426614174000"
	bookID2 := "123e4567-e89b-12d3-a456-426614174001"

	_, err = tx.Exec(ctx, `
		INSERT INTO class_books (class_name, book_id) VALUES
		('5 A', $1),
		('5A', $1),
		(' 6 B ', $2),
		('7', $1),
		('07', $1),
		('8', $2),
		('10A', $1)
	`, bookID, bookID2)
	require.NoError(t, err)

	// Run exact queries from NormalizeAllClasses

	// 1. Delete dup spaces
	_, err = tx.Exec(ctx, `
		DELETE FROM class_books cb1
		WHERE class_name LIKE '% %'
		AND EXISTS (
			SELECT 1 FROM class_books cb2
			WHERE cb2.class_name = REPLACE(cb1.class_name, ' ', '')
			AND cb2.book_id = cb1.book_id
		)
	`)
	require.NoError(t, err)

	// 2. Remove spaces
	_, err = tx.Exec(ctx, `
		UPDATE class_books
		SET class_name = REPLACE(class_name, ' ', '')
		WHERE class_name LIKE '% %'
	`)
	require.NoError(t, err)

	// 3. Delete dup leading zero
	_, err = tx.Exec(ctx, `
		DELETE FROM class_books cb1
		WHERE (class_name ~ '^[1-9][^0-9]' OR class_name ~ '^[1-9]$')
		AND EXISTS (
			SELECT 1 FROM class_books cb2
			WHERE cb2.class_name = '0' || cb1.class_name
			AND cb2.book_id = cb1.book_id
		)
	`)
	require.NoError(t, err)

	// 4. Update leading zero
	_, err = tx.Exec(ctx, `
		UPDATE class_books
		SET class_name = '0' || class_name
		WHERE class_name ~ '^[1-9][^0-9]' OR class_name ~ '^[1-9]$'
	`)
	require.NoError(t, err)

	// Verify
	rows, err := tx.Query(ctx, "SELECT class_name, book_id FROM class_books ORDER BY class_name, book_id")
	require.NoError(t, err)
	defer rows.Close()

	type result struct {
		ClassName string
		BookID    string
	}
	var results []result
	for rows.Next() {
		var r result
		err := rows.Scan(&r.ClassName, &r.BookID)
		require.NoError(t, err)
		results = append(results, r)
	}
	require.NoError(t, rows.Err())

	expected := []result{
		{"05A", bookID},
		{"06B", bookID2},
		{"07", bookID},
		{"08", bookID2},
		{"10A", bookID},
	}

	assert.Equal(t, expected, results)
}
