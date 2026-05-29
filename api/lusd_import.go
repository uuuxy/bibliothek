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
		if err := r.ParseMultipartForm(5 << 20); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		defer file.Close()

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

		headerMap := make(map[string]int)
		for idx, h := range headers {
			headerMap[strings.ToLower(strings.TrimSpace(h))] = idx
		}

		// Validate required headers
		requiredCols := []string{"lusd_id", "vorname", "nachname", "klasse"}
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
		defer tx.Rollback(ctx)

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

		var lusdIDs []string
		var newCount int
		var updatedCount int
		lineNum := 1

		type studentRow struct {
			LusdID   string
			Vorname  string
			Nachname string
			Klasse   string
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

			lusdID := strings.TrimSpace(row[headerMap["lusd_id"]])
			vorname := strings.TrimSpace(row[headerMap["vorname"]])
			nachname := strings.TrimSpace(row[headerMap["nachname"]])
			klasse := strings.TrimSpace(row[headerMap["klasse"]])

			if lusdID == "" || vorname == "" || nachname == "" || klasse == "" {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("empty value on row %d", lineNum))
				return
			}

			sRow := studentRow{
				LusdID:   lusdID,
				Vorname:  vorname,
				Nachname: nachname,
				Klasse:   klasse,
				LineNum:  lineNum,
			}

			if idx, exists := seenIndex[lusdID]; exists {
				// Replace the existing one, don't append to lusdIDs again
				parsedRows[idx] = sRow
			} else {
				seenIndex[lusdID] = len(parsedRows)
				parsedRows = append(parsedRows, sRow)
				lusdIDs = append(lusdIDs, lusdID)
			}
		}

		if len(parsedRows) > 0 {
			// 5b. Find existing LUSD IDs
			existingSet := make(map[string]bool)
			rows, err := tx.Query(ctx, "SELECT lusd_id FROM schueler WHERE lusd_id = ANY($1)", lusdIDs)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to query existing students: %w", err))
				return
			}
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err == nil {
					existingSet[id] = true
				}
			}
			rows.Close()

			// Prepare arrays for UNNEST
			var (
				arrBarcode []string
				arrVorname []string
				arrNach    []string
				arrKlasse  []string
				arrAbJahr  []int
				arrLusdID  []string
			)

			defaultAbgaengerJahr := time.Now().Year() + 5

			for _, p := range parsedRows {
				if !existingSet[p.LusdID] {
					// new
					arrBarcode = append(arrBarcode, fmt.Sprintf("S-%05d", startNum))
					startNum++
					newCount++
				} else {
					// existing, barcode will be ignored on update but needs a non-null valid dummy
					// we can just put a blank string here, but to avoid UNIQUE constraint violations
					// before the ON CONFLICT kicks in, we just use the LUSD ID or a dummy with random/unique part.
					// Actually, the ON CONFLICT will evaluate before inserting, but it evaluates the index on lusd_id.
					// We'll just provide a dummy string that is unique per row
					arrBarcode = append(arrBarcode, fmt.Sprintf("dummy-%s", p.LusdID))
					updatedCount++
				}
				arrVorname = append(arrVorname, p.Vorname)
				arrNach = append(arrNach, p.Nachname)
				arrKlasse = append(arrKlasse, p.Klasse)
				arrAbJahr = append(arrAbJahr, defaultAbgaengerJahr)
				arrLusdID = append(arrLusdID, p.LusdID)
			}

			// 5c. Bulk Upsert
			qUpsert := `
				INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id, ist_abgaenger)
				SELECT * FROM UNNEST($1::varchar[], $2::varchar[], $3::varchar[], $4::varchar[], $5::int[], $6::varchar[], $7::boolean[])
				ON CONFLICT (lusd_id) DO UPDATE
				SET vorname = EXCLUDED.vorname,
					nachname = EXCLUDED.nachname,
					klasse = EXCLUDED.klasse,
					ist_abgaenger = false,
					aktualisiert_am = CURRENT_TIMESTAMP
			`

			// We need a boolean array of false for ist_abgaenger
			arrIstAbg := make([]bool, len(parsedRows))

			_, err = tx.Exec(ctx, qUpsert, arrBarcode, arrVorname, arrNach, arrKlasse, arrAbJahr, arrLusdID, arrIstAbg)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("bulk upsert failed: %w", err))
				return
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
