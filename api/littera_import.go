package api

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/service"
	"bibliothek/pkg/closeutil"
	"bibliothek/repository"

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
		if err := r.ParseMultipartForm(100 << 20); err != nil { //nolint:gosec // Pre-existing G120
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		defer closeutil.LogClose(file, "littera import file")

		content, err := io.ReadAll(file)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		contentStr := string(content)
		isXML := strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".xml") || strings.Contains(strings.ToLower(contentStr), "<?xml") || strings.Contains(strings.ToLower(contentStr), "<katalogisat")

		bookRepo := repository.NewBookRepository(s.DB.Pool)
		importSvc := service.NewImportService(bookRepo, s.DB.Pool)

		if isXML {
			importedCount, err := importSvc.ParseLitteraXML(r.Context(), bytes.NewReader(content))
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}

			if claims, ok := auth.GetClaims(r.Context()); ok {
				details := fmt.Sprintf(`{"updated_titles":%d,"type":"xml"}`, importedCount)
				logExec(s.DB.Pool.Exec(r.Context(), "INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse) VALUES ($1, $2, $3::jsonb, $4)", claims.UserID, "LUSD_IMPORT", details, getIP(r)))
			}

			response := map[string]interface{}{
				"imported_count":       importedCount,
				"updated_titles_count": importedCount,
				"type":                 "xml",
				"message":              "MAB2-XML Katalogisat erfolgreich importiert",
			}

			RespondJSON(w, http.StatusOK, response)
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
			defer closeutil.LogClose(f, "littera import file")
			sheetName := f.GetSheetName(f.GetActiveSheetIndex())
			rows, err = f.GetRows(sheetName, excelize.Options{RawCellValue: true})
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

		newTitlesCount, importedCopiesCount, err := importSvc.ImportDynamic(r.Context(), rows, headerMap)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Admin audit log
		if claims, ok := auth.GetClaims(r.Context()); ok {
			details := fmt.Sprintf(`{"new_titles":%d,"imported_copies":%d}`, newTitlesCount, importedCopiesCount)
			logExec(s.DB.Pool.Exec(r.Context(), "INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse) VALUES ($1, $2, $3::jsonb, $4)", claims.UserID, "LUSD_IMPORT", details, getIP(r)))
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

// BestandImportHandler verarbeitet den Upload der finalen Bestands-CSV (Semikolon-separiert).
func (s *Server) BestandImportHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	if err := r.ParseMultipartForm(20 << 20); err != nil { //nolint:gosec // Pre-existing G120
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("keine Datei hochgeladen"))
		return
	}
	defer closeutil.LogClose(file, "littera import file")

	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".csv") {
		apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("es werden nur CSV-Dateien akzeptiert"))
		return
	}

	bookRepo := repository.NewBookRepository(s.DB.Pool)
	importSvc := service.NewImportService(bookRepo, s.DB.Pool)

	newTitles, importedCopies, err := importSvc.ImportLitteraBestand(r.Context(), file)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	response := map[string]interface{}{
		"new_titles_count":      newTitles,
		"imported_copies_count": importedCopies,
		"message":               "Bestands-CSV erfolgreich importiert",
	}

	RespondJSON(w, http.StatusOK, response)
}
