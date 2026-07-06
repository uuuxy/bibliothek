package api

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/httpresp"
)

// ImportStudentsHandler handles CSV file uploads for importing student records.
// Supports comma and semicolon delimiters. Updates classes of existing students (UPSERT)
// and registers new students within an ACID transaction.
func (s *Server) ImportStudentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Limit post request size to max 5MB
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
		defer closeutil.LogClose(file, "import upload")

		content, err := io.ReadAll(file)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Detect CSV delimiter (semicolon is default in German Excel exports)
		delimiter := ','
		contentStr := string(content)
		if strings.Count(contentStr, ";") > strings.Count(contentStr, ",") {
			delimiter = ';'
		}

		reader := csv.NewReader(strings.NewReader(contentStr))
		reader.Comma = delimiter
		reader.LazyQuotes = true

		// Parse column headers
		headers, err := reader.Read()
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		headerMap := make(map[string]int)
		for idx, h := range headers {
			headerMap[strings.ToLower(strings.TrimSpace(h))] = idx
		}

		// Validate that required columns exist
		requiredCols := []string{"barcode_id", "vorname", "nachname", "klasse", "abgaenger_jahr"}
		for _, col := range requiredCols {
			if _, exists := headerMap[col]; !exists {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("missing required column '%s'", col))
				return
			}
		}

		ctx := r.Context()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer db.SafeRollback(ctx, tx)

		type studentRecord struct {
			Vorname       string
			Nachname      string
			Klasse        string
			AbgaengerJahr int32
		}

		dedupMap := make(map[string]studentRecord)
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

			barcodeID := strings.TrimSpace(row[headerMap["barcode_id"]])
			vorname := strings.TrimSpace(row[headerMap["vorname"]])
			nachname := strings.TrimSpace(row[headerMap["nachname"]])
			klasse := strings.TrimSpace(row[headerMap["klasse"]])
			abgaengerJahrStr := strings.TrimSpace(row[headerMap["abgaenger_jahr"]])

			if barcodeID == "" || vorname == "" || nachname == "" || klasse == "" || abgaengerJahrStr == "" {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("empty value on row %d", lineNum))
				return
			}

			abgaengerJahr, err := strconv.Atoi(abgaengerJahrStr)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("invalid graduation year '%s' on row %d: %w", abgaengerJahrStr, lineNum, err))
				return
			}

			dedupMap[barcodeID] = studentRecord{
				Vorname:       vorname,
				Nachname:      nachname,
				Klasse:        klasse,
				AbgaengerJahr: int32(abgaengerJahr),
			}
		}

		var barcodeIDs []string
		var vornamen []string
		var nachnamen []string
		var klassen []string
		var abgaengerJahre []int32

		for barcodeID, rec := range dedupMap {
			barcodeIDs = append(barcodeIDs, barcodeID)
			vornamen = append(vornamen, rec.Vorname)
			nachnamen = append(nachnamen, rec.Nachname)
			klassen = append(klassen, rec.Klasse)
			abgaengerJahre = append(abgaengerJahre, rec.AbgaengerJahr)
		}

		count := len(barcodeIDs)
		if count > 0 {
			upsertQuery := `
				INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
				SELECT * FROM UNNEST($1::text[], $2::text[], $3::text[], $4::text[], $5::int[])
				ON CONFLICT (barcode_id) DO UPDATE
				SET klasse = EXCLUDED.klasse,
				    abgaenger_jahr = EXCLUDED.abgaenger_jahr,
				    aktualisiert_am = CURRENT_TIMESTAMP
			`
			_, err = tx.Exec(ctx, upsertQuery, barcodeIDs, vornamen, nachnamen, klassen, abgaengerJahre)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("database error during bulk import: %w", err))
				return
			}
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		httpresp.Write(w, []byte(fmt.Sprintf(`{"status":"success","processed":%d}`, count)))
	}
}
