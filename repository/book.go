package repository

import (
	"bibliothek/db"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// BookRepository definiert alle Datenbank-Operationen für physische Buchexemplare
// und die Suche nach Buchtiteln im Katalog.
type BookRepository interface {
	// GetCopyByBarcode sucht ein physisches Buchexemplar anhand seines Barcodes.
	// Liefert die Exemplardaten inklusive der Metadaten des verknüpften Buchtitels zurück.
	// Gibt nil zurück, wenn kein Exemplar gefunden wurde.
	GetCopyByBarcode(ctx context.Context, barcode string) (*BookCopy, error)

	// SearchTitles führt eine Volltextsuche (german tsvector) über Titel, Autoren und ISBNs aus
	// und sortiert die Ergebnisse nach ihrer Relevanz (ts_rank).
	SearchTitles(ctx context.Context, queryText string) ([]BookTitle, error)

	// SearchTitlesFuzzy führt eine performante Fuzzy-Suche (Teilstring-Suche mittels ILIKE)
	// über Titel, Autor und ISBN mit einer Ergebnismengenbegrenzung aus.
	SearchTitlesFuzzy(ctx context.Context, queryText string, limit int) ([]BookTitle, error)

	// UpdateCopyDamageNote aktualisiert die Zustandsnotiz (z. B. Beschädigungen) eines Buchexemplars.
	UpdateCopyDamageNote(ctx context.Context, id string, note string) error

	// UpdateCopyBarcode ändert den zugewiesenen Barcode eines physischen Exemplars.
	UpdateCopyBarcode(ctx context.Context, id string, barcode string) error

	// UpdateCopyStatus ändert den Ausleih- und Aussonderungsstatus eines Exemplars.
	UpdateCopyStatus(ctx context.Context, id string, istAusleihbar bool, istAusgesondert bool, zustandNotiz string) error

	// DecommissionCopy kennzeichnet ein Exemplar als dauerhaft ausgesondert und sperrt die Ausleihe.
	DecommissionCopy(ctx context.Context, id string) error

	// GenerateBarcodes erzeugt eine Serie fortlaufender Buch-Barcodes (Präfix "B-") über eine DB-Sequence.
	GenerateBarcodes(ctx context.Context, count int) ([]string, error)

	// BulkInsertCopies fügt mehrere Buchexemplare performant per Massen-Insert (CopyFrom) in die Datenbank ein.
	BulkInsertCopies(ctx context.Context, copies []BookCopyInsert) error
	
	// BulkInsertCopiesTx führt BulkInsertCopies innerhalb einer expliziten SQL-Transaktion aus.
	BulkInsertCopiesTx(ctx context.Context, tx pgx.Tx, copies []BookCopyInsert) error

	// UpsertBookTitle speichert oder aktualisiert ein Buchtitel-Objekt in der Datenbank.
	UpsertBookTitle(ctx context.Context, title BookTitle) error
}

// BookCopyInsert beschreibt die Datenstruktur für das Einfügen neuer Buchexemplare im Bulk-Verfahren.
type BookCopyInsert struct {
	// TitelID verweist auf die Metadaten des Buchtitels.
	TitelID string
	// BarcodeID ist der eindeutige Barcode des neuen Exemplars.
	BarcodeID string
	// ZustandNotiz dokumentiert den Initialzustand des Buchs (optional).
	ZustandNotiz string
	// IstAusleihbar gibt an, ob das Buch direkt verliehen werden darf.
	IstAusleihbar bool
	// EtikettGedruckt speichert, ob das Barcode-Etikett bereits gedruckt wurde.
	EtikettGedruckt bool
	// Einkaufspreis speichert den Netto-Anschaffungspreis des Exemplars.
	Einkaufspreis float64
}

// pgBookRepository implementiert das BookRepository für PostgreSQL.
type pgBookRepository struct {
	db db.PgxPoolIface
}

// NewBookRepository erstellt eine neue Instanz des PostgreSQL-basierten Book-Repositorys.
func NewBookRepository(db db.PgxPoolIface) BookRepository {
	return &pgBookRepository{db: db}
}

// scanBookCopy ist eine Hilfsfunktion zum Einlesen einer Zeile in ein BookCopy-Objekt.
func scanBookCopy(row Scanner) (*BookCopy, error) {
	var bc BookCopy
	err := row.Scan(
		&bc.ID, &bc.TitelID, &bc.BarcodeID, &bc.ZustandNotiz, &bc.ErworbenAm, &bc.IstAusleihbar, &bc.IstAusgesondert, &bc.ErstelltAm, &bc.AktualisiertAm,
		&bc.Titel, &bc.Autor, &bc.Verlag, &bc.ISBN, &bc.CoverURL, &bc.Medientyp, &bc.Signatur, &bc.ZielJahrgang, &bc.ErweiterteEigenschaften,
	)
	if err != nil {
		return nil, err
	}
	return &bc, nil
}

// scanBookTitle ist eine Hilfsfunktion zum Einlesen einer Zeile in ein BookTitle-Objekt.
func scanBookTitle(row Scanner) (*BookTitle, error) {
	var t BookTitle
	err := row.Scan(
		&t.ID, &t.Titel, &t.Untertitel, &t.Autor, &t.ISBN, &t.Verlag, &t.Erscheinungsjahr, &t.Beschreibung, &t.CoverURL, &t.Medientyp, &t.Signatur, &t.ZielJahrgang, &t.ErstelltAm, &t.AktualisiertAm, &t.ErweiterteEigenschaften,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

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
func (r *pgBookRepository) UpsertBookTitle(ctx context.Context, t BookTitle) error {
	query := `
		INSERT INTO buecher_titel (titel, autor, isbn, verlag, erscheinungsjahr, signatur, ziel_jahrgang, aktualisiert_am)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, 0), $6, $7, CURRENT_TIMESTAMP)
		ON CONFLICT (isbn) DO UPDATE SET 
		    titel = EXCLUDED.titel, 
		    autor = EXCLUDED.autor, 
		    verlag = EXCLUDED.verlag, 
		    erscheinungsjahr = EXCLUDED.erscheinungsjahr,
		    signatur = EXCLUDED.signatur,
		    ziel_jahrgang = EXCLUDED.ziel_jahrgang,
		    aktualisiert_am = CURRENT_TIMESTAMP
	`

	_, err := r.db.Exec(ctx, query, t.Titel, t.Autor, t.ISBN, t.Verlag, t.Erscheinungsjahr, t.Signatur, t.ZielJahrgang)
	return err
}
