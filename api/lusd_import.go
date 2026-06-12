package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bibliothek/apierrors"
)

// LUSDImportResponse matches the required JSON response structure.
type LUSDImportResponse struct {
	Neu                         int `json:"neu"`
	Aktualisiert                int `json:"aktualisiert"`
	AbgaengerMitOffenenBuechern int `json:"abgaenger_mit_offenen_buechern"`
}

// ImportLUSDHandler parses LUSD school-year changeover CSVs, upserting student records,
// flagging students not in the CSV as graduates, and returning active loan counts for graduates.
func (s *Server) ImportLUSDHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Limit multipart form size to max 5MB
		r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
		if err := r.ParseMultipartForm(5 << 20); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		defer func() { _ = file.Close() }()

		content, err := io.ReadAll(file)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 2. Detect CSV delimiter (semicolon vs comma)
		delimiter := ','
		contentStr := string(content)
		if strings.Count(contentStr, ";") > strings.Count(contentStr, ",") {
			delimiter = ';'
		}

		reader := csv.NewReader(strings.NewReader(contentStr))
		reader.Comma = delimiter
		reader.LazyQuotes = true

		// Read headers
		headers, err := reader.Read()
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		// DSGVO Art. 5 Abs. 1 lit. c – Datensparsamkeit:
		// Ausschließlich die folgenden fünf Felder werden aus der CSV gelesen und
		// verarbeitet. Alle weiteren Spalten (Adress-, Kontakt- und sonstige
		// personenbezogene Daten) werden nie indiziert und sofort verworfen.
		const (
			colLUSDID       = "lusd_id"
			colVorname      = "vorname"
			colNachname     = "nachname"
			colKlasse       = "klasse"
			colGeburtsdatum = "geburtsdatum" // optional
		)

		headerMap := make(map[string]int)
		for idx, h := range headers {
			norm := strings.ToLower(strings.TrimSpace(h))
			// Whitelist: nur erlaubte Spalten werden im Index registriert.
			switch norm {
			case colLUSDID, colVorname, colNachname, colKlasse, colGeburtsdatum:
				headerMap[norm] = idx
				// Alle anderen Spalten werden bewusst ignoriert (DSGVO-Whitelist).
			}
		}

		// Validate required headers
		requiredCols := []string{colLUSDID, colVorname, colNachname, colKlasse}
		for _, col := range requiredCols {
			if _, exists := headerMap[col]; !exists {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("missing required column '%s'", col))
				return
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
		defer cancel()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		// 4. Determine next sequential barcode index for S-XXXXX barcodes
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

		lusdIDs := make([]string, 0)
		var newCount int
		var updatedCount int
		lineNum := 1

		type studentRow struct {
			LusdID   string
			Vorname  string
			Nachname string
			Klasse   string
			GebDatum *time.Time
			LineNum  int
		}
		var parsedRows []studentRow

		// 5a. Parse rows into memory
		// We use a map to deduplicate rows by lusd_id, keeping the latest one,
		// to prevent "ON CONFLICT DO UPDATE command cannot affect row a second time" errors.
		seenIndex := make(map[string]int)

		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			lineNum++
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("error parsing row %d: %w", lineNum, err))
				return
			}

			// Whitelist: Nur die fünf erlaubten LUSD-Felder werden extrahiert.
			lusdID := strings.TrimSpace(row[headerMap[colLUSDID]])
			vorname := strings.TrimSpace(row[headerMap[colVorname]])
			nachname := strings.TrimSpace(row[headerMap[colNachname]])
			klasse := strings.TrimSpace(row[headerMap[colKlasse]])

			// geburtsdatum ist optional; nicht alle LUSD-Exporte enthalten es.
			var geburtsdatum *time.Time
			if idx, ok := headerMap[colGeburtsdatum]; ok && idx < len(row) {
				if raw := strings.TrimSpace(row[idx]); raw != "" {
					for _, layout := range []string{"02.01.2006", "2006-01-02", "01/02/2006"} {
						if t, parseErr := time.ParseInLocation(layout, raw, time.UTC); parseErr == nil {
							t2 := t
							geburtsdatum = &t2
							break
						}
					}
					// Nicht parsbare Werte werden als NULL behandelt und nicht protokolliert
				}
			}

			if vorname == "" || nachname == "" || klasse == "" {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("empty value on row %d", lineNum))
				return
			}

			sRow := studentRow{
				LusdID:   lusdID,
				Vorname:  vorname,
				Nachname: nachname,
				Klasse:   klasse,
				GebDatum: geburtsdatum,
				LineNum:  lineNum,
			}

			if lusdID != "" {
				if idx, exists := seenIndex[lusdID]; exists {
					// Replace the existing one
					parsedRows[idx] = sRow
					continue
				}
				seenIndex[lusdID] = len(parsedRows)
				lusdIDs = append(lusdIDs, lusdID)
			}
			parsedRows = append(parsedRows, sRow)
		}

		if len(parsedRows) > 0 {
			// Fast resolution of existing students in memory
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
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to load existing students: %w", err))
				return
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

			// Bulk UPDATE
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
					apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("bulk update failed: %w", err))
					return
				}
			}

			// Bulk INSERT
			if len(insBarcode) > 0 {
				qInsert := `
					INSERT INTO schueler (barcode_id, vorname, nachname, klasse, geburtsdatum, abgaenger_jahr, lusd_id, ist_abgaenger)
					SELECT * FROM UNNEST($1::varchar[], $2::varchar[], $3::varchar[], $4::varchar[], $5::date[], $6::int[], $7::varchar[], $8::boolean[])
				`
				arrIstAbg := make([]bool, len(insBarcode))
				_, err = tx.Exec(ctx, qInsert, insBarcode, insVorname, insNach, insKlasse, insGeb, insAbJahr, insLusd, arrIstAbg)
				if err != nil {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("bulk insert failed: %w", err))
					return
				}
			}
		}

		// 6. Diffing: Set ist_abgaenger = true for students not present in CSV
		qMarkAbgaenger := `
			UPDATE schueler
			SET ist_abgaenger = true, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE lusd_id IS NOT NULL AND NOT (lusd_id = ANY($1)) AND ist_abgaenger = false
		`
		_, err = tx.Exec(ctx, qMarkAbgaenger, lusdIDs)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("diffing update failed: %w", err))
			return
		}

		// 7. Count active borrowed books for all tagged graduates
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
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("counting active loans for graduates failed: %w", err))
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 8. Stream the JSON summary response
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(LUSDImportResponse{
			Neu:                         newCount,
			Aktualisiert:                updatedCount,
			AbgaengerMitOffenenBuechern: abgaengerOpenCount,
		})
	}
}
