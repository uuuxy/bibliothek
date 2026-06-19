package inventur

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// handleExportCSV handles the GET /api/admin/books/export route
func (handler *APIHandler) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := handler.repo.FetchAllBooksForCSVExport(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fehler beim datenbank-export")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="bestand_export_%s.csv"`, time.Now().Format("2006-01-02")))

	// Write UTF-8 BOM so Excel opens it correctly with UTF-8
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(w)
	writer.Comma = ';' // German Excel standard

	err = writer.WriteAll(rows)
	if err != nil {
		// Cannot write error response since headers are already sent
		return
	}
}

// FetchAllBooksForCSVExport queries all books and their copies
func (repo *BookRepository) FetchAllBooksForCSVExport(ctx context.Context) ([][]string, error) {
	query := `
		SELECT 
			bt.titel, 
			coalesce(bt.autor, ''), 
			coalesce(bt.verlag, ''), 
			coalesce(bt.isbn, ''), 
			coalesce(bt.erscheinungsjahr, 0), 
			coalesce(bt.subject, ''), 
			coalesce(be.barcode_id, ''),
			coalesce(be.zustand_notiz, '')
		FROM buecher_titel bt
		LEFT JOIN buecher_exemplare be ON bt.id = be.titel_id AND be.ist_ausgesondert = false
		ORDER BY bt.titel, be.barcode_id;
	`

	pgRows, err := repo.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer pgRows.Close()

	var csvData [][]string
	csvData = append(csvData, []string{"Titel", "Autor", "Verlag", "ISBN", "Jahr", "Kategorie", "Barcode", "Zustand"})

	for pgRows.Next() {
		var titel, autor, verlag, isbn, subject, barcode, zustand string
		var jahr int

		if err := pgRows.Scan(&titel, &autor, &verlag, &isbn, &jahr, &subject, &barcode, &zustand); err != nil {
			return nil, err
		}

		jahrStr := ""
		if jahr > 0 {
			jahrStr = strconv.Itoa(jahr)
		}

		// Prefix ISBN with a single quote so Excel treats it as text and doesn't remove leading zeros
		if isbn != "" {
			isbn = "'" + isbn
		}

		csvData = append(csvData, []string{
			titel, autor, verlag, isbn, jahrStr, subject, barcode, zustand,
		})
	}

	return csvData, pgRows.Err()
}
