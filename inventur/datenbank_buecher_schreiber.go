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
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted, medientyp, erweiterte_eigenschaften, jahrgang_von, jahrgang_bis)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9::text, '')::date, $10, $11, $12, $13)
		RETURNING id`

	medientyp := book.Medientyp
	if medientyp == "" {
		medientyp = "Buch"
	}

	properties := book.ErweiterteEigenschaften
	if properties == nil {
		properties = make(map[string]any)
	}

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
		medientyp,
		properties,
		book.JahrgangVon,
		book.JahrgangBis,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("buch konnte nicht erstellt werden: %w", handleDbError(err))
	}

	// Bestand synchronisieren
	if book.Stock > 0 {
		if syncErr := repo.syncBookStock(ctx, id, book.Stock); syncErr != nil {
			// Log error, but don't fail the creation
			fmt.Printf("Warnung: Konnte Exemplare nach Erstellung nicht synchronisieren: %v\n", syncErr)
		}
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
	medientypen := make([]string, len(books))
	jahrgaengeVon := make([]int, len(books))
	jahrgaengeBis := make([]int, len(books))

	for i, b := range books {
		isbns[i] = b.ISBN
		titles[i] = b.Title
		authors[i] = b.Author
		coverUrls[i] = b.CoverURL
		subjects[i] = b.Subject
		grades[i] = b.GradeLevel
		tracks[i] = b.Track
		// #nosec G115 - Stock is a physical book count, fits easily in int32
		stocks[i] = int32(b.Stock)
		lastCounteds[i] = b.LastCounted
		medientypen[i] = b.Medientyp
		if medientypen[i] == "" {
			medientypen[i] = "Buch"
		}
		jahrgaengeVon[i] = b.JahrgangVon
		jahrgaengeBis[i] = b.JahrgangBis
	}

	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted, medientyp, jahrgang_von, jahrgang_bis)
		SELECT t.isbn, t.titel, t.autor, t.cover_url, t.subject, t.grade_level, t.track, t.stock, NULLIF(t.last_counted_text, '')::date, t.medientyp, t.jahrgang_von, t.jahrgang_bis
		FROM UNNEST($1::text[], $2::text[], $3::text[], $4::text[], $5::text[], $6::smallint[], $7::text[], $8::int[], $9::text[], $10::text[], $11::int[], $12::int[])
		AS t(isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted_text, medientyp, jahrgang_von, jahrgang_bis)
		ON CONFLICT (isbn) DO UPDATE SET
			titel = EXCLUDED.titel,
			autor = EXCLUDED.autor,
			cover_url = EXCLUDED.cover_url,
			subject = EXCLUDED.subject,
			grade_level = EXCLUDED.grade_level,
			track = EXCLUDED.track,
			stock = buecher_titel.stock + EXCLUDED.stock,
			last_counted = EXCLUDED.last_counted,
			medientyp = EXCLUDED.medientyp,
			jahrgang_von = EXCLUDED.jahrgang_von,
			jahrgang_bis = EXCLUDED.jahrgang_bis
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
		medientypen,
		jahrgaengeVon,
		jahrgaengeBis,
	)
	if err != nil {
		return 0, fmt.Errorf("bücher konnten nicht im batch importiert werden: %w", err)
	}

	return cmdTag.RowsAffected(), nil
}

// UpsertBook inserts or updates a book record.
func (repo *BookRepository) UpsertBook(ctx context.Context, book Book) (string, error) {
	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted, medientyp, jahrgang_von, jahrgang_bis)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9::text, '')::date, $10, $11, $12)
		ON CONFLICT (isbn) DO UPDATE SET
			titel = EXCLUDED.titel,
			autor = EXCLUDED.autor,
			subject = EXCLUDED.subject,
			grade_level = EXCLUDED.grade_level,
			track = EXCLUDED.track,
			stock = buecher_titel.stock + EXCLUDED.stock,
			last_counted = EXCLUDED.last_counted,
			medientyp = EXCLUDED.medientyp,
			jahrgang_von = EXCLUDED.jahrgang_von,
			jahrgang_bis = EXCLUDED.jahrgang_bis
		RETURNING id`

	medientyp := book.Medientyp
	if medientyp == "" {
		medientyp = "Buch"
	}

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
		medientyp,
		book.JahrgangVon,
		book.JahrgangBis,
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
			last_counted = NULLIF($9::text, '')::date,
			medientyp = $10,
			erweiterte_eigenschaften = $11,
			jahrgang_von = $12,
			jahrgang_bis = $13
		WHERE id = $14`

	medientyp := book.Medientyp
	if medientyp == "" {
		medientyp = "Buch"
	}

	properties := book.ErweiterteEigenschaften
	if properties == nil {
		properties = make(map[string]any)
	}

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
		medientyp,
		properties,
		book.JahrgangVon,
		book.JahrgangBis,
		id,
	)
	if err != nil {
		return fmt.Errorf("buch konnte nicht aktualisiert werden: %w", handleDbError(err))
	}

	if result.RowsAffected() == 0 {
		return ErrBookNotFound
	}

	// Bestand synchronisieren
	if syncErr := repo.syncBookStock(ctx, id, book.Stock); syncErr != nil {
		// Log error, but don't fail the update entirely
		fmt.Printf("Warnung: Konnte Exemplare nach Aktualisierung nicht synchronisieren: %v\n", syncErr)
	}

	return nil
}

// DeleteBooks deletes multiple book records.
func (repo *BookRepository) DeleteBooks(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	var activeLoans int
	err := repo.db.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM ausleihen a 
		JOIN buecher_exemplare e ON a.exemplar_id = e.id 
		WHERE e.titel_id = ANY($1::uuid[]) AND a.rueckgabe_am IS NULL`, ids).Scan(&activeLoans)
	if err != nil {
		return fmt.Errorf("fehler bei der prüfung auf aktive ausleihen: %w", err)
	}
	if activeLoans > 0 {
		return fmt.Errorf("löschen abgebrochen: Mindestens ein Exemplar dieser Titel ist aktuell verliehen")
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

	// Clean up related records for ALL copies of these titles to prevent ON DELETE RESTRICT errors
	if _, err = repo.db.Exec(ctx, "DELETE FROM schadensfaelle WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = ANY($1::uuid[]))", ids); err != nil {
		return fmt.Errorf("failed to delete damage records for titles: %w", err)
	}
	if _, err = repo.db.Exec(ctx, "DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = ANY($1::uuid[])) AND rueckgabe_am IS NOT NULL", ids); err != nil {
		return fmt.Errorf("failed to delete past loans for titles: %w", err)
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
		// #nosec G304 - name is sanitized using filepath.Base
		_ = os.Remove(filepath.Join("uploads", name))
	}

	return nil
}

// syncBookStock synchronizes the physical buecher_exemplare records to match the expected stock.
func (repo *BookRepository) syncBookStock(ctx context.Context, titelID string, expectedStock int) error {
	var currentStock int
	err := repo.db.QueryRow(ctx, `SELECT COUNT(*) FROM buecher_exemplare WHERE titel_id = $1 AND ist_ausgesondert = false`, titelID).Scan(&currentStock)
	if err != nil {
		return fmt.Errorf("fehler beim ermitteln des aktuellen bestands: %w", err)
	}

	if expectedStock > currentStock {
		numToCreate := expectedStock - currentStock
		if numToCreate > 0 {
			_, _ = repo.db.Exec(ctx, `CREATE SEQUENCE IF NOT EXISTS sys_barcode_seq START 100000`)
			for i := 0; i < numToCreate; i++ {
				_, err := repo.db.Exec(ctx, `
					INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz)
					VALUES ($1, 'SYS-' || nextval('sys_barcode_seq')::text, true, 'Automatisch generiert')
				`, titelID)
				if err != nil {
					return fmt.Errorf("fehler beim generieren von exemplaren: %w", err)
				}
			}
		}
	} else if expectedStock < currentStock {
		numToRetire := currentStock - expectedStock
		
		// 1. Versuchen, nicht-ausgeliehene Exemplare auszusondern
		query := `
			UPDATE buecher_exemplare
			SET ist_ausgesondert = true, zustand_notiz = COALESCE(zustand_notiz || ' | ', '') || 'Automatisch ausgesondert'
			WHERE id IN (
				SELECT e.id
				FROM buecher_exemplare e
				LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
				WHERE e.titel_id = $1 AND e.ist_ausgesondert = false AND a.id IS NULL
				LIMIT $2
			)
		`
		result, err := repo.db.Exec(ctx, query, titelID, numToRetire)
		if err != nil {
			return fmt.Errorf("fehler beim aussondern von exemplaren: %w", err)
		}
		
		retired := result.RowsAffected()
		if retired < int64(numToRetire) {
			// 2. Fallback: Auch ausgeliehene Exemplare aussondern, falls nötig
			remainingToRetire := int64(numToRetire) - retired
			fallbackQuery := `
				UPDATE buecher_exemplare
				SET ist_ausgesondert = true, zustand_notiz = COALESCE(zustand_notiz || ' | ', '') || 'Automatisch ausgesondert (war ausgeliehen)'
				WHERE id IN (
					SELECT e.id
					FROM buecher_exemplare e
					WHERE e.titel_id = $1 AND e.ist_ausgesondert = false
					LIMIT $2
				)
			`
			_, err = repo.db.Exec(ctx, fallbackQuery, titelID, remainingToRetire)
			if err != nil {
				return fmt.Errorf("fehler beim aussondern (fallback): %w", err)
			}
		}
	}

	return nil
}
