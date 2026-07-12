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

// leseUploadInhalt begrenzt, parst und liest die hochgeladene Datei. Bei Fehler
// wird bereits geantwortet und ok=false zurückgegeben.
func (s *Server) leseUploadInhalt(w http.ResponseWriter, r *http.Request, maxBytes int64) (content []byte, filename string, ok bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return nil, "", false
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return nil, "", false
	}
	defer closeutil.LogClose(file, litteraImportSource)

	content, err = io.ReadAll(file)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, "", false
	}
	return content, fileHeader.Filename, true
}

// istLitteraXML erkennt ein MAB2-XML-Katalogisat an Endung oder Inhalt.
func istLitteraXML(filename, contentStr string) bool {
	lowerName := strings.ToLower(filename)
	lowerContent := strings.ToLower(contentStr)
	return strings.HasSuffix(lowerName, ".xml") ||
		strings.Contains(lowerContent, "<?xml") ||
		strings.Contains(lowerContent, "<katalogisat")
}

// leseTabellarischeDaten liest die Zeilen aus einer XLSX- oder CSV-Datei.
func leseTabellarischeDaten(filename string, content []byte, contentStr string) (rows [][]string, isXLSX bool, err error) {
	if strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		f, err := excelize.OpenReader(bytes.NewReader(content))
		if err != nil {
			return nil, true, fmt.Errorf("failed to open excel file: %w", err)
		}
		defer closeutil.LogClose(f, litteraImportSource)
		sheetName := f.GetSheetName(f.GetActiveSheetIndex())
		rows, err = f.GetRows(sheetName, excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, true, fmt.Errorf("failed to get excel rows: %w", err)
		}
		return rows, true, nil
	}

	reader := csv.NewReader(strings.NewReader(contentStr))
	reader.Comma = detectCSVDelimiter(contentStr)
	reader.LazyQuotes = true
	rows, err = reader.ReadAll()
	if err != nil {
		return nil, false, fmt.Errorf("failed to read csv content: %w", err)
	}
	return rows, false, nil
}

// importHeaderMitPflichtspalten baut die Spaltenzuordnung und prüft die
// Pflichtspalten (Titel, Barcode).
func importHeaderMitPflichtspalten(rows [][]string) (map[string]int, error) {
	if len(rows) < 1 {
		return nil, fmt.Errorf("empty file")
	}
	headerMap := buildImportHeaderMap(rows[0])
	if _, ok := headerMap["titel"]; !ok {
		return nil, fmt.Errorf("missing required column: titel")
	}
	if _, ok := headerMap["barcode"]; !ok {
		return nil, fmt.Errorf("missing required column: barcode/exemplarnummer")
	}
	return headerMap, nil
}

// logImportAudit schreibt einen Audit-Log-Eintrag, sofern ein Nutzer im Kontext ist.
func (s *Server) logImportAudit(r *http.Request, aktion, details string) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		return
	}
	logExec(s.DB.Pool.Exec(r.Context(), "INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse) VALUES ($1, $2, $3::jsonb, $4)", claims.UserID, aktion, details, getIP(r)))
}

// verarbeiteLitteraXML importiert ein MAB2-XML-Katalogisat und antwortet.
func (s *Server) verarbeiteLitteraXML(w http.ResponseWriter, r *http.Request, content []byte) {
	importSvc := service.NewImportService(repository.NewBookRepository(s.DB.Pool), s.DB.Pool)
	importedCount, err := importSvc.ParseLitteraXML(r.Context(), bytes.NewReader(content))
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	s.logImportAudit(r, "LUSD_IMPORT", fmt.Sprintf(`{"updated_titles":%d,"type":"xml"}`, importedCount))

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"imported_count":       importedCount,
		"updated_titles_count": importedCount,
		"type":                 "xml",
		"message":              "MAB2-XML Katalogisat erfolgreich importiert",
	})
}

func (s *Server) LitteraImportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		content, filename, ok := s.leseUploadInhalt(w, r, 100<<20) // 100 MB limit
		if !ok {
			return
		}

		contentStr := string(content)
		if istLitteraXML(filename, contentStr) {
			s.verarbeiteLitteraXML(w, r, content)
			return
		}

		rows, isXLSX, err := leseTabellarischeDaten(filename, content, contentStr)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		headerMap, err := importHeaderMitPflichtspalten(rows)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		importSvc := service.NewImportService(repository.NewBookRepository(s.DB.Pool), s.DB.Pool)
		newTitlesCount, importedCopiesCount, err := importSvc.ImportDynamic(r.Context(), rows, headerMap)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		s.logImportAudit(r, "LUSD_IMPORT", fmt.Sprintf(`{"new_titles":%d,"imported_copies":%d}`, newTitlesCount, importedCopiesCount))

		importType := "csv"
		if isXLSX {
			importType = "xlsx"
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
