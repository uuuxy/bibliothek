package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// UpdateCopyDamageNote setzt den Zustandstext eines Exemplars.
func (r *pgBookRepository) UpdateCopyDamageNote(ctx context.Context, id string, note string) error {
	query := `
		UPDATE buecher_exemplare
		SET zustand_notiz = $1, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, note, id)
	return err
}

// UpdateCopyBarcode ändert die Barcode-Zuordnung eines Exemplars.
func (r *pgBookRepository) UpdateCopyBarcode(ctx context.Context, id string, barcode string) error {
	query := `
		UPDATE buecher_exemplare
		SET barcode_id = $1, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, barcode, id)
	return err
}

// UpdateCopyStatus ändert den Verleihstatus und Zustand eines Exemplars.
func (r *pgBookRepository) UpdateCopyStatus(ctx context.Context, id string, istAusleihbar bool, istAusgesondert bool, zustandNotiz string) error {
	query := `
		UPDATE buecher_exemplare
		SET ist_ausleihbar = $1, ist_ausgesondert = $2, zustand_notiz = $3, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $4
	`
	_, err := r.db.Exec(ctx, query, istAusleihbar, istAusgesondert, zustandNotiz, id)
	return err
}

// DecommissionCopy sortiert ein Buch aus und sperrt es dauerhaft.
func (r *pgBookRepository) DecommissionCopy(ctx context.Context, id string) error {
	query := `
		UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// GenerateBarcodes erzeugt ein Array von count fortlaufenden Barcodes.
func (r *pgBookRepository) GenerateBarcodes(ctx context.Context, count int) ([]string, error) {
	query := "SELECT 'B-' || LPAD(nextval('barcode_seq')::TEXT, 5, '0') FROM generate_series(1, $1)"
	rows, err := r.db.Query(ctx, query, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var barcodes []string
	for rows.Next() {
		var barcodeID string
		if err := rows.Scan(&barcodeID); err != nil {
			return nil, err
		}
		barcodes = append(barcodes, barcodeID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(barcodes) != count {
		return nil, errors.New("barcode sequence generation count mismatch")
	}
	return barcodes, nil
}

// BulkInsertCopies fügt Exemplare im Bulk in die Datenbank ein.
func (r *pgBookRepository) BulkInsertCopies(ctx context.Context, copies []BookCopyInsert) error {
	if len(copies) == 0 {
		return nil
	}

	var copyRows [][]any
	for _, c := range copies {
		copyRows = append(copyRows, []any{
			c.TitelID, c.BarcodeID, c.ZustandNotiz, c.IstAusleihbar, c.EtikettGedruckt, c.Einkaufspreis,
		})
	}

	_, err := r.db.CopyFrom(
		ctx,
		pgx.Identifier{"buecher_exemplare"},
		[]string{"titel_id", "barcode_id", "zustand_notiz", "ist_ausleihbar", "etikett_gedruckt", "einkaufspreis"},
		pgx.CopyFromRows(copyRows),
	)
	return err
}

// BulkInsertCopiesTx fügt Exemplare im Bulk innerhalb einer Transaktion ein.
func (r *pgBookRepository) BulkInsertCopiesTx(ctx context.Context, tx pgx.Tx, copies []BookCopyInsert) error {
	if len(copies) == 0 {
		return nil
	}

	var copyRows [][]any
	for _, c := range copies {
		copyRows = append(copyRows, []any{
			c.TitelID, c.BarcodeID, c.ZustandNotiz, c.IstAusleihbar, c.EtikettGedruckt, c.Einkaufspreis,
		})
	}

	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"buecher_exemplare"},
		[]string{"titel_id", "barcode_id", "zustand_notiz", "ist_ausleihbar", "etikett_gedruckt", "einkaufspreis"},
		pgx.CopyFromRows(copyRows),
	)
	return err
}

// UpsertBookTitle speichert oder aktualisiert ein Buchtitel-Objekt.
// signatur: COALESCE-Schutz — die Signatur klebt physisch auf dem Buchrücken.
// Ein Re-Import mit leerer Signatur-Spalte darf einen befüllten Wert NIE
// überschreiben (sonst droht Re-Labeling des Bestands); eine nicht-leere
// Littera-Signatur gewinnt weiterhin 1:1.
func (r *pgBookRepository) UpsertBookTitle(ctx context.Context, t BookTitle) error {
	query := `
		INSERT INTO buecher_titel (titel, autor, isbn, verlag, erscheinungsjahr, signatur, ziel_jahrgang, aktualisiert_am)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, 0), NULLIF($6, ''), $7, CURRENT_TIMESTAMP)
		ON CONFLICT (isbn) DO UPDATE SET
		    titel = EXCLUDED.titel,
		    autor = EXCLUDED.autor,
		    verlag = EXCLUDED.verlag,
		    erscheinungsjahr = EXCLUDED.erscheinungsjahr,
		    signatur = COALESCE(NULLIF(EXCLUDED.signatur, ''), buecher_titel.signatur),
		    ziel_jahrgang = EXCLUDED.ziel_jahrgang,
		    aktualisiert_am = CURRENT_TIMESTAMP
	`

	_, err := r.db.Exec(ctx, query, t.Titel, t.Autor, t.ISBN, t.Verlag, t.Erscheinungsjahr, t.Signatur, t.ZielJahrgang)
	return err
}
