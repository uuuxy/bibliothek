package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"bibliothek/apierrors"
)

// LitteraImportResponse matches the JSON response structure.
type LitteraImportResponse struct {
	NewTitles      int `json:"new_titles_count"`
	ImportedCopies int `json:"imported_copies_count"`
}

// LitteraImportHandler parses LITTERA CSV exports and upserts books and copies.
func (s *Server) LitteraImportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Limit multipart form size to max 10MB
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
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

		headers, err := reader.Read()
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		// Map LITTERA columns flexibly
		headerMap := make(map[string]int)
		for idx, h := range headers {
			norm := strings.ToLower(strings.TrimSpace(h))
			// Handle common LITTERA header aliases
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

		var newTitlesCount int
		var importedCopiesCount int
		lineNum := 1

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

			// Helper to safely get column by key
			getCol := func(key string) string {
				if idx, ok := headerMap[key]; ok && idx < len(row) {
					return strings.TrimSpace(row[idx])
				}
				return ""
			}

			titel := getCol("titel")
			barcode := getCol("barcode")
			if titel == "" || barcode == "" {
				continue // Skip empty rows
			}

			autor := getCol("autor")
			verlag := getCol("verlag")
			isbn := strings.ReplaceAll(getCol("isbn"), "-", "")
			jahrStr := getCol("jahr")
			kategorie := getCol("kategorie")

			var jahr int
			if j, err := strconv.Atoi(jahrStr); err == nil {
				jahr = j
			}

			// 1. Resolve Title ID (By ISBN or exact Title)
			var titelID string
			qFindTitel := `
				SELECT id FROM buecher_titel
				WHERE (isbn = $1 AND $1 != '') OR (titel = $2)
				LIMIT 1
			`
			err = tx.QueryRow(ctx, qFindTitel, isbn, titel).Scan(&titelID)

			// 2. Insert if not exists
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					qInsertTitel := `
						INSERT INTO buecher_titel (titel, autor, verlag, isbn, erscheinungsjahr, subject)
						VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, 0), $6)
						RETURNING id
					`
					// We use NULLIF because ISBN might violate unique constraint if empty string
					err = tx.QueryRow(ctx, qInsertTitel, titel, autor, verlag, isbn, jahr, kategorie).Scan(&titelID)
					if err != nil {
						apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to insert title '%s' on line %d: %w", titel, lineNum, err))
						return
					}
					newTitlesCount++
				} else {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to query title: %w", err))
					return
				}
			}

			// 3. Insert Copy (Exemplar)
			// wir nutzen ON CONFLICT DO NOTHING, falls der Barcode schon existiert (doppelter Import-Schutz)
			qInsertExemplar := `
				INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am)
				VALUES ($1, $2, CURRENT_DATE)
				ON CONFLICT (barcode_id) DO NOTHING
				RETURNING id
			`
			var exemplarID string
			err = tx.QueryRow(ctx, qInsertExemplar, titelID, barcode).Scan(&exemplarID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					// Duplicate barcode ignored gracefully
				} else {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to insert copy for barcode '%s' on line %d: %w", barcode, lineNum, err))
					return
				}
			} else {
				importedCopiesCount++
			}
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(LitteraImportResponse{
			NewTitles:      newTitlesCount,
			ImportedCopies: importedCopiesCount,
		})
	}
}
