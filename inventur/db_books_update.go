package inventur

import (
	"context"
	"fmt"

	"bibliothek/db"
)

// UpdateBook updates metadata fields of a book.
//
// aktualisiert_am wurde bis zum 10.08.2026 als EINZIGE Spalte von buecher_titel beim
// Aktualisieren nicht gesetzt. Sie steht in der API-Antwort (repository/book_search.go
// liefert sie mit) und behauptete dort den Zeitpunkt des ANLEGENS. In der Oberfläche liest
// sie heute niemand — aber ein Feld, das etwas anderes sagt als sein Name, ist eine Falle
// für den Nächsten, etwa für einen Abgleich, der „was hat sich seit gestern geändert"
// darüber beantworten will. Gefunden hat es schema_paritaet_test.go.
//
// Die Erklärung steht hier und nicht als SQL-Kommentar in der Anweisung: Ein Kommentar im
// Query-String reist bei jedem Aufruf zum Server mit und lässt jeden Test scheitern, der
// die Anweisung als Ganzes festhält.
func (repo *BookRepository) UpdateBook(ctx context.Context, id string, book Book) error {
	// subject ist FK auf die Systematik (Migration 078): unbekannte Fächer erst
	// registrieren, die kanonische Schreibweise schreiben, Leerwert wird NULL.
	kanonisch, err := StelleFaecherSicher(ctx, repo.db, []string{book.Subject})
	if err != nil {
		return err
	}

	query := `
		UPDATE buecher_titel
		SET isbn = $1,
			titel = $2,
			autor = $3,
			cover_url = $4,
			subject = NULLIF($5, ''),
			grade_level = $6,
			track = $7,
			last_counted = NULLIF($8::text, '')::date,
			medientyp = $9,
			erweiterte_eigenschaften = $10,
			jahrgang_von = $11,
			jahrgang_bis = $12,
			untertitel = $13,
			verlag = $14,
			erscheinungsjahr = $15,
			beschreibung = $16,
			signatur = COALESCE(NULLIF($18, ''), signatur),
			aktualisiert_am = NOW()
		WHERE id = $17`

	medientyp := book.Medientyp
	if medientyp == "" {
		medientyp = "Buch"
	}

	properties := book.ErweiterteEigenschaften
	if properties == nil {
		properties = make(map[string]any)
	}

	// Titel-Update und Bestands-Synchronisierung atomar: Schlägt der Sync fehl (z. B. die
	// zweistufige Aussonderung), bleibt sonst ein Titel mit falschem Bestand zurück, den
	// der Handler als erfolgreich meldet (Transaktionsgrenzen-Sweep A3). Der Sync-Fehler
	// wird deshalb ZURÜCKGEGEBEN, nicht nur geloggt.
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("buch konnte nicht aktualisiert werden: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	result, err := tx.Exec(
		ctx,
		query,
		book.ISBN,
		book.Title,
		book.Author,
		book.CoverURL,
		kanonisch[book.Subject],
		book.GradeLevel,
		book.Track,
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

	if err := repo.syncBookStock(ctx, tx, id, book.Stock); err != nil {
		return fmt.Errorf("exemplare konnten nicht synchronisiert werden: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("buch konnte nicht aktualisiert werden: %w", err)
	}
	return nil
}

// syncBookStock synchronizes the physical buecher_exemplare records to match the expected stock.
func (repo *BookRepository) syncBookStock(ctx context.Context, q dbSchreiber, titelID string, expectedStock int) error {
	var currentStock int
	err := q.QueryRow(ctx, `SELECT COUNT(*) FROM buecher_exemplare WHERE titel_id = $1 AND ist_ausgesondert = false`, titelID).Scan(&currentStock)
	if err != nil {
		return fmt.Errorf("fehler beim ermitteln des aktuellen bestands: %w", err)
	}

	if expectedStock > currentStock {
		numToCreate := expectedStock - currentStock
		if numToCreate > 0 {
			_, _ = q.Exec(ctx, `CREATE SEQUENCE IF NOT EXISTS sys_barcode_seq START 100000`) //nolint:errcheck
			_, err := q.Exec(ctx, `
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
			SET ist_ausgesondert = true, aussonderung_grund = 'BESTANDSKORREKTUR',
			    zustand_notiz = COALESCE(zustand_notiz || ' | ', '') || 'Automatisch ausgesondert'
			WHERE id IN (
				SELECT e.id
				FROM buecher_exemplare e
				LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
				WHERE e.titel_id = $1 AND e.ist_ausgesondert = false AND a.id IS NULL
				LIMIT $2
			)
		`
		result, err := q.Exec(ctx, query, titelID, numToRetire)
		if err != nil {
			return fmt.Errorf("fehler beim aussondern von exemplaren: %w", err)
		}

		retired := result.RowsAffected()
		if retired < int64(numToRetire) {
			// 2. Fallback: Auch ausgeliehene Exemplare aussondern, falls nötig
			remainingToRetire := int64(numToRetire) - retired
			fallbackQuery := `
				UPDATE buecher_exemplare
				SET ist_ausgesondert = true, aussonderung_grund = 'BESTANDSKORREKTUR',
				    zustand_notiz = COALESCE(zustand_notiz || ' | ', '') || 'Automatisch ausgesondert (war ausgeliehen)'
				WHERE id IN (
					SELECT e.id
					FROM buecher_exemplare e
					WHERE e.titel_id = $1 AND e.ist_ausgesondert = false
					LIMIT $2
				)
			`
			_, err = q.Exec(ctx, fallbackQuery, titelID, remainingToRetire)
			if err != nil {
				return fmt.Errorf("fehler beim aussondern (fallback): %w", err)
			}
		}
	}

	return nil
}
