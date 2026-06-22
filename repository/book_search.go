package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// GetCopyByBarcode löst einen Buch-Barcode auf.
func (r *pgBookRepository) GetCopyByBarcode(ctx context.Context, barcode string) (*BookCopy, error) {
	query := `
		SELECT 
			e.id, e.titel_id, coalesce(e.barcode_id, ''), coalesce(e.zustand_notiz, ''), e.erworben_am, coalesce(e.ist_ausleihbar, false), coalesce(e.ist_ausgesondert, false), e.erstellt_am, e.aktualisiert_am,
			coalesce(t.titel, ''), coalesce(t.autor, ''), coalesce(t.verlag, ''), coalesce(t.isbn, ''), coalesce(t.cover_url, ''), coalesce(t.medientyp, ''), coalesce(t.signatur, ''), coalesce(t.ziel_jahrgang, 0), coalesce(t.erweiterte_eigenschaften, '{}'::jsonb)
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE e.barcode_id = $1
		LIMIT 1
	`
	bc, err := scanBookCopy(r.db.QueryRow(ctx, query, barcode))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return bc, nil
}

// SearchTitles führt eine sprachenspezifische Volltextsuche über Buchtitel und Autoren durch.
func (r *pgBookRepository) SearchTitles(ctx context.Context, queryText string) ([]BookTitle, error) {
	query := `
		SELECT 
			id, coalesce(titel, ''), coalesce(untertitel, ''), coalesce(autor, ''), coalesce(isbn, ''), coalesce(verlag, ''), coalesce(erscheinungsjahr, 0), coalesce(beschreibung, ''), coalesce(cover_url, ''), coalesce(medientyp, ''), coalesce(signatur, ''), coalesce(ziel_jahrgang, 0), erstellt_am, aktualisiert_am, coalesce(erweiterte_eigenschaften, '{}'::jsonb)
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
		t, err := scanBookTitle(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// SearchTitlesFuzzy führt eine Wildcard-Suche für Auto-Vervollständigungen oder ungenaue Anfragen aus.
func (r *pgBookRepository) SearchTitlesFuzzy(ctx context.Context, queryText string, limit int) ([]BookTitle, error) {
	query := `
		SELECT 
			id, coalesce(titel, ''), coalesce(untertitel, ''), coalesce(autor, ''), coalesce(isbn, ''), coalesce(verlag, ''), coalesce(erscheinungsjahr, 0), coalesce(beschreibung, ''), coalesce(cover_url, ''), coalesce(medientyp, ''), coalesce(signatur, ''), coalesce(ziel_jahrgang, 0), erstellt_am, aktualisiert_am, coalesce(erweiterte_eigenschaften, '{}'::jsonb)
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
		t, err := scanBookTitle(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// GetTitleByIDTx sucht einen Buchtitel anhand seiner UUID, optimiert für Transaktionen.
func (r *pgBookRepository) GetTitleByIDTx(ctx context.Context, tx pgx.Tx, id string) (*BookTitle, error) {
	query := "SELECT id, coalesce(titel, ''), coalesce(autor, ''), coalesce(isbn, ''), coalesce(verlag, '') FROM buecher_titel WHERE id = $1"
	
	var t BookTitle
	err := tx.QueryRow(ctx, query, id).Scan(&t.ID, &t.Titel, &t.Autor, &t.ISBN, &t.Verlag)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
