package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"

	"github.com/jackc/pgx/v5"
	"github.com/xuri/excelize/v2"
)

type LitteraImportResponse struct {
	NewTitles      int    `json:"new_titles_count,omitempty"`
	ImportedCopies int    `json:"imported_copies_count,omitempty"`
	UpdatedTitles  int    `json:"updated_titles_count,omitempty"`
	Type           string `json:"type"` // "csv", "xml" or "xlsx"
}

func (s *Server) LitteraImportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 100<<20) // 100 MB limit
		if err := r.ParseMultipartForm(100 << 20); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		file, fileHeader, err := r.FormFile("file")
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

		contentStr := string(content)
		isXML := strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".xml") || strings.Contains(contentStr, "<?xml") || strings.Contains(contentStr, "<katalogisat")

		if isXML {
			s.handleLitteraXMLImport(w, r, content)
			return
		}

		var rows [][]string
		var isXLSX bool
		if strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".xlsx") {
			isXLSX = true
			f, err := excelize.OpenReader(bytes.NewReader(content))
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("failed to open excel file: %w", err))
				return
			}
			defer f.Close()
			sheetName := f.GetSheetName(f.GetActiveSheetIndex())
			rows, err = f.GetRows(sheetName)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("failed to get excel rows: %w", err))
				return
			}
		} else {
			delimiter := ','
			if strings.Count(contentStr, ";") > strings.Count(contentStr, ",") {
				delimiter = ';'
			}
			reader := csv.NewReader(strings.NewReader(contentStr))
			reader.Comma = delimiter
			reader.LazyQuotes = true
			rows, err = reader.ReadAll()
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("failed to read csv content: %w", err))
				return
			}
		}

		if len(rows) < 1 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("empty file"))
			return
		}

		headers := rows[0]
		headerMap := make(map[string]int)
		for idx, h := range headers {
			norm := strings.ToLower(strings.TrimSpace(h))
			switch {
			case strings.Contains(norm, "titel") || norm == "titelliste":
				headerMap["titel"] = idx
			case strings.Contains(norm, "autor") || norm == "verfasser":
				headerMap["autor"] = idx
			case strings.Contains(norm, "verlag"):
				headerMap["verlag"] = idx
			case strings.Contains(norm, "isbn"):
				headerMap["isbn"] = idx
			case strings.Contains(norm, "jahr") || norm == "ersch.jahr" || norm == "erscheinungsjahr":
				headerMap["jahr"] = idx
			case strings.Contains(norm, "kategorie") || strings.Contains(norm, "systematik") || norm == "fach":
				headerMap["kategorie"] = idx
			case strings.Contains(norm, "barcode") || strings.Contains(norm, "exemplar") || norm == "signatur" || norm == "inventarnummer":
				headerMap["barcode"] = idx
			}
		}

		if _, ok := headerMap["titel"]; !ok {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("missing required column: titel"))
			return
		}
		if _, ok := headerMap["barcode"]; !ok {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("missing required column: barcode/exemplarnummer"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
		defer cancel()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		// Preload existing titles for fast mapping
		dbRows, err := tx.Query(ctx, "SELECT id, coalesce(isbn, ''), titel FROM buecher_titel")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		isbnToID := make(map[string]string)
		titelToID := make(map[string]string)
		for dbRows.Next() {
			var id, isbn, titel string
			if err := dbRows.Scan(&id, &isbn, &titel); err == nil {
				if isbn != "" {
					isbnToID[isbn] = id
				}
				titelToID[titel] = id
			}
		}
		dbRows.Close()

		type NewTitle struct {
			Titel     string
			Autor     string
			Verlag    string
			ISBN      string
			Jahr      int
			Kategorie string
		}

		type CopyData struct {
			TitelID string
			Barcode string
		}

		newTitlesMap := make(map[string]*NewTitle) // key: isbn or titel
		var newTitlesOrder []string

		for _, row := range rows[1:] {
			getCol := func(key string) string {
				if idx, ok := headerMap[key]; ok && idx < len(row) {
					return strings.TrimSpace(row[idx])
				}
				return ""
			}

			titel := getCol("titel")
			barcode := getCol("barcode")
			if titel == "" || barcode == "" {
				continue
			}

			isbn := strings.ReplaceAll(getCol("isbn"), "-", "")

			titelID := ""
			if isbn != "" && isbnToID[isbn] != "" {
				titelID = isbnToID[isbn]
			} else if titelToID[titel] != "" {
				titelID = titelToID[titel]
			}

			if titelID == "" {
				// Needs new title
				cacheKey := isbn
				if cacheKey == "" {
					cacheKey = titel
				}
				if _, exists := newTitlesMap[cacheKey]; !exists {
					var jahr int
					if j, err := strconv.Atoi(getCol("jahr")); err == nil {
						jahr = j
					}
					newTitlesMap[cacheKey] = &NewTitle{
						Titel:     titel,
						Autor:     getCol("autor"),
						Verlag:    getCol("verlag"),
						ISBN:      isbn,
						Jahr:      jahr,
						Kategorie: getCol("kategorie"),
					}
					newTitlesOrder = append(newTitlesOrder, cacheKey)
				}
			}
		}

		var newTitlesCount int
		// Insert new titles using batch
		if len(newTitlesOrder) > 0 {
			batch := &pgx.Batch{}
			qInsertTitel := `
				INSERT INTO buecher_titel (titel, autor, verlag, isbn, erscheinungsjahr, subject)
				VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, 0), $6)
				RETURNING id
			`
			for _, key := range newTitlesOrder {
				t := newTitlesMap[key]
				batch.Queue(qInsertTitel, t.Titel, t.Autor, t.Verlag, t.ISBN, t.Jahr, t.Kategorie)
			}

			br := tx.SendBatch(ctx, batch)
			for _, key := range newTitlesOrder {
				var insertedID string
				err := br.QueryRow().Scan(&insertedID)
				if err != nil {
					br.Close()
					apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to insert title batch: %w", err))
					return
				}
				t := newTitlesMap[key]
				if t.ISBN != "" {
					isbnToID[t.ISBN] = insertedID
				}
				titelToID[t.Titel] = insertedID
				newTitlesCount++
			}
			br.Close()
		}

		// Second pass: Now all titles have IDs, collect all copies again
		var copiesToInsert []CopyData

		for _, row := range rows[1:] {
			getCol := func(key string) string {
				if idx, ok := headerMap[key]; ok && idx < len(row) {
					return strings.TrimSpace(row[idx])
				}
				return ""
			}
			titel := getCol("titel")
			barcode := getCol("barcode")
			if titel == "" || barcode == "" {
				continue
			}
			isbn := strings.ReplaceAll(getCol("isbn"), "-", "")

			titelID := ""
			if isbn != "" && isbnToID[isbn] != "" {
				titelID = isbnToID[isbn]
			} else if titelToID[titel] != "" {
				titelID = titelToID[titel]
			}

			if titelID != "" {
				copiesToInsert = append(copiesToInsert, CopyData{TitelID: titelID, Barcode: barcode})
			}
		}

		var importedCopiesCount int
		// Insert copies using batch ON CONFLICT DO NOTHING
		if len(copiesToInsert) > 0 {
			batchCopies := &pgx.Batch{}
			qInsertExemplar := `
				INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am)
				VALUES ($1, $2, CURRENT_DATE)
				ON CONFLICT (barcode_id) DO NOTHING
				RETURNING id
			`
			for _, c := range copiesToInsert {
				batchCopies.Queue(qInsertExemplar, c.TitelID, c.Barcode)
			}

			bcr := tx.SendBatch(ctx, batchCopies)
			for i := 0; i < len(copiesToInsert); i++ {
				var id string
				err := bcr.QueryRow().Scan(&id)
				if err == nil {
					importedCopiesCount++
				}
			}
			bcr.Close()
		}

		// Admin audit log
		if claims, ok := auth.GetClaims(r.Context()); ok {
			details := fmt.Sprintf(`{"new_titles":%d,"imported_copies":%d}`, newTitlesCount, importedCopiesCount)
			_, _ = tx.Exec(ctx, "INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse) VALUES ($1, $2, $3::jsonb, $4)", claims.UserID, "LUSD_IMPORT", details, r.RemoteAddr)
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		var importType string
		if isXLSX {
			importType = "xlsx"
		} else {
			importType = "csv"
		}

		RespondJSON(w, http.StatusOK, LitteraImportResponse{
			NewTitles:      newTitlesCount,
			ImportedCopies: importedCopiesCount,
			Type:           importType,
		})
	}
}
