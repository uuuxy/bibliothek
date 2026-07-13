package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// NormalisiereTitelKey bildet den Schlüssel für das Titel-Matching der Importe:
// Anführungszeichen raus, Whitespace kollabiert. Littera-Exporte kodieren
// Anführungszeichen uneinheitlich (PDF-Konvertierung: `""South Africa"""`,
// XML: `"South Africa"`) — ohne Normalisierung matchen dieselben Titel nie
// und jeder Import legt Dubletten an. Nur für Map-Schlüssel verwenden, der
// gespeicherte Titel behält seine Anführungszeichen.
func NormalisiereTitelKey(titel string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(titel, `"`, "")), " ")
}

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

// BulkUpsertBookTitles speichert viele Titel in EINEM gepipelineten Batch statt
// je Titel eine eigene Datenbank-Rundreise (der frühere N+1 ließ den MAB2-Import
// mit ~15.000 Titeln gegen eine nicht-lokale DB in Client-Timeouts laufen).
//
// Jeder Titel wird zuerst über die ISBN, dann über den Titel gegen den Bestand
// gematcht. Der Titel-Fallback gilt AUCH für Datensätze mit ISBN: die Bestands-CSV
// legt Titel ohne ISBN an — würde das Katalogisat-XML danach nur über die ISBN
// matchen, entstünde für jeden dieser Titel eine Dublette. Beim Titel-Match wird
// die ISBN nachgetragen (eine bereits gesetzte ISBN bleibt unangetastet). Die
// Signatur wird beim Update nie mit einem leeren Wert überschrieben (das
// Rücken-Etikett gewinnt).
//
// Zurückgegeben wird die Zahl der verarbeiteten (eingefügten oder
// aktualisierten) Titel.
func (r *pgBookRepository) BulkUpsertBookTitles(ctx context.Context, titles []BookTitle) (int, error) {
	if len(titles) == 0 {
		return 0, nil
	}

	// Alles-oder-nichts: entweder der komplette Import landet, oder gar nichts
	// (kein halb importierter Katalog bei einem Fehler in der Mitte).
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // No-op nach erfolgreichem Commit.

	// 1. Bestand einmalig vorladen (isbn→id, titel→id).
	isbnToID, titelToID, err := ladeTitelBestand(ctx, tx)
	if err != nil {
		return 0, err
	}

	const qInsert = `
		INSERT INTO buecher_titel (titel, autor, isbn, verlag, erscheinungsjahr, signatur, aktualisiert_am)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, 0), NULLIF($6, ''), CURRENT_TIMESTAMP)
	`
	const qUpdate = `
		UPDATE buecher_titel SET
			titel = $2,
			autor = $3,
			verlag = $4,
			erscheinungsjahr = NULLIF($5, 0),
			signatur = COALESCE(NULLIF($6, ''), signatur),
			isbn = COALESCE(isbn, NULLIF($7, '')),
			aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	// 2. In-Batch-Dedup (letzter Datensatz gewinnt): verhindert, dass derselbe
	//    Titel bzw. dieselbe ISBN innerhalb einer Datei doppelt geschrieben wird.
	c := &titelUpsertContext{
		isbnToID:  isbnToID,
		titelToID: titelToID,
		seenISBN:  make(map[string]bool),
		seenTitel: make(map[string]bool),
	}

	batch := &pgx.Batch{}
	queued := 0
	for _, t := range titles {
		if queueTitelUpsert(batch, t, c, qInsert, qUpdate) {
			queued++
		}
	}

	if queued == 0 {
		return 0, nil
	}

	br := tx.SendBatch(ctx, batch)
	for i := 0; i < queued; i++ {
		if _, err := br.Exec(); err != nil {
			_ = br.Close() //nolint:errcheck
			return 0, err
		}
	}
	if err := br.Close(); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return queued, nil
}

// ladeTitelBestand lädt den vorhandenen Titelbestand als isbn→id- und titel→id-Maps.
func ladeTitelBestand(ctx context.Context, tx pgx.Tx) (isbnToID, titelToID map[string]string, err error) {
	rows, err := tx.Query(ctx, "SELECT id, COALESCE(isbn, ''), titel FROM buecher_titel")
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	isbnToID = make(map[string]string)
	titelToID = make(map[string]string)
	for rows.Next() {
		var id, isbn, titel string
		if err := rows.Scan(&id, &isbn, &titel); err != nil {
			return nil, nil, err
		}
		if isbn != "" {
			isbnToID[isbn] = id
		}
		titelToID[NormalisiereTitelKey(titel)] = id
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return isbnToID, titelToID, nil
}

// titelUpsertContext bündelt die Lookup-Maps (Bestand) und die In-Batch-Dedup-Maps.
type titelUpsertContext struct {
	isbnToID, titelToID map[string]string
	seenISBN, seenTitel map[string]bool
}

// queueTitelUpsert reiht einen Titel als UPDATE (bekannt) oder INSERT (neu) in den Batch
// ein. Das Matching läuft zweistufig: bevorzugt über die ISBN, sonst über den Titel —
// auch für Datensätze MIT ISBN, weil der Bestand denselben Titel bereits ohne ISBN
// enthalten kann (Bestands-CSV). Dedupliziert innerhalb der Datei nach ISBN und Titel.
// Liefert false, wenn der Titel übersprungen wurde (leer oder Dublette).
func queueTitelUpsert(batch *pgx.Batch, t BookTitle, c *titelUpsertContext, qInsert, qUpdate string) bool {
	if t.Titel == "" {
		return false
	}
	if t.ISBN != "" && c.seenISBN[t.ISBN] {
		return false
	}

	titelKey := NormalisiereTitelKey(t.Titel)
	id, bekannt := c.isbnToID[t.ISBN]
	if !bekannt {
		id, bekannt = c.titelToID[titelKey]
	}
	// Unbekannter Titel, aber bereits in dieser Datei eingereiht: kein zweiter
	// INSERT — sonst legt eine Datei mit ISBN- und ISBN-losen Varianten desselben
	// Titels zwei Zeilen an.
	if !bekannt && c.seenTitel[titelKey] {
		return false
	}
	c.seenTitel[titelKey] = true
	if t.ISBN != "" {
		c.seenISBN[t.ISBN] = true
	}

	if bekannt {
		batch.Queue(qUpdate, id, t.Titel, t.Autor, t.Verlag, t.Erscheinungsjahr, t.Signatur, t.ISBN)
	} else {
		batch.Queue(qInsert, t.Titel, t.Autor, t.ISBN, t.Verlag, t.Erscheinungsjahr, t.Signatur)
	}
	return true
}
