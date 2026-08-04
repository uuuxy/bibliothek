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
			search_vector @@ plainto_tsquery('german', $1::text) 
			OR titel ILIKE '%' || $1::text || '%'
			OR autor ILIKE '%' || $1::text || '%'
			OR isbn ILIKE '%' || $1::text || '%'
			OR replace(isbn, '-', '') = replace($1::text, '-', '')
		ORDER BY ts_rank(search_vector, plainto_tsquery('german', $1::text)) DESC, titel ASC
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

// SearchTitlesFuzzy führt eine Wildcard-Suche für Auto-Vervollständigungen oder ungenaue Anfragen aus
// und liefert zusätzlich die Gesamtzahl der Treffer (nicht nur die des Limits).
//
// Wie bei der Schülersuche muss jedes Token irgendeine Spalte treffen, statt die
// komplette Eingabe eine einzelne: "rowling harry" findet damit den Titel über
// Autor + Titel, was der frühere Ganzstring-Vergleich nicht konnte. Die
// ISBN-Gleichheit bleibt als eigener Zweig bestehen — dort wird die ganze,
// bindestrichbereinigte Eingabe verglichen, nicht tokenweise.
func (r *pgBookRepository) SearchTitlesFuzzy(ctx context.Context, queryText string, limit int) ([]BookTitle, int, error) {
	tokens := suchTokens(queryText)
	if len(tokens) == 0 {
		return nil, 0, nil
	}

	query := `
		WITH tokens AS (
			SELECT suchnorm(t) AS norm, lower(t) AS roh FROM unnest($1::text[]) AS t
		)
		SELECT
			id, coalesce(titel, ''), coalesce(untertitel, ''), coalesce(autor, ''), coalesce(isbn, ''), coalesce(verlag, ''), coalesce(erscheinungsjahr, 0), coalesce(beschreibung, ''), coalesce(cover_url, ''), coalesce(medientyp, ''), coalesce(signatur, ''), coalesce(ziel_jahrgang, 0), erstellt_am, aktualisiert_am, coalesce(erweiterte_eigenschaften, '{}'::jsonb),
			count(*) OVER () AS gesamt
		FROM buecher_titel b
		WHERE (
			-- Index-Anker: erstes Token als direkte LIKE-Bedingung, damit der Planer
			-- die Ausdrucks-Indizes aus Migration 054 per BitmapOr zieht. Die
			-- Ausdrücke müssen buchstäblich denen der Indizes entsprechen — ein
			-- coalesce() darum herum macht den Zweig unindexierbar und kippt die
			-- ganze Abfrage in den Seq Scan (siehe SearchStudentsFuzzy).
			   suchnorm(b.titel)    LIKE '%' || suchnorm($4::text) || '%'
			OR suchnorm(b.autor)    LIKE '%' || suchnorm($4::text) || '%'
			OR lower(b.isbn)        LIKE '%' || lower($4::text) || '%'
			OR lower(b.signatur)    LIKE '%' || lower($4::text) || '%'
			OR replace(b.isbn, '-', '') = replace($2::text, '-', '')
		  )
		  AND (
			   replace(b.isbn, '-', '') = replace($2::text, '-', '')
			OR (
				SELECT bool_and(
					   suchnorm(coalesce(b.titel, ''))     LIKE '%' || tokens.norm || '%'
					OR suchnorm(coalesce(b.autor, ''))     LIKE '%' || tokens.norm || '%'
					OR lower(coalesce(b.isbn, ''))         LIKE '%' || tokens.roh || '%'
					OR lower(coalesce(b.signatur, ''))     LIKE '%' || tokens.roh || '%')
				FROM tokens
			   )
		  )
		ORDER BY
			(SELECT count(*) FROM tokens WHERE suchnorm(coalesce(b.titel, '')) LIKE tokens.norm || '%') DESC,
			b.titel ASC
		LIMIT $3
	`
	rows, err := r.db.Query(ctx, query, tokens, queryText, limit, tokens[0])
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []BookTitle
	gesamt := 0
	for rows.Next() {
		var zeilenGesamt int
		t, err := scanBookTitleMitZusatz(rows, &zeilenGesamt)
		if err != nil {
			return nil, 0, err
		}
		gesamt = zeilenGesamt
		results = append(results, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return results, gesamt, nil
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
