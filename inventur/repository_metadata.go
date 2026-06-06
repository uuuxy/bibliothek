package inventur

import (
	"context"
	"fmt"
)

func (repo *BookRepository) GetBookByID(ctx context.Context, id string) (*Book, error) {
	query := `
		SELECT id, COALESCE(isbn, '') AS isbn, titel AS title, COALESCE(autor, '') AS author, COALESCE(cover_url, '') AS cover_url, COALESCE(subject, '') AS subject, COALESCE(grade_level, 0) AS grade_level, COALESCE(track, '') AS track, stock, TO_CHAR(last_counted, 'YYYY-MM-DD') as last_counted, sort_order, COALESCE(medientyp, 'Buch') AS medientyp, COALESCE(jahrgang_von, 5) AS jahrgang_von, COALESCE(jahrgang_bis, 10) AS jahrgang_bis, erweiterte_eigenschaften
		FROM buecher_titel
		WHERE id = $1::uuid`

	var book Book
	err := repo.db.QueryRow(ctx, query, id).Scan(
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
		&book.Medientyp,
		&book.JahrgangVon,
		&book.JahrgangBis,
		&book.ErweiterteEigenschaften,
	)
	if err != nil {
		return nil, fmt.Errorf("buch nicht gefunden")
	}
	return &book, nil
}

func (repo *BookRepository) UpdateBookMetadata(ctx context.Context, id string, title, author, coverURL string) error {
	query := `
		UPDATE buecher_titel
		SET titel = COALESCE(NULLIF($1, ''), titel),
		    autor = COALESCE(NULLIF($2, ''), autor),
		    cover_url = COALESCE(NULLIF($3, ''), cover_url)
		WHERE id = $4::uuid`

	result, err := repo.db.Exec(ctx, query, title, author, coverURL, id)
	if err != nil {
		return fmt.Errorf("metadaten konnten nicht aktualisiert werden: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrBookNotFound
	}
	return nil
}

func (repo *BookRepository) UpdateBookCategory(ctx context.Context, id string, subject string, gradeLevel int16) error {
	query := `
		UPDATE buecher_titel
		SET subject = $1,
		    grade_level = $2
		WHERE id = $3::uuid`

	result, err := repo.db.Exec(ctx, query, subject, gradeLevel, id)
	if err != nil {
		return fmt.Errorf("kategorie konnte nicht aktualisiert werden: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrBookNotFound
	}
	return nil
}
