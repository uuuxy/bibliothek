package api

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

type parsedStudentRow struct {
	LusdID   string
	Vorname  string
	Nachname string
	Klasse   string
	GebDatum *time.Time
	LineNum  int
}

// parseLUSDCSV parses the LUSD CSV content and extracts valid student records.
func parseLUSDCSV(content []byte) ([]parsedStudentRow, []string, error) {
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
		return nil, nil, fmt.Errorf("fehler beim lesen der csv-kopfzeile: %w", err)
	}

	const (
		colLUSDID       = "lusd_id"
		colVorname      = "vorname"
		colNachname     = "nachname"
		colKlasse       = "klasse"
		colGeburtsdatum = "geburtsdatum"
	)

	headerMap := make(map[string]int)
	for idx, h := range headers {
		norm := strings.ToLower(strings.TrimSpace(h))
		switch norm {
		case colLUSDID, colVorname, colNachname, colKlasse, colGeburtsdatum:
			headerMap[norm] = idx
		}
	}

	requiredCols := []string{colLUSDID, colVorname, colNachname, colKlasse}
	for _, col := range requiredCols {
		if _, exists := headerMap[col]; !exists {
			return nil, nil, fmt.Errorf("Pflichtspalte '%s' fehlt in der CSV-Kopfzeile — ist das die richtige LUSD-Exportdatei?", col)
		}
	}

	var parsedRows []parsedStudentRow
	var lusdIDs []string
	seenIndex := make(map[string]int)
	lineNum := 1

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		lineNum++
		if err != nil {
			return nil, nil, fmt.Errorf("Zeile %d der CSV-Datei ist nicht lesbar: %w", lineNum, err)
		}

		lusdID := strings.TrimSpace(row[headerMap[colLUSDID]])
		vorname := strings.TrimSpace(row[headerMap[colVorname]])
		nachname := strings.TrimSpace(row[headerMap[colNachname]])
		klasse := strings.TrimSpace(row[headerMap[colKlasse]])

		var geburtsdatum *time.Time
		if idx, ok := headerMap[colGeburtsdatum]; ok && idx < len(row) {
			if raw := strings.TrimSpace(row[idx]); raw != "" {
				for _, layout := range []string{"02.01.2006", "2006-01-02", "01/02/2006"} {
					if t, parseErr := time.ParseInLocation(layout, raw, time.UTC); parseErr == nil {
						t2 := t
						geburtsdatum = &t2
						break
					}
				}
			}
		}

		if vorname == "" || nachname == "" || klasse == "" {
			return nil, nil, fmt.Errorf("Zeile %d enthält ein leeres Pflichtfeld (Vorname/Nachname/Klasse)", lineNum)
		}

		sRow := parsedStudentRow{
			LusdID:   lusdID,
			Vorname:  vorname,
			Nachname: nachname,
			Klasse:   klasse,
			GebDatum: geburtsdatum,
			LineNum:  lineNum,
		}

		if lusdID != "" {
			if idx, exists := seenIndex[lusdID]; exists {
				parsedRows[idx] = sRow
				continue
			}
			seenIndex[lusdID] = len(parsedRows)
			lusdIDs = append(lusdIDs, lusdID)
		}
		parsedRows = append(parsedRows, sRow)
	}

	return parsedRows, lusdIDs, nil
}
