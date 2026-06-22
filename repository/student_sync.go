package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

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
