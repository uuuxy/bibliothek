package api

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bibliothek/db"
)

// syncLUSDData synchronizes the parsed LUSD rows with the database, performing batch inserts, updates, and flagging graduates.
func syncLUSDData(ctx context.Context, dbPool db.PgxPoolIface, parsedRows []parsedStudentRow, lusdIDs []string) (*LUSDImportResponse, error) {
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var lastBarcode string
	qLast := `
		SELECT barcode_id 
		FROM schueler 
		WHERE barcode_id LIKE 'S-%' 
		ORDER BY barcode_id DESC 
		LIMIT 1
	`
	err = tx.QueryRow(ctx, qLast).Scan(&lastBarcode)
	startNum := 10001
	if err == nil {
		re := regexp.MustCompile(`S-(\d+)`)
		matches := re.FindStringSubmatch(lastBarcode)
		if len(matches) > 1 {
			if parsed, err := strconv.Atoi(matches[1]); err == nil {
				startNum = parsed + 1
			}
		}
	}

	var newCount int
	var updatedCount int

	if len(parsedRows) > 0 {
		type existingStudent struct {
			ID            string
			LusdID        *string
			VornameLower  string
			NachnameLower string
			GebDatum      string
		}

		dbStudents := make([]existingStudent, 0)
		rows, err := tx.Query(ctx, "SELECT id, lusd_id, lower(vorname), lower(nachname), coalesce(geburtsdatum, '1900-01-01'::DATE) FROM schueler")
		if err != nil {
			return nil, fmt.Errorf("failed to load existing students: %w", err)
		}
		for rows.Next() {
			var s existingStudent
			var geb time.Time
			if err := rows.Scan(&s.ID, &s.LusdID, &s.VornameLower, &s.NachnameLower, &geb); err == nil {
				s.GebDatum = geb.Format("2006-01-02")
				dbStudents = append(dbStudents, s)
			}
		}
		rows.Close()

		mapLusd := make(map[string]string)
		mapFallback := make(map[string]string)
		for _, s := range dbStudents {
			if s.LusdID != nil && *s.LusdID != "" {
				mapLusd[*s.LusdID] = s.ID
			}
			key := s.VornameLower + "|" + s.NachnameLower + "|" + s.GebDatum
			mapFallback[key] = s.ID
		}

		var (
			updID      []string
			updVorname []string
			updNach    []string
			updKlasse  []string
			updGeb     []*time.Time
			updLusd    []*string

			insBarcode []string
			insVorname []string
			insNach    []string
			insKlasse  []string
			insGeb     []*time.Time
			insAbJahr  []int
			insLusd    []*string
		)

		for _, p := range parsedRows {
			var dbID string
			if p.LusdID != "" {
				dbID = mapLusd[p.LusdID]
			}
			if dbID == "" {
				gebStr := "1900-01-01"
				if p.GebDatum != nil {
					gebStr = p.GebDatum.Format("2006-01-02")
				}
				key := strings.ToLower(p.Vorname) + "|" + strings.ToLower(p.Nachname) + "|" + gebStr
				dbID = mapFallback[key]
			}

			var ptrLusd *string
			if p.LusdID != "" {
				lusd := p.LusdID
				ptrLusd = &lusd
			}

			if dbID != "" && dbID != "processing" {
				updID = append(updID, dbID)
				updVorname = append(updVorname, p.Vorname)
				updNach = append(updNach, p.Nachname)
				updKlasse = append(updKlasse, p.Klasse)
				updGeb = append(updGeb, p.GebDatum)
				updLusd = append(updLusd, ptrLusd)
				updatedCount++
			} else {
				barcode := fmt.Sprintf("S-%05d", startNum)
				startNum++
				insBarcode = append(insBarcode, barcode)
				insVorname = append(insVorname, p.Vorname)
				insNach = append(insNach, p.Nachname)
				insKlasse = append(insKlasse, p.Klasse)
				insGeb = append(insGeb, p.GebDatum)
				insAbJahr = append(insAbJahr, calculateAbgaengerJahr(p.Klasse))
				insLusd = append(insLusd, ptrLusd)
				newCount++

				if p.LusdID != "" {
					mapLusd[p.LusdID] = "processing"
				}
			}
		}

		if len(updID) > 0 {
			qUpdate := `
				UPDATE schueler s
				SET vorname = d.vorname,
					nachname = d.nachname,
					klasse = d.klasse,
					geburtsdatum = d.geburtsdatum,
					ist_abgaenger = false,
					aktualisiert_am = CURRENT_TIMESTAMP,
					lusd_id = COALESCE(d.lusd_id, s.lusd_id)
				FROM (
					SELECT * FROM UNNEST($1::uuid[], $2::varchar[], $3::varchar[], $4::varchar[], $5::date[], $6::varchar[])
					AS u(id, vorname, nachname, klasse, geburtsdatum, lusd_id)
				) d
				WHERE s.id = d.id
			`
			_, err = tx.Exec(ctx, qUpdate, updID, updVorname, updNach, updKlasse, updGeb, updLusd)
			if err != nil {
				return nil, fmt.Errorf("bulk update failed: %w", err)
			}
		}

		if len(insBarcode) > 0 {
			qInsert := `
				INSERT INTO schueler (barcode_id, vorname, nachname, klasse, geburtsdatum, abgaenger_jahr, lusd_id, ist_abgaenger)
				SELECT * FROM UNNEST($1::varchar[], $2::varchar[], $3::varchar[], $4::varchar[], $5::date[], $6::int[], $7::varchar[], $8::boolean[])
			`
			arrIstAbg := make([]bool, len(insBarcode))
			_, err = tx.Exec(ctx, qInsert, insBarcode, insVorname, insNach, insKlasse, insGeb, insAbJahr, insLusd, arrIstAbg)
			if err != nil {
				return nil, fmt.Errorf("bulk insert failed: %w", err)
			}
		}
	}

	qMarkAbgaenger := `
		UPDATE schueler
		SET ist_abgaenger = true, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE lusd_id IS NOT NULL AND NOT (lusd_id = ANY($1)) AND ist_abgaenger = false
	`
	_, err = tx.Exec(ctx, qMarkAbgaenger, lusdIDs)
	if err != nil {
		return nil, fmt.Errorf("diffing update failed: %w", err)
	}

	var abgaengerOpenCount int
	qCountLoans := `
		SELECT COUNT(DISTINCT schueler_id)
		FROM ausleihen
		WHERE rueckgabe_am IS NULL 
		  AND schueler_id IN (
			  SELECT id FROM schueler WHERE ist_abgaenger = true
		  )
	`
	err = tx.QueryRow(ctx, qCountLoans).Scan(&abgaengerOpenCount)
	if err != nil {
		return nil, fmt.Errorf("counting active loans for graduates failed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("transaction commit failed: %w", err)
	}

	return &LUSDImportResponse{
		Neu:                         newCount,
		Aktualisiert:                updatedCount,
		AbgaengerMitOffenenBuechern: abgaengerOpenCount,
	}, nil
}
