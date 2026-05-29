package inventur

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreateBook inserts a new book record.
func (repo *BookRepository) CreateBook(ctx context.Context, book Book) (string, error) {
	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9::text, '')::date)
		RETURNING id`

	var id string
	err := repo.db.QueryRow(
		ctx,
		query,
		book.ISBN,
		book.Title,
		book.Author,
		book.CoverURL,
		book.Subject,
		book.GradeLevel,
		book.Track,
		book.Stock,
		book.LastCounted,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("buch konnte nicht erstellt werden: %w", handleDbError(err))
	}

	return id, nil
}

// UpsertBooksBatch handles batch upserting book records.
func (repo *BookRepository) UpsertBooksBatch(ctx context.Context, books []Book) (int64, error) {
	if len(books) == 0 {
		return 0, nil
	}

	isbns := make([]string, len(books))
	titles := make([]string, len(books))
	authors := make([]string, len(books))
	coverUrls := make([]string, len(books))
	subjects := make([]string, len(books))
	grades := make([]int16, len(books))
	tracks := make([]string, len(books))
	stocks := make([]int32, len(books))
	lastCounteds := make([]*string, len(books))

	for i, b := range books {
		isbns[i] = b.ISBN
		titles[i] = b.Title
		authors[i] = b.Author
		coverUrls[i] = b.CoverURL
		subjects[i] = b.Subject
		grades[i] = b.GradeLevel
		tracks[i] = b.Track
		stocks[i] = int32(b.Stock)
		lastCounteds[i] = b.LastCounted
	}

	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted)
		SELECT t.isbn, t.titel, t.autor, t.cover_url, t.subject, t.grade_level, t.track, t.stock, NULLIF(t.last_counted_text, '')::date
		FROM UNNEST($1::text[], $2::text[], $3::text[], $4::text[], $5::text[], $6::smallint[], $7::text[], $8::int[], $9::text[])
		AS t(isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted_text)
		ON CONFLICT (isbn) DO UPDATE SET
			titel = EXCLUDED.titel,
			autor = EXCLUDED.autor,
			cover_url = EXCLUDED.cover_url,
			subject = EXCLUDED.subject,
			grade_level = EXCLUDED.grade_level,
			track = EXCLUDED.track,
			stock = buecher_titel.stock + EXCLUDED.stock,
			last_counted = EXCLUDED.last_counted
	`

	cmdTag, err := repo.db.Exec(
		ctx,
		query,
		isbns,
		titles,
		authors,
		coverUrls,
		subjects,
		grades,
		tracks,
		stocks,
		lastCounteds,
	)
	if err != nil {
		return 0, fmt.Errorf("bücher konnten nicht im batch importiert werden: %w", err)
	}

	return cmdTag.RowsAffected(), nil
}

// UpsertBook inserts or updates a book record.
func (repo *BookRepository) UpsertBook(ctx context.Context, book Book) (string, error) {
	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9::text, '')::date)
		ON CONFLICT (isbn) DO UPDATE SET
			titel = EXCLUDED.titel,
			autor = EXCLUDED.autor,
			subject = EXCLUDED.subject,
			grade_level = EXCLUDED.grade_level,
			track = EXCLUDED.track,
			stock = buecher_titel.stock + EXCLUDED.stock,
			last_counted = EXCLUDED.last_counted
		RETURNING id`

	var id string
	err := repo.db.QueryRow(
		ctx,
		query,
		book.ISBN,
		book.Title,
		book.Author,
		book.CoverURL,
		book.Subject,
		book.GradeLevel,
		book.Track,
		book.Stock,
		book.LastCounted,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("buch konnte nicht importiert werden: %w", err)
	}

	return id, nil
}

// UpdateStock modifies the stock level of a book.
func (repo *BookRepository) UpdateStock(ctx context.Context, id string, stock int) error {
	query := `UPDATE buecher_titel SET stock = $1 WHERE id = $2`
	result, err := repo.db.Exec(ctx, query, stock, id)
	if err != nil {
		return fmt.Errorf("bestand konnte nicht aktualisiert werden: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBookNotFound
	}

	return nil
}

// UpdateBook updates metadata fields of a book.
func (repo *BookRepository) UpdateBook(ctx context.Context, id string, book Book) error {
	query := `
		UPDATE buecher_titel
		SET isbn = $1,
			titel = $2,
			autor = $3,
			cover_url = $4,
			subject = $5,
			grade_level = $6,
			track = $7,
			stock = $8,
			last_counted = NULLIF($9::text, '')::date
		WHERE id = $10`

	result, err := repo.db.Exec(
		ctx,
		query,
		book.ISBN,
		book.Title,
		book.Author,
		book.CoverURL,
		book.Subject,
		book.GradeLevel,
		book.Track,
		book.Stock,
		book.LastCounted,
		id,
	)
	if err != nil {
		return fmt.Errorf("buch konnte nicht aktualisiert werden: %w", handleDbError(err))
	}

	if result.RowsAffected() == 0 {
		return ErrBookNotFound
	}

	return nil
}

// DeleteBooks deletes multiple book records.
func (repo *BookRepository) DeleteBooks(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	coverRows, err := repo.db.Query(ctx, "SELECT cover_url FROM buecher_titel WHERE id = ANY($1::uuid[]) AND cover_url LIKE '/uploads/%'", ids)
	if err != nil {
		return fmt.Errorf("cover-dateien konnten nicht ermittelt werden: %w", err)
	}
	localCovers := make([]string, 0)
	for coverRows.Next() {
		var coverURL string
		if scanErr := coverRows.Scan(&coverURL); scanErr != nil {
			coverRows.Close()
			return fmt.Errorf("cover-pfade konnten nicht gelesen werden: %w", scanErr)
		}
		localCovers = append(localCovers, coverURL)
	}
	coverRows.Close()
	if rowsErr := coverRows.Err(); rowsErr != nil {
		return fmt.Errorf("cover-pfade konnten nicht iteriert werden: %w", rowsErr)
	}

	query := `DELETE FROM buecher_titel WHERE id = ANY($1::uuid[])`
	result, err := repo.db.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("bücher konnten nicht gelöscht werden: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrBookNotFound
	}

	for _, coverURL := range localCovers {
		if !strings.HasPrefix(coverURL, "/uploads/") {
			continue
		}
		name := filepath.Base(coverURL)
		if name == "" || name == "." || name == "/" {
			continue
		}
		_ = os.Remove(filepath.Join("uploads", name))
	}

	return nil
}
