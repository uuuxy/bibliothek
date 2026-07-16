package inventur

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"bibliothek/pkg/safego"

	"github.com/xuri/excelize/v2"
)

func extractImportRows(w http.ResponseWriter, request *http.Request) ([][]string, error) {
	request.Body = http.MaxBytesReader(w, request.Body, 100<<20)
	if err := request.ParseMultipartForm(100 << 20); err != nil { // 100 MB
		return nil, errors.New("datei zu groß oder ungültig")
	}

	file, fileHeader, err := request.FormFile("file")
	if err != nil {
		return nil, errors.New("keine datei gefunden")
	}
	defer func() { _ = file.Close() }() //nolint:errcheck

	if strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".csv") {
		return parseCSVRows(file)
	}
	return parseExcelRows(file)
}

// parseCSVRows liest eine CSV-Datei mit heuristischer Trennzeichenerkennung (, oder ;).
func parseCSVRows(file io.Reader) ([][]string, error) {
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.New("fehler beim lesen der csv-datei")
	}
	contentStr := string(content)
	delimiter := ','
	if strings.Count(contentStr, ";") > strings.Count(contentStr, ",") {
		delimiter = ';'
	}
	reader := csv.NewReader(strings.NewReader(contentStr))
	reader.Comma = delimiter
	reader.LazyQuotes = true
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, errors.New("ungültige csv-datei")
	}
	if len(rows) == 0 {
		return nil, errors.New("datei ist leer")
	}
	return rows, nil
}

// parseExcelRows liest die erste Tabelle einer Excel-Datei.
func parseExcelRows(file io.Reader) ([][]string, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, errors.New("ungültige excel-datei")
	}
	defer func() { _ = f.Close() }() //nolint:errcheck

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
		errateSpaltenAusInhalt(header, colIdx)
	}
	return colIdx, hasHeader
}

// errateSpaltenAusInhalt rät die Spaltenzuordnung anhand des Zeileninhalts, wenn keine
// Kopfzeile erkannt wurde: ISBN am 978/979-Präfix, reine Zahlen als Bestand, längere
// Texte als Titel. Mutiert colIdx.
func errateSpaltenAusInhalt(header []string, colIdx map[string]int) {
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
		// Semaphore VOR dem Start der Goroutine belegen, damit die Schleife blockiert und
		// nicht für jede Zeile sofort eine Goroutine erzeugt. Bei großen Importen (z. B. 20k
		// Zeilen) entstünden sonst zehntausende gleichzeitig lebende Goroutinen, die alle nur
		// auf einen Semaphor-Slot warten — unnötiger Speicher- und Scheduler-Druck.
		sem <- struct{}{}
		go func(row []string) {
			defer wg.Done()
			defer func() { <-sem }()
			defer safego.Guard("excel-import-zeile")

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

// prepareImportRows validiert Methode und Datei, ermittelt die Spaltenzuordnung und
// liefert die Datenzeilen (ohne Kopfzeile). ok=false bedeutet: die Fehlerantwort wurde
// bereits geschrieben.
func (handler *APIHandler) prepareImportRows(writer http.ResponseWriter, request *http.Request) (dataRows [][]string, colIdx map[string]int, ok bool) {
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "nur post-anfragen erlaubt")
		return nil, nil, false
	}

	rows, err := extractImportRows(writer, request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, err.Error())
		return nil, nil, false
	}

	colIdx, hasHeader := determineColumnIndices(rows[0])
	if colIdx["isbn"] == -1 {
		writeError(writer, http.StatusBadRequest, "spalte 'isbn' fehlt in der datei. bitte stellen sie sicher, dass eine isbn-spalte vorhanden ist.")
		return nil, nil, false
	}

	dataRows = rows
	if hasHeader {
		dataRows = rows[1:]
	}

	const maxImportRows = 100000
	if len(dataRows) > maxImportRows {
		writeError(writer, http.StatusBadRequest, fmt.Sprintf("zu viele zeilen (%d), maximal %d erlaubt", len(dataRows), maxImportRows))
		return nil, nil, false
	}

	return dataRows, colIdx, true
}

// persistImportedBooks schreibt die eingelesenen Bücher per Batch-Upsert; schlägt der
// Batch fehl, wird zeilenweise als Fallback eingefügt. Liefert die aktualisierten
// Zähler zurück.
func (handler *APIHandler) persistImportedBooks(ctx context.Context, booksToUpsert []Book, failed int32, firstError error) (imported, outFailed int32, outErr error) {
	if len(booksToUpsert) == 0 {
		return 0, failed, firstError
	}

	count, err := handler.repo.UpsertBooksBatch(ctx, booksToUpsert)
	if err != nil {
		// Fallback: Einzelne Inserts, wenn der Batch-Insert fehlgeschlägt (z.B. wegen Constraint-Fehlern bei einem Buch)
		for _, book := range booksToUpsert {
			_, singleErr := handler.repo.UpsertBook(ctx, book)
			if singleErr != nil {
				failed++
				if firstError == nil {
					firstError = singleErr
				}
			} else {
				imported++
			}
		}
		return imported, failed, firstError
	}

	// #nosec G115 - count is bounded by maxImportRows (5000)
	imported += int32(count)
	if imported == 0 {
		// Falls count 0 ist wegen z.B. nur Updates in manchen DB Versionen
		// #nosec G115 - len is bounded by maxImportRows (5000)
		imported = int32(len(booksToUpsert))
	}
	return imported, failed, firstError
}

func (handler *APIHandler) handleImportExcel(writer http.ResponseWriter, request *http.Request) {
	dataRows, colIdx, ok := handler.prepareImportRows(writer, request)
	if !ok {
		return
	}

	booksToUpsert, failed, firstError := handler.processImportRows(request.Context(), dataRows, colIdx)
	imported, failed, firstError := handler.persistImportedBooks(request.Context(), booksToUpsert, failed, firstError)

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
