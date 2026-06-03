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
					// (enthält ggf. personenbezogene Daten – DSGVO-Datensparsamkeit).
				}
			}

			if lusdID == "" || vorname == "" || nachname == "" || klasse == "" {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("empty value on row %d", lineNum))
				return
			}

			lusdIDs = append(lusdIDs, lusdID)

			var dbID string
			err = tx.QueryRow(ctx, "SELECT id FROM schueler WHERE lusd_id = $1 LIMIT 1", lusdID).Scan(&dbID)
			if err == nil {
				// Student exists -> aktualisiere ausschließlich DSGVO-Whitelist-Felder.
				qUpdate := `
					UPDATE schueler
					SET vorname = $1,
					    nachname = $2,
					    klasse = $3,
					    geburtsdatum = $5,
					    ist_abgaenger = false,
					    aktualisiert_am = CURRENT_TIMESTAMP
					WHERE id = $4
				`
				_, err = tx.Exec(ctx, qUpdate, vorname, nachname, klasse, dbID, geburtsdatum)
				if err != nil {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("update failed for LUSD_ID %s on row %d: %w", lusdID, lineNum, err))
					return
				}
				updatedCount++
			} else {
				// Student does not exist -> insert new student with generated S- barcode.
				// Nur Whitelist-Felder werden gespeichert (DSGVO Art. 5 Abs. 1 lit. c).
				barcodeID := fmt.Sprintf("S-%05d", startNum)
				startNum++

				qInsert := `
					INSERT INTO schueler
						(barcode_id, vorname, nachname, klasse, geburtsdatum,
						 abgaenger_jahr, lusd_id, ist_abgaenger)
					VALUES ($1, $2, $3, $4, $5, $6, $7, false)
				`
				_, err = tx.Exec(ctx, qInsert,
					barcodeID, vorname, nachname, klasse, geburtsdatum,
					calculateAbgaengerJahr(klasse), lusdID)
				if err != nil {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("insert failed for LUSD_ID %s on row %d: %w", lusdID, lineNum, err))
					return
				}
				newCount++
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
