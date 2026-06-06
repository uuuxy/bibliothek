package inventur

import (
	"context"
	"fmt"
)

func (repo *BookRepository) GetClassGroups(ctx context.Context, branch string, sortOrder string) ([]ClassGroup, error) {
	query := `
		SELECT 
			cb.class_name, b.id, b.titel AS title, COALESCE(b.subject, '') AS subject, 
			COALESCE(b.track, '') AS track, COALESCE(b.cover_url, '') AS cover_url, 
			COALESCE(b.isbn, '') AS isbn,
			COUNT(e.id) FILTER (WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false AND a.id IS NULL) AS verfuegbar,
			COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND coalesce(e.zustand_notiz, '') NOT LIKE 'Im Zulauf%' AND coalesce(e.zustand_notiz, '') != 'bestellt' AND coalesce(e.zustand_notiz, '') NOT LIKE 'Bestellt%') AS gesamt
		FROM class_books cb
		JOIN buecher_titel b ON cb.book_id = b.id
		LEFT JOIN buecher_exemplare e ON e.titel_id = b.id
		LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
		WHERE ($1 = '' OR cb.class_name ILIKE '%' || $1 || '%')
		GROUP BY cb.class_name, b.id, b.titel, b.subject, b.track, b.cover_url, b.isbn
		ORDER BY `

	if branch == "" {
		// Fallback: Workflow-Reihenfolge F, G, R, H
		query += `
			CASE 
				WHEN cb.class_name ILIKE '%F%' THEN 1
				WHEN cb.class_name ILIKE '%G%' THEN 2
				WHEN cb.class_name ILIKE '%R%' THEN 3
				WHEN cb.class_name ILIKE '%H%' THEN 4
				ELSE 5
			END, `
	}

	gradeCast := `CAST(SUBSTRING(cb.class_name FROM '^[0-9]+') AS INTEGER)`
	descending := sortOrder == "desc"

	if descending {
		query += gradeCast + ` DESC, cb.class_name DESC, b.titel ASC`
	} else {
		query += gradeCast + ` ASC, cb.class_name ASC, b.titel ASC`
	}

	rows, err := repo.db.Query(ctx, query, branch)
	if err != nil {
		return nil, fmt.Errorf("klassen-bücher konnten nicht geladen werden: %w", err)
	}
	defer rows.Close()

	groupsMap := make(map[string][]ClassBook)
	var classNames []string

	for rows.Next() {
		var className string
		var book ClassBook
		err := rows.Scan(&className, &book.ID, &book.Title, &book.Subject, &book.Track, &book.CoverURL, &book.ISBN, &book.Verfuegbar, &book.Gesamt)
		if err != nil {
			return nil, fmt.Errorf("daten konnten nicht gelesen werden: %w", err)
		}
		book.Stock = book.Gesamt

		if _, exists := groupsMap[className]; !exists {
			classNames = append(classNames, className)
		}
		groupsMap[className] = append(groupsMap[className], book)
	}

	var result []ClassGroup
	for _, name := range classNames {
		result = append(result, ClassGroup{
			ClassName: name,
			Books:     groupsMap[name],
		})
	}

	return result, nil
}

func (repo *BookRepository) UpdateClassBooks(ctx context.Context, oldClassName string, newClassNames []string, bookIDs []string) error {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("transaktion konnte nicht gestartet werden: %w", err)
	}
	defer tx.Rollback(ctx)

	// If there's an old class name, delete it.
	if oldClassName != "" {
		_, err = tx.Exec(ctx, `DELETE FROM class_books WHERE class_name = $1`, oldClassName)
		if err != nil {
			return fmt.Errorf("alte zuweisungen konnten nicht gelöscht werden: %w", err)
		}
	}

	// ⚡ Bolt: Batch DELETE existing bindings for all target classes (overwrite)
	if len(newClassNames) > 0 {
		_, err = tx.Exec(ctx, `DELETE FROM class_books WHERE class_name = ANY($1)`, newClassNames)
		if err != nil {
			return fmt.Errorf("vorhandene zuweisungen des neuen namens konnten nicht gelöscht werden: %w", err)
		}
	}

	// ⚡ Bolt: Batch INSERT new bindings using PostgreSQL unnest
	if len(bookIDs) > 0 && len(newClassNames) > 0 {
		var insertClasses []string
		var insertBooks []string

		for _, className := range newClassNames {
			for _, bookID := range bookIDs {
				insertClasses = append(insertClasses, className)
				insertBooks = append(insertBooks, bookID)
			}
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO class_books (class_name, book_id)
			SELECT class_name, book_id::uuid FROM unnest($1::text[], $2::text[]) AS t(class_name, book_id)`, insertClasses, insertBooks)
		if err != nil {
			return fmt.Errorf("neue zuweisung konnte nicht gespeichert werden: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("transaktion konnte nicht abgeschlossen werden: %w", err)
	}

	return nil
}

func (repo *BookRepository) DeleteClassGroup(ctx context.Context, className string) error {
	_, err := repo.db.Exec(ctx, `DELETE FROM class_books WHERE class_name = $1`, className)
	if err != nil {
		return fmt.Errorf("klasse konnte nicht gelöscht werden: %w", err)
	}
	return nil
}

func (repo *BookRepository) NormalizeAllClasses(ctx context.Context) error {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("transaktion konnte nicht gestartet werden: %w", err)
	}
	defer tx.Rollback(ctx)

	// Step 1: Bereinigung von Leerzeichen.
	// Lösche Duplikate, die entstehen würden, wenn wir die Leerzeichen entfernen.
	_, err = tx.Exec(ctx, `
		DELETE FROM class_books cb1
		WHERE class_name LIKE '% %'
		AND EXISTS (
			SELECT 1 FROM class_books cb2
			WHERE cb2.class_name = REPLACE(cb1.class_name, ' ', '')
			AND cb2.book_id = cb1.book_id
		)
	`)
	if err != nil {
		return fmt.Errorf("fehler beim bereinigen doppelter klassennamen vor leerzeichen-entfernung: %w", err)
	}

	// Update zum Entfernen von Leerzeichen
	_, err = tx.Exec(ctx, `
		UPDATE class_books
		SET class_name = REPLACE(class_name, ' ', '')
		WHERE class_name LIKE '% %'
	`)
	if err != nil {
		return fmt.Errorf("fehler beim entfernen von leerzeichen in klassennamen: %w", err)
	}

	// Step 2: Delete books that would cause a unique key violation (führende Nullen)
	_, err = tx.Exec(ctx, `
		DELETE FROM class_books cb1
		WHERE (class_name ~ '^[1-9][^0-9]' OR class_name ~ '^[1-9]$')
		AND EXISTS (
			SELECT 1 FROM class_books cb2
			WHERE cb2.class_name = '0' || cb1.class_name
			AND cb2.book_id = cb1.book_id
		)
	`)
	if err != nil {
		return fmt.Errorf("fehler beim bereinigen doppelter klassennamen: %w", err)
	}

	// Step 2: Update the remaining rows
	_, err = tx.Exec(ctx, `
		UPDATE class_books
		SET class_name = '0' || class_name
		WHERE class_name ~ '^[1-9][^0-9]' OR class_name ~ '^[1-9]$'
	`)
	if err != nil {
		return fmt.Errorf("fehler beim normalisieren der klassennamen: %w", err)
	}

	return tx.Commit(ctx)
}

func (repo *BookRepository) AddBooksToClasses(ctx context.Context, classNames []string, bookIDs []string) error {
	if len(classNames) == 0 || len(bookIDs) == 0 {
		return nil
	}

	var insertClasses []string
	var insertBooks []string

	for _, className := range classNames {
		for _, bookID := range bookIDs {
			insertClasses = append(insertClasses, className)
			insertBooks = append(insertBooks, bookID)
		}
	}

	// Use ON CONFLICT DO NOTHING so existing assignments are ignored and not duplicated
	query := `
		INSERT INTO class_books (class_name, book_id)
		SELECT class_name, book_id::uuid FROM unnest($1::text[], $2::text[]) AS t(class_name, book_id)
		ON CONFLICT (class_name, book_id) DO NOTHING
	`

	_, err := repo.db.Exec(ctx, query, insertClasses, insertBooks)
	if err != nil {
		return fmt.Errorf("fehler beim hinzufügen der bücher zu den klassen: %w", err)
	}

	return nil
}
