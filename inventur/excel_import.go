package inventur

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/xuri/excelize/v2"
)

func extractExcelRows(w http.ResponseWriter, request *http.Request) ([][]string, error) {
	request.Body = http.MaxBytesReader(w, request.Body, 100<<20)
	err := request.ParseMultipartForm(100 << 20) // 100 MB
	if err != nil {
		return nil, errors.New("datei zu groß oder ungültig")
	}

	file, _, err := request.FormFile("file")
	if err != nil {
		return nil, errors.New("keine datei gefunden")
	}
	defer func() { _ = file.Close() }()

	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, errors.New("ungültige excel-datei")
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("excel-datei ist leer")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil || len(rows) < 1 {
		return nil, errors.New("keine daten gefunden")
	}

	return rows, nil
}

func determineColumnIndices(header []string) (map[string]int, bool) {
	colIdx := map[string]int{"isbn": -1, "titel": -1, "autor": -1, "fach": -1, "klasse": -1, "bestand": -1}
	hasHeader := true

	// Versuche Header zu erkennen
	for i, col := range header {
		if field := mapHeaderToField(col); field != "" {
			colIdx[field] = i
		}
	}

	// Fallback: Wenn kein Header erkannt wurde (keine ISBN Spalte), versuche Inhalt zu erraten
	if colIdx["isbn"] == -1 {
		hasHeader = false
		// Check if first row is ISBN
		for i, col := range header {
			val := strings.TrimSpace(col)
			cleanVal := strings.ReplaceAll(val, "-", "")
			if (strings.HasPrefix(val, "978") || strings.HasPrefix(val, "979")) && len(cleanVal) >= 10 {
				colIdx["isbn"] = i
				continue
			}
			if _, err := strconv.Atoi(val); err == nil {
				if colIdx["bestand"] == -1 {
					colIdx["bestand"] = i
				}
				continue
			}
			if colIdx["titel"] == -1 && len(val) > 2 {
				colIdx["titel"] = i
			}
		}
	}
	return colIdx, hasHeader
}

func (handler *APIHandler) processImportRows(ctx context.Context, dataRows [][]string, colIdx map[string]int) ([]Book, int32, error) {
	var failed int32 = 0
	var firstError error
	var errMutex sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	var booksToUpsert []Book
	var booksMutex sync.Mutex

	for _, row := range dataRows {
		wg.Add(1)
		go func(row []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			book, err := verarbeiteImportZeile(ImportConfig{
				Ctx:       ctx,
				Row:       row,
				ColIdx:    colIdx,
				Repo:      handler.repo,
				Metadaten: handler.metadaten,
			})
			if err != nil {
				atomic.AddInt32(&failed, 1)
				errMutex.Lock()
				if firstError == nil {
					firstError = err
				}
				errMutex.Unlock()
			} else if book != nil {
				booksMutex.Lock()
				booksToUpsert = append(booksToUpsert, *book)
				booksMutex.Unlock()
			}
		}(row)
	}
	wg.Wait()

	return booksToUpsert, failed, firstError
}

func (handler *APIHandler) handleImportExcel(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "nur post-anfragen erlaubt")
		return
	}

	rows, err := extractExcelRows(writer, request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return
	}

	colIdx, hasHeader := determineColumnIndices(rows[0])

	if colIdx["isbn"] == -1 {
		writeError(writer, http.StatusBadRequest, "spalte 'isbn' konnte nicht gefunden werden")
		return
	}

	dataRows := rows
	if hasHeader {
		dataRows = rows[1:]
	}

	const maxImportRows = 100000
	if len(dataRows) > maxImportRows {
		writeError(writer, http.StatusBadRequest, fmt.Sprintf("zu viele zeilen (%d), maximal %d erlaubt", len(dataRows), maxImportRows))
		return
	}

	var imported int32 = 0
	booksToUpsert, failed, firstError := handler.processImportRows(request.Context(), dataRows, colIdx)

	if len(booksToUpsert) > 0 {
		count, err := handler.repo.UpsertBooksBatch(request.Context(), booksToUpsert)
		if err != nil {
			// Fallback: Einzelne Inserts, wenn der Batch-Insert fehlgeschlägt (z.B. wegen Constraint-Fehlern bei einem Buch)
			for _, book := range booksToUpsert {
				_, singleErr := handler.repo.UpsertBook(request.Context(), book)
				if singleErr != nil {
					failed++
					if firstError == nil {
						firstError = singleErr
					}
				} else {
					imported++
				}
			}
		} else {
			// #nosec G115 - count is bounded by maxImportRows (5000)
			imported += int32(count)
			if imported == 0 {
				// Falls count 0 ist wegen z.B. nur Updates in manchen DB Versionen
				// #nosec G115 - len is bounded by maxImportRows (5000)
				imported = int32(len(booksToUpsert))
			}
		}
	}

	if imported == 0 && failed > 0 {
		msg := "keine bücher konnten importiert werden."
		if firstError != nil {
			msg += " fehler: " + firstError.Error()
		}
		writeError(writer, http.StatusBadRequest, msg)
		return
	}

	message := fmt.Sprintf("%d bücher importiert", imported)
	if failed > 0 {
		message += fmt.Sprintf(", %d fehlgeschlagen", failed)
	}

	writeJSON(writer, http.StatusOK, map[string]any{
		"message":  message,
		"imported": imported,
		"failed":   failed,
	})
}
