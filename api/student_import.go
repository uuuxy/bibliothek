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
	"github.com/jackc/pgx/v5"
)

// ImportStudentsLUSDHandler handles LUSD-compliant CSV uploads for admins.
func (s *Server) ImportStudentsLUSDHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Parse Multipart Form with MaxBytesReader
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
		defer func() { _ = tx.Rollback(ctx) }()

		// Fetch all existing students into maps to prevent N+1 queries
		type existingStudent struct {
			ID        string
			LusdID    *string
			BarcodeID string
			Vorname   string
			Nachname  string
		}

		rows, err := tx.Query(ctx, "SELECT id, lusd_id, barcode_id, vorname, nachname FROM schueler")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		lusdMap := make(map[string]string)
		barcodeMap := make(map[string]string)
		nameMap := make(map[string]string)

		for rows.Next() {
			var s existingStudent
			if err := rows.Scan(&s.ID, &s.LusdID, &s.BarcodeID, &s.Vorname, &s.Nachname); err != nil {
				rows.Close()
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			if s.LusdID != nil {
				lusdMap[*s.LusdID] = s.ID
			}
			if s.BarcodeID != "" {
				barcodeMap[s.BarcodeID] = s.ID
			}
			nameKey := strings.ToLower(s.Vorname) + "|" + strings.ToLower(s.Nachname)
			nameMap[nameKey] = s.ID
		}
		rows.Close()

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
			re := regexp.MustCompile(`S-(\d+)`)
			matches := re.FindStringSubmatch(lastBarcode)
			if len(matches) > 1 {
				if parsed, err := strconv.Atoi(matches[1]); err == nil {
					startNum = parsed + 1
				}
			}
		}

		importedCount := 0
		lineNum := 1

		type updateData struct {
			ID     string
			Klasse string
			LusdID *string
		}

		type insertData struct {
			BarcodeID      string
			Vorname        string
			Nachname       string
			Klasse         string
			AbgaengerJahr  int
			LusdID         *string
		}

		var updates []updateData
		var inserts []insertData

		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			lineNum++
			if err != nil {
				_ = tx.Rollback(ctx)
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("fehler in Zeile %d: %w", lineNum, err))
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
				if id, ok := lusdMap[*lusdID]; ok {
					existingID = id
					found = true
				}
			}

			// 2. Try by barcodeID
			if !found && barcodeID != "" {
				if id, ok := barcodeMap[barcodeID]; ok {
					existingID = id
					found = true
				}
			}

			// 3. Try by Name combination
			if !found {
				nameKey := strings.ToLower(vorname) + "|" + strings.ToLower(nachname)
				if id, ok := nameMap[nameKey]; ok {
					existingID = id
					found = true
				}
			}

			if found {
				updates = append(updates, updateData{
					ID:     existingID,
					Klasse: klasse,
					LusdID: lusdID,
				})
			} else {
				// Generate new barcode if empty
				if barcodeID == "" {
					barcodeID = fmt.Sprintf("S-%05d", startNum)
					startNum++
				}

				defaultAbgaengerJahr := time.Now().Year() + 5
				inserts = append(inserts, insertData{
					BarcodeID:     barcodeID,
					Vorname:       vorname,
					Nachname:      nachname,
					Klasse:        klasse,
					AbgaengerJahr: defaultAbgaengerJahr,
					LusdID:        lusdID,
				})
			}
			importedCount++
		}

		if len(updates) > 0 {
			var ids []string
			var klassen []string
			var lusdIDs []*string
			for _, u := range updates {
				ids = append(ids, u.ID)
				klassen = append(klassen, u.Klasse)
				lusdIDs = append(lusdIDs, u.LusdID)
			}

			qUpdate := `
				UPDATE schueler s
				SET klasse = data.klasse,
					lusd_id = COALESCE(data.lusd_id, s.lusd_id),
					aktualisiert_am = CURRENT_TIMESTAMP
				FROM (
					SELECT unnest($1::uuid[]) AS id,
						   unnest($2::text[]) AS klasse,
						   unnest($3::text[]) AS lusd_id
				) AS data
				WHERE s.id = data.id
			`
			_, err = tx.Exec(ctx, qUpdate, ids, klassen, lusdIDs)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim Aktualisieren: %w", err))
				return
			}
		}

		if len(inserts) > 0 {
			var copyRows [][]any
			for _, i := range inserts {
				copyRows = append(copyRows, []any{
					i.BarcodeID, i.Vorname, i.Nachname, i.Klasse, i.AbgaengerJahr, i.LusdID,
				})
			}
			_, err = tx.CopyFrom(
				ctx,
				pgx.Identifier{"schueler"},
				[]string{"barcode_id", "vorname", "nachname", "klasse", "abgaenger_jahr", "lusd_id"},
				pgx.CopyFromRows(copyRows),
			)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim Einfügen: %w", err))
				return
			}
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
