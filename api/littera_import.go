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
		if err := r.ParseMultipartForm(100 << 20); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		defer closeutil.LogClose(file, litteraImportSource)

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
			defer closeutil.LogClose(f, litteraImportSource)
			sheetName := f.GetSheetName(f.GetActiveSheetIndex())
			rows, err = f.GetRows(sheetName, excelize.Options{RawCellValue: true})
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("failed to get excel rows: %w", err))
				return
			}
		} else {
			reader := csv.NewReader(strings.NewReader(contentStr))
			reader.Comma = detectCSVDelimiter(contentStr)
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

		headerMap := buildImportHeaderMap(rows[0])
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

// BestandImportHandler verarbeitet den Upload der finalen Bestands-CSV.
//
// Er nutzt denselben robusten Pfad wie der Littera-Import (Trennzeichen-Erkennung,
// namensbasiertes Spalten-Mapping, Titel-Dedup über ImportDynamic). Dadurch ist es
// egal, ob die Datei Komma- oder Semikolon-getrennt ist bzw. die Zustand-Spalte
// enthält — ein Formatfehler des Nutzers wird als 400 (nicht 500) gemeldet.
func (s *Server) BestandImportHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("keine Datei hochgeladen"))
		return
	}
	defer closeutil.LogClose(file, litteraImportSource)

	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".csv") {
		apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("es werden nur CSV-Dateien akzeptiert"))
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	contentStr := string(content)

	reader := csv.NewReader(strings.NewReader(contentStr))
	reader.Comma = detectCSVDelimiter(contentStr)
	reader.LazyQuotes = true
	rows, err := reader.ReadAll()
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("CSV konnte nicht gelesen werden: %w", err))
		return
	}
	if len(rows) < 2 {
		apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("die CSV enthält keine Datenzeilen"))
		return
	}

	headerMap := buildImportHeaderMap(rows[0])
	if _, ok := headerMap["titel"]; !ok {
		apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("pflichtspalte fehlt: Titel"))
		return
	}
	if _, ok := headerMap["barcode"]; !ok {
		apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("pflichtspalte fehlt: Barcode/Exemplarnummer"))
		return
	}

	bookRepo := repository.NewBookRepository(s.DB.Pool)
	importSvc := service.NewImportService(bookRepo, s.DB.Pool)

	newTitles, importedCopies, err := importSvc.ImportDynamic(r.Context(), rows, headerMap)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	if claims, ok := auth.GetClaims(r.Context()); ok {
		details := fmt.Sprintf(`{"new_titles":%d,"imported_copies":%d}`, newTitles, importedCopies)
		logExec(s.DB.Pool.Exec(r.Context(), "INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse) VALUES ($1, $2, $3::jsonb, $4)", claims.UserID, "BESTAND_IMPORT", details, getIP(r)))
	}

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"new_titles_count":      newTitles,
		"imported_copies_count": importedCopies,
		"message":               "Bestands-CSV erfolgreich importiert",
	})
}
