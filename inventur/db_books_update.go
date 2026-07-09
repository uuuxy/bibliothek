package inventur

import (
	"context"
	"fmt"
)

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
			jahrgang_bis = $13,
			untertitel = $14,
			verlag = $15,
			erscheinungsjahr = $16,
			beschreibung = $17,
			signatur = COALESCE(NULLIF($19, ''), signatur)
		WHERE id = $18`

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
		book.Untertitel,
		book.Verlag,
		book.Erscheinungsjahr,
		book.Beschreibung,
		id,
		book.Signatur, // $19 — leerer Wert lässt die verklebte Signatur unangetastet
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
			_, err := repo.db.Exec(ctx, `
				INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz)
				SELECT $1, 'SYS-' || nextval('sys_barcode_seq')::text, true, 'Automatisch generiert'
				FROM generate_series(1, $2)
			`, titelID, numToCreate)
			if err != nil {
				return fmt.Errorf("fehler beim generieren von exemplaren im batch: %w", err)
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
