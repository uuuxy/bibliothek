package repository

import (
	"bibliothek/db"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	
)

// BookRepository defines operations for physical copies and book metadata search.
type BookRepository interface {
	// GetCopyByBarcode resolves a physical book copy by barcode, joining its title metadata. Returns nil if not found.
	GetCopyByBarcode(ctx context.Context, barcode string) (*BookCopy, error)
	// SearchTitles performs full-text query matching on book titles and authors, ranking results.
	SearchTitles(ctx context.Context, queryText string) ([]BookTitle, error)
	// SearchTitlesFuzzy performs a fuzzy search using ILIKE.
	SearchTitlesFuzzy(ctx context.Context, queryText string, limit int) ([]BookTitle, error)
}

type pgBookRepository struct {
	db db.PgxPoolIface
}

// NewBookRepository builds a PostgreSQL-backed BookRepository.
func NewBookRepository(db db.PgxPoolIface) BookRepository {
	return &pgBookRepository{db: db}
}

// GetCopyByBarcode resolves physical book copy by barcode.
func (r *pgBookRepository) GetCopyByBarcode(ctx context.Context, barcode string) (*BookCopy, error) {
	query := `
		SELECT 
			e.id, e.titel_id, e.barcode_id, coalesce(e.zustand_notiz, ''), e.erworben_am, e.ist_ausleihbar, e.ist_ausgesondert, e.erstellt_am, e.aktualisiert_am,
			t.titel, coalesce(t.autor, ''), coalesce(t.verlag, ''), coalesce(t.isbn, ''), coalesce(t.cover_url, ''), t.medientyp, t.erweiterte_eigenschaften
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE e.barcode_id = $1
		LIMIT 1
	`
	var bc BookCopy
	err := r.db.QueryRow(ctx, query, barcode).Scan(
		&bc.ID, &bc.TitelID, &bc.BarcodeID, &bc.ZustandNotiz, &bc.ErworbenAm, &bc.IstAusleihbar, &bc.IstAusgesondert, &bc.ErstelltAm, &bc.AktualisiertAm,
		&bc.Titel, &bc.Autor, &bc.Verlag, &bc.ISBN, &bc.CoverURL, &bc.Medientyp, &bc.ErweiterteEigenschaften,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &bc, nil
}

// SearchTitles performs full-text query matching on book titles and authors.
func (r *pgBookRepository) SearchTitles(ctx context.Context, queryText string) ([]BookTitle, error) {
	// Fall back to ILIKE substring checks if full text search is insufficient for partial keyword fragments.
	query := `
		SELECT id, titel, coalesce(untertitel, ''), coalesce(autor, ''), coalesce(isbn, ''), coalesce(verlag, ''), coalesce(erscheinungsjahr, 0), coalesce(beschreibung, ''), coalesce(cover_url, ''), medientyp, erstellt_am, aktualisiert_am, erweiterte_eigenschaften
		FROM buecher_titel
		WHERE 
			search_vector @@ plainto_tsquery('german', $1) 
			OR titel ILIKE '%' || $1 || '%'
			OR autor ILIKE '%' || $1 || '%'
			OR isbn ILIKE '%' || $1 || '%'
			OR replace(isbn, '-', '') = replace($1, '-', '')
		ORDER BY ts_rank(search_vector, plainto_tsquery('german', $1)) DESC, titel ASC
		LIMIT 50
	`
	rows, err := r.db.Query(ctx, query, queryText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []BookTitle
	for rows.Next() {
		var t BookTitle
		err := rows.Scan(
			&t.ID, &t.Titel, &t.Untertitel, &t.Autor, &t.ISBN, &t.Verlag, &t.Erscheinungsjahr, &t.Beschreibung, &t.CoverURL, &t.Medientyp, &t.ErstelltAm, &t.AktualisiertAm, &t.ErweiterteEigenschaften,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// SearchTitlesFuzzy performs a fuzzy search on book titles, authors, and ISBNs.
func (r *pgBookRepository) SearchTitlesFuzzy(ctx context.Context, queryText string, limit int) ([]BookTitle, error) {
	query := `
		SELECT id, titel, coalesce(untertitel, ''), coalesce(autor, ''), coalesce(isbn, ''), coalesce(verlag, ''), coalesce(erscheinungsjahr, 0), coalesce(beschreibung, ''), coalesce(cover_url, ''), medientyp, erstellt_am, aktualisiert_am, erweiterte_eigenschaften
		FROM buecher_titel
		WHERE titel ILIKE '%' || $1 || '%'
		   OR autor ILIKE '%' || $1 || '%'
		   OR isbn ILIKE '%' || $1 || '%'
		   OR replace(isbn, '-', '') = replace($1, '-', '')
		ORDER BY titel ASC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, queryText, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []BookTitle
	for rows.Next() {
		var t BookTitle
		err := rows.Scan(
			&t.ID, &t.Titel, &t.Untertitel, &t.Autor, &t.ISBN, &t.Verlag, &t.Erscheinungsjahr, &t.Beschreibung, &t.CoverURL, &t.Medientyp, &t.ErstelltAm, &t.AktualisiertAm, &t.ErweiterteEigenschaften,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
