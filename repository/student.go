package repository

import (
	"bibliothek/db"
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// Scanner kapselt die Scan-Schnittstelle von pgx.Row und pgx.Rows,
// um gemeinsame Helferfunktionen zum Einlesen von Zeilen zu ermöglichen.
type Scanner interface {
	Scan(dest ...any) error
}

// StudentRepository definiert die Operationen zur Abfrage und zum Abgleich von Schülern in der Datenbank.
type StudentRepository interface {
	// GetByBarcode sucht einen Schüler anhand seiner Barcode-ID (Schülerausweis).
	// Liefert nil zurück, wenn kein Schüler gefunden wurde.
	GetByBarcode(ctx context.Context, barcode string) (*Student, error)

	// GetByID sucht einen Schüler anhand seiner UUID (Primärschlüssel).
	// Liefert nil zurück, wenn kein Schüler gefunden wurde.
	GetByID(ctx context.Context, id string) (*Student, error)

	// SearchStudentsFuzzy führt eine Teilstring-Suche über Vorname, Nachname und Barcode-ID aus.
	SearchStudentsFuzzy(ctx context.Context, queryText string, limit int) ([]Student, error)

	// GetNextSequence ermittelt die nächste freie Barcode-Nummer für neue Schülerausweise (Format: "S-1xxxx").
	GetNextSequence(ctx context.Context) (int, error)

	// GetAllLUSDStudents lädt alle Schüler-IDs, LUSD-IDs, Namen und Geburtsdaten zur Vorbereitung eines LUSD-Abgleichs.
	GetAllLUSDStudents(ctx context.Context) ([]Student, error)

	// BulkSyncLUSD führt den LUSD-Datenabgleich (Massen-Update und Massen-Insert) in einer Transaktion durch.
	// Schueler, die nicht mehr im LUSD-Datenbestand gelistet sind, werden automatisch als Schulabgänger (ist_abgaenger = true) markiert
	// und deren Vormerkungen gelöscht.
	// Gibt die Anzahl der Abgänger zurück, die noch offene Ausleihen haben.
	BulkSyncLUSD(ctx context.Context, updates []StudentUpdate, inserts []StudentInsert, allLusdIDs []string) (int, error)
}

// StudentUpdate definiert die Datenstruktur für Aktualisierungen eines Schülers während des LUSD-Imports.
type StudentUpdate struct {
	ID           string
	Vorname      string
	Nachname     string
	Klasse       string
	Geburtsdatum *string // Format: YYYY-MM-DD
	LusdID       *string
}

// StudentInsert definiert die Datenstruktur für neu anzulegende Schüler während des LUSD-Imports.
type StudentInsert struct {
	BarcodeID     string
	Vorname       string
	Nachname      string
	Klasse        string
	Geburtsdatum  *string // Format: YYYY-MM-DD
	AbgaengerJahr int
	LusdID        *string
	IstAbgaenger  bool
}

// pgStudentRepository implementiert das StudentRepository für PostgreSQL.
type pgStudentRepository struct {
	db db.PgxPoolIface
}

// NewStudentRepository erzeugt eine neue Instanz des PostgreSQL-basierten StudentRepositorys.
func NewStudentRepository(db db.PgxPoolIface) StudentRepository {
	return &pgStudentRepository{db: db}
}

// scanStudent ist eine Hilfsfunktion zum Einlesen einer Datenbankzeile in das Student-Modell.
func scanStudent(row Scanner) (*Student, error) {
	var s Student
	err := row.Scan(
		&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.LusdID, &s.IstAbgaenger, &s.Geburtsdatum, &s.ErstelltAm, &s.AktualisiertAm, &s.IsManuallyBlocked, &s.BlockReason,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetByBarcode liest einen Schüler anhand seiner Barcode-ID aus.
func (r *pgStudentRepository) GetByBarcode(ctx context.Context, barcode string) (*Student, error) {
	query := `
		SELECT id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, ''), coalesce(abgaenger_jahr, 0), coalesce(ist_gesperrt, false), lusd_id, coalesce(ist_abgaenger, false), TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am, coalesce(is_manually_blocked, false), block_reason
		FROM schueler
		WHERE barcode_id = $1 AND deleted_at IS NULL
		LIMIT 1
	`
	s, err := scanStudent(r.db.QueryRow(ctx, query, barcode))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

// GetByID liest einen Schüler anhand seiner UUID aus.
func (r *pgStudentRepository) GetByID(ctx context.Context, id string) (*Student, error) {
	query := `
		SELECT id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, ''), coalesce(abgaenger_jahr, 0), coalesce(ist_gesperrt, false), lusd_id, coalesce(ist_abgaenger, false), TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am, coalesce(is_manually_blocked, false), block_reason
		FROM schueler
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1
	`
	s, err := scanStudent(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

// SearchStudentsFuzzy durchsucht die Schülerschaft nach Namen oder Barcodes.
func (r *pgStudentRepository) SearchStudentsFuzzy(ctx context.Context, queryText string, limit int) ([]Student, error) {
	query := `
		SELECT id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, ''), coalesce(abgaenger_jahr, 0), coalesce(ist_gesperrt, false), lusd_id, coalesce(ist_abgaenger, false), TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am, coalesce(is_manually_blocked, false), block_reason
		FROM schueler
		WHERE (vorname ILIKE '%' || $1 || '%' 
		   OR nachname ILIKE '%' || $1 || '%'
		   OR barcode_id ILIKE '%' || $1 || '%')
		  AND deleted_at IS NULL
		ORDER BY nachname ASC, vorname ASC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, queryText, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Student
	for rows.Next() {
		s, err := scanStudent(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// GetNextSequence ermittelt die fortlaufende Barcode-Sequenz für Schüler (z. B. "S-10005").
func (r *pgStudentRepository) GetNextSequence(ctx context.Context) (int, error) {
	var lastBarcode string
	qLast := `
		SELECT barcode_id 
		FROM schueler 
		WHERE barcode_id LIKE 'S-%' AND deleted_at IS NULL
		ORDER BY barcode_id DESC 
		LIMIT 1
	`
	err := r.db.QueryRow(ctx, qLast).Scan(&lastBarcode)
	startNum := 10001
	if err == nil && len(lastBarcode) > 2 {
		if parsed, err2 := strconv.Atoi(lastBarcode[2:]); err2 == nil {
			startNum = parsed + 1
		}
	}
	return startNum, nil
}

// GetAllLUSDStudents liest alle LUSD-relevanten Felder aus der Datenbank aus.
func (r *pgStudentRepository) GetAllLUSDStudents(ctx context.Context) ([]Student, error) {
	rows, err := r.db.Query(ctx, "SELECT id, lusd_id, lower(vorname), lower(nachname), coalesce(geburtsdatum, '1900-01-01'::DATE) FROM schueler WHERE deleted_at IS NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Student
	for rows.Next() {
		var s Student
		var geb time.Time
		if err := rows.Scan(&s.ID, &s.LusdID, &s.Vorname, &s.Nachname, &geb); err == nil {
			dateStr := geb.Format("2006-01-02")
			s.Geburtsdatum = &dateStr
			results = append(results, s)
		}
	}
	return results, rows.Err()
}

// BulkSyncLUSD synchronisiert Schülerdaten aus dem hessischen LUSD-Import.
// Diese Methode verwendet Massentransaktions-Methoden (UNNEST für Massen-Updates und CopyFrom für Massen-Inserts)
// zur Optimierung der Performance auf Produktionsdatenbanken.
func (r *pgStudentRepository) BulkSyncLUSD(ctx context.Context, updates []StudentUpdate, inserts []StudentInsert, allLusdIDs []string) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Massen-Update mit pgx und UNNEST durchführen (reduziert Roundtrips drastisch)
	if len(updates) > 0 {
		var updID, updVorname, updNach, updKlasse []string
		var updGeb, updLusd []*string

		for _, u := range updates {
			updID = append(updID, u.ID)
			updVorname = append(updVorname, u.Vorname)
			updNach = append(updNach, u.Nachname)
			updKlasse = append(updKlasse, u.Klasse)
			updGeb = append(updGeb, u.Geburtsdatum)
			updLusd = append(updLusd, u.LusdID)
		}

		qUpdate := `
			UPDATE schueler s
			SET vorname = d.vorname,
				nachname = d.nachname,
				klasse = d.klasse,
				geburtsdatum = d.geburtsdatum::date,
				ist_abgaenger = false,
				aktualisiert_am = CURRENT_TIMESTAMP,
				lusd_id = COALESCE(d.lusd_id, s.lusd_id)
			FROM (
				SELECT * FROM UNNEST($1::uuid[], $2::varchar[], $3::varchar[], $4::varchar[], $5::varchar[], $6::varchar[])
				AS u(id, vorname, nachname, klasse, geburtsdatum, lusd_id)
			) d
			WHERE s.id = d.id
		`
		_, err = tx.Exec(ctx, qUpdate, updID, updVorname, updNach, updKlasse, updGeb, updLusd)
		if err != nil {
			return 0, err
		}
	}

	// 2. Massen-Insert per CopyFrom durchführen
	if len(inserts) > 0 {
		var copyRows [][]any
		for _, i := range inserts {
			var geb any = nil
			if i.Geburtsdatum != nil {
				geb = *i.Geburtsdatum
			}
			copyRows = append(copyRows, []any{
				i.BarcodeID, i.Vorname, i.Nachname, i.Klasse, geb, i.AbgaengerJahr, i.LusdID, i.IstAbgaenger,
			})
		}
		_, err = tx.CopyFrom(
			ctx,
			pgx.Identifier{"schueler"},
			[]string{"barcode_id", "vorname", "nachname", "klasse", "geburtsdatum", "abgaenger_jahr", "lusd_id", "ist_abgaenger"},
			pgx.CopyFromRows(copyRows),
		)
		if err != nil {
			return 0, err
		}
	}

	// 3. Alle Schüler als Abgänger markieren, die eine LUSD-ID besitzen, aber nicht mehr in der aktuellen Importdatei enthalten sind
	qMarkAbgaenger := `
		UPDATE schueler
		SET ist_abgaenger = true, 
		    abgaenger_jahr = EXTRACT(YEAR FROM CURRENT_DATE),
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE lusd_id IS NOT NULL AND NOT (lusd_id = ANY($1)) AND ist_abgaenger = false
	`
	_, err = tx.Exec(ctx, qMarkAbgaenger, allLusdIDs)
	if err != nil {
		return 0, err
	}

	// 4. Reservierungen (Vormerkungen) von Abgängern automatisch löschen
	qDeleteVormerkungen := `
		DELETE FROM vormerkungen 
		WHERE schueler_id IN (
			SELECT id FROM schueler WHERE ist_abgaenger = true AND deleted_at IS NULL
		)
	`
	_, err = tx.Exec(ctx, qDeleteVormerkungen)
	if err != nil {
		return 0, err
	}

	// 5. Zählen, wie viele der neuen Abgänger noch Bücher zu Hause haben, damit Administratoren benachrichtigt werden können
	var abgaengerOpenCount int
	qCountLoans := `
		SELECT COUNT(DISTINCT schueler_id)
		FROM ausleihen
		WHERE rueckgabe_am IS NULL 
		  AND schueler_id IN (
			  SELECT id FROM schueler WHERE ist_abgaenger = true AND deleted_at IS NULL
		  )
	`
	err = tx.QueryRow(ctx, qCountLoans).Scan(&abgaengerOpenCount)
	if err != nil {
		return 0, err
	}

	return abgaengerOpenCount, tx.Commit(ctx)
}
