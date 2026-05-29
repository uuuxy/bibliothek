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

	"github.com/jackc/pgx/v5"

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

		type csvRow struct {
			lusdID   string
			vorname  string
			nachname string
			klasse   string
			lineNum  int
		}
		var parsedRows []csvRow

		// 5. Parse rows and collect them
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
			parsedRows = append(parsedRows, csvRow{
				lusdID:   lusdID,
				vorname:  vorname,
				nachname: nachname,
				klasse:   klasse,
				lineNum:  lineNum,
			})
		}

		// 5a. Fetch all existing lusd_ids
		existingMap := make(map[string]bool)
		if len(lusdIDs) > 0 {
			qExisting := `SELECT lusd_id FROM schueler WHERE lusd_id = ANY($1)`
			rows, err := tx.Query(ctx, qExisting, lusdIDs)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to fetch existing students: %w", err))
				return
			}
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err == nil {
					existingMap[id] = true
				}
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("error iterating existing students: %w", err))
				return
			}
		}

		// 5b. Queue batched updates and inserts
		batch := &pgx.Batch{}
		qUpdate := `
			UPDATE schueler
			SET vorname = $1, nachname = $2, klasse = $3, ist_abgaenger = false, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE lusd_id = $4
		`
		qInsert := `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id, ist_abgaenger)
			VALUES ($1, $2, $3, $4, $5, $6, false)
		`
		defaultAbgaengerJahr := time.Now().Year() + 5

		for _, r := range parsedRows {
			if existingMap[r.lusdID] {
				batch.Queue(qUpdate, r.vorname, r.nachname, r.klasse, r.lusdID)
				updatedCount++
			} else {
				barcodeID := fmt.Sprintf("S-%05d", startNum)
				startNum++
				batch.Queue(qInsert, barcodeID, r.vorname, r.nachname, r.klasse, defaultAbgaengerJahr, r.lusdID)
				newCount++
			}
		}

		if batch.Len() > 0 {
			br := tx.SendBatch(ctx, batch)
			for i := 0; i < batch.Len(); i++ {
				_, err := br.Exec()
				if err != nil {
					br.Close()
					apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("batch execution failed at index %d: %w", i, err))
					return
				}
			}
			err = br.Close()
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to close batch: %w", err))
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
