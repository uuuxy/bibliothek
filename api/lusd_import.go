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
	"github.com/jackc/pgx/v5"
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

		type studentRow struct {
			LusdID   string
			Vorname  string
			Nachname string
			Klasse   string
			LineNum  int
		}
		var parsedRows []studentRow
		var lusdIDs []string
		var newCount int
		var updatedCount int
		lineNum := 1

		// 5. Parse rows and execute Upserts
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

			lusdIDs = append(lusdIDs, lusdID)
			parsedRows = append(parsedRows, studentRow{
				LusdID:   lusdID,
				Vorname:  vorname,
				Nachname: nachname,
				Klasse:   klasse,
				LineNum:  lineNum,
			})
		}

		// Pre-fetch all existing students
		existingStudents := make(map[string]string)
		if len(lusdIDs) > 0 {
			qSelect := `SELECT id, lusd_id FROM schueler WHERE lusd_id = ANY($1)`
			rows, err := tx.Query(ctx, qSelect, lusdIDs)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to fetch existing students: %w", err))
				return
			}
			defer rows.Close()

			for rows.Next() {
				var id, lusdID string
				if err := rows.Scan(&id, &lusdID); err != nil {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to scan student row: %w", err))
					return
				}
				existingStudents[lusdID] = id
			}
			if err := rows.Err(); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("error iterating existing students: %w", err))
				return
			}
		}

		batch := &pgx.Batch{}

		for _, row := range parsedRows {
			if dbID, exists := existingStudents[row.LusdID]; exists {
				qUpdate := `
					UPDATE schueler
					SET vorname = $1, nachname = $2, klasse = $3, ist_abgaenger = false, aktualisiert_am = CURRENT_TIMESTAMP
					WHERE id = $4
				`
				batch.Queue(qUpdate, row.Vorname, row.Nachname, row.Klasse, dbID)
				updatedCount++
			} else {
				barcodeID := fmt.Sprintf("S-%05d", startNum)
				startNum++

				defaultAbgaengerJahr := time.Now().Year() + 5

				qInsert := `
					INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id, ist_abgaenger)
					VALUES ($1, $2, $3, $4, $5, $6, false)
				`
				batch.Queue(qInsert, barcodeID, row.Vorname, row.Nachname, row.Klasse, defaultAbgaengerJahr, row.LusdID)
				newCount++
			}
		}

		if batch.Len() > 0 {
			br := tx.SendBatch(ctx, batch)
			err = br.Close()
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("batch upsert failed: %w", err))
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
