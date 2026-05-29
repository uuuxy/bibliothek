package inventur

import (
	"context"
	"fmt"
)

// ListBooks lists books matching subject, grade level, and text query.
func (repo *BookRepository) ListBooks(ctx context.Context, subject string, grade *int16, searchQuery string) ([]Book, error) {
	query := `
		SELECT id, COALESCE(isbn, '') AS isbn, titel AS title, COALESCE(autor, '') AS author, COALESCE(cover_url, '') AS cover_url, COALESCE(subject, '') AS subject, COALESCE(grade_level, 0) AS grade_level, COALESCE(track, '') AS track, stock, TO_CHAR(last_counted, 'YYYY-MM-DD') as last_counted, sort_order
		FROM buecher_titel
		WHERE ($1 = '' OR subject = $1)
		  AND ($2::smallint IS NULL OR grade_level = $2)
		  AND ($3 = '' OR titel ILIKE '%' || $3 || '%' OR autor ILIKE '%' || $3 || '%' OR isbn ILIKE '%' || $3 || '%' OR subject ILIKE '%' || $3 || '%' OR CAST(id AS TEXT) ILIKE '%' || $3 || '%')
		ORDER BY sort_order ASC, titel ASC`

	rows, err := repo.db.Query(ctx, query, subject, grade, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("bücher konnten nicht geladen werden: %w", err)
	}
	defer rows.Close()

	books := make([]Book, 0)
	for rows.Next() {
		var book Book
		err := rows.Scan(
			&book.ID,
			&book.ISBN,
			&book.Title,
			&book.Author,
			&book.CoverURL,
			&book.Subject,
			&book.GradeLevel,
			&book.Track,
			&book.Stock,
			&book.LastCounted,
			&book.SortOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("daten konnten nicht gelesen werden: %w", err)
		}
		books = append(books, book)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fehler beim iterieren: %w", err)
	}

	return books, nil
}

// ListExternalCoverBooks lists books having external cover URLs.
func (repo *BookRepository) ListExternalCoverBooks(ctx context.Context, limit int) ([]Book, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, COALESCE(isbn, '') AS isbn, titel AS title, COALESCE(cover_url, '') AS cover_url
		FROM buecher_titel
		WHERE cover_url LIKE 'http%'
		ORDER BY id ASC
		LIMIT $1`

	rows, err := repo.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("bücher mit externen covern konnten nicht geladen werden: %w", err)
	}
	defer rows.Close()

	books := make([]Book, 0)
	for rows.Next() {
		var book Book
		if scanErr := rows.Scan(&book.ID, &book.ISBN, &book.Title, &book.CoverURL); scanErr != nil {
			return nil, fmt.Errorf("bücher mit externen covern konnten nicht gelesen werden: %w", scanErr)
		}
		books = append(books, book)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("fehler beim iterieren externer cover-bücher: %w", rowsErr)
	}

	return books, nil
}

// ListBooksByIDs retrieves list of books for provided IDs.
func (repo *BookRepository) ListBooksByIDs(ctx context.Context, ids []string) ([]Book, error) {
	if len(ids) == 0 {
		return []Book{}, nil
	}

	query := `
		SELECT id, COALESCE(isbn, '') AS isbn, titel AS title, COALESCE(cover_url, '') AS cover_url
		FROM buecher_titel
		WHERE id = ANY($1::uuid[])
		ORDER BY id ASC`

	rows, err := repo.db.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("bücher nach ids konnten nicht geladen werden: %w", err)
	}
	defer rows.Close()

	books := make([]Book, 0)
	for rows.Next() {
		var book Book
		if scanErr := rows.Scan(&book.ID, &book.ISBN, &book.Title, &book.CoverURL); scanErr != nil {
			return nil, fmt.Errorf("bücher nach ids konnten nicht gelesen werden: %w", scanErr)
		}
		books = append(books, book)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("fehler beim iterieren der buch-ids: %w", rowsErr)
	}

	return books, nil
}
