package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bibliothek/apierrors"
)

// ImportStudentsLUSDHandler handles LUSD-compliant CSV uploads for admins.
func (s *Server) ImportStudentsLUSDHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Parse Multipart Form
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

		// Hessen LUSD standard CSV uses semicolon (;)
		reader := csv.NewReader(strings.NewReader(string(content)))
		reader.Comma = ';'
		reader.LazyQuotes = true

		headers, err := reader.Read()
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("CSV-Header konnte nicht gelesen werden: %w", err))
			return
		}

		headerMap := make(map[string]int)
		for idx, h := range headers {
			headerMap[strings.ToLower(strings.TrimSpace(h))] = idx
		}

		// Resolve column indexes
		getColIdx := func(keys []string) int {
			for _, k := range keys {
				if idx, ok := headerMap[k]; ok {
					return idx
				}
			}
			return -1
		}

		lusdIDIdx := getColIdx([]string{"lusd_id", "schueler_id", "id", "lusd-id", "schüler-id", "schüler_id", "schuelerid", "schülerid", "lusd id", "schüler id", "schueler id"})
		vornameIdx := getColIdx([]string{"vorname", "first_name", "firstname", "rufname"})
		nachnameIdx := getColIdx([]string{"nachname", "last_name", "lastname", "name", "familienname"})
		klasseIdx := getColIdx([]string{"klasse", "class", "jahrgang", "klassenbezeichnung"})
		barcodeIdx := getColIdx([]string{"barcode_id", "barcode", "barcode-id"})

		// Validation
		if vornameIdx == -1 || nachnameIdx == -1 || klasseIdx == -1 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("CSV muss mindestens die Spalten 'Vorname', 'Nachname' und 'Klasse' enthalten"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
		defer cancel()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback(ctx)

		// Get next barcode sequence S-XXXXX helper
		var lastBarcode string
		qLast := `
			SELECT barcode_id 
			FROM schueler 
			WHERE barcode_id LIKE 'S-%' 
			ORDER BY barcode_id DESC 
			LIMIT 1
			FOR UPDATE
		`
		err = tx.QueryRow(ctx, qLast).Scan(&lastBarcode)
		startNum := 10001
		if err == nil {
			re := regexp.MustCompile(`^S-(\d{5,})$`)
			matches := re.FindStringSubmatch(lastBarcode)
			if len(matches) > 1 {
				if parsed, err := strconv.Atoi(matches[1]); err == nil {
					startNum = parsed + 1
				}
			}
		}

		importedCount := 0
		lineNum := 1

		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			lineNum++
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("Fehler in Zeile %d: %w", lineNum, err))
				return
			}

			if len(row) <= vornameIdx || len(row) <= nachnameIdx || len(row) <= klasseIdx {
				continue
			}

			vorname := strings.TrimSpace(row[vornameIdx])
			nachname := strings.TrimSpace(row[nachnameIdx])
			klasse := strings.TrimSpace(row[klasseIdx])

			if vorname == "" || nachname == "" || klasse == "" {
				continue // Skip invalid rows
			}

			var lusdID *string
			if lusdIDIdx != -1 && len(row) > lusdIDIdx {
				val := strings.TrimSpace(row[lusdIDIdx])
				if val != "" {
					lusdID = &val
				}
			}

			var barcodeID string
			if barcodeIdx != -1 && len(row) > barcodeIdx {
				barcodeID = strings.TrimSpace(row[barcodeIdx])
			}

			// Try to find student
			var existingID string
			found := false

			// 1. Try by lusdID
			if lusdID != nil {
				err = tx.QueryRow(ctx, "SELECT id FROM schueler WHERE lusd_id = $1 LIMIT 1", *lusdID).Scan(&existingID)
				if err == nil {
					found = true
				}
			}

			// 2. Try by barcodeID
			if !found && barcodeID != "" {
				err = tx.QueryRow(ctx, "SELECT id FROM schueler WHERE barcode_id = $1 LIMIT 1", barcodeID).Scan(&existingID)
				if err == nil {
					found = true
				}
			}

			// 3. Try by Name combination
			if !found {
				err = tx.QueryRow(ctx, "SELECT id FROM schueler WHERE lower(vorname) = lower($1) AND lower(nachname) = lower($2) LIMIT 1", vorname, nachname).Scan(&existingID)
				if err == nil {
					found = true
				}
			}

			if found {
				// Update student's class (Versetzung)
				qUpdate := `
					UPDATE schueler 
					SET klasse = $1, aktualisiert_am = CURRENT_TIMESTAMP
				`
				params := []any{klasse}
				paramCount := 2

				if lusdID != nil {
					qUpdate += fmt.Sprintf(", lusd_id = $%d", paramCount)
					params = append(params, *lusdID)
					paramCount++
				}

				qUpdate += fmt.Sprintf(" WHERE id = $%d", paramCount)
				params = append(params, existingID)

				_, err = tx.Exec(ctx, qUpdate, params...)
				if err != nil {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
					return
				}
			} else {
				// Generate new barcode if empty
				if barcodeID == "" {
					barcodeID = fmt.Sprintf("S-%05d", startNum)
					startNum++
				}

				defaultAbgaengerJahr := time.Now().Year() + 5
				qInsert := `
					INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id)
					VALUES ($1, $2, $3, $4, $5, $6)
				`
				_, err = tx.Exec(ctx, qInsert, barcodeID, vorname, nachname, klasse, defaultAbgaengerJahr, lusdID)
				if err != nil {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
					return
				}
			}
			importedCount++
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   "success",
			"imported": importedCount,
		})
	}
}
