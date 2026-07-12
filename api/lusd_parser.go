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

const (
	lusdColID           = "lusd_id"
	lusdColVorname      = "vorname"
	lusdColNachname     = "nachname"
	lusdColKlasse       = "klasse"
	lusdColGeburtsdatum = "geburtsdatum"
)

// lusdHeaderMap ordnet die bekannten Spalten ihren Indizes zu und prüft, ob alle
// Pflichtspalten (LUSD-ID, Vorname, Nachname, Klasse) vorhanden sind.
func lusdHeaderMap(headers []string) (map[string]int, error) {
	headerMap := make(map[string]int)
	for idx, h := range headers {
		norm := strings.ToLower(strings.TrimSpace(h))
		switch norm {
		case lusdColID, lusdColVorname, lusdColNachname, lusdColKlasse, lusdColGeburtsdatum:
			headerMap[norm] = idx
		}
	}

	for _, col := range []string{lusdColID, lusdColVorname, lusdColNachname, lusdColKlasse} {
		if _, exists := headerMap[col]; !exists {
			return nil, fmt.Errorf("pflichtspalte '%s' fehlt in der CSV-Kopfzeile — ist das die richtige LUSD-Exportdatei?", col)
		}
	}
	return headerMap, nil
}

// parseLUSDGebDatum liest das optionale Geburtsdatum und probiert mehrere Layouts.
func parseLUSDGebDatum(row []string, headerMap map[string]int) *time.Time {
	idx, ok := headerMap[lusdColGeburtsdatum]
	if !ok || idx >= len(row) {
		return nil
	}
	raw := strings.TrimSpace(row[idx])
	if raw == "" {
		return nil
	}
	for _, layout := range []string{dateFormatDE, dateFormatISO, "01/02/2006"} {
		if t, parseErr := time.ParseInLocation(layout, raw, time.UTC); parseErr == nil {
			t2 := t
			return &t2
		}
	}
	return nil
}

// parseLUSDRow parst eine Datenzeile und validiert die Pflichtfelder.
func parseLUSDRow(row []string, headerMap map[string]int, lineNum int) (parsedStudentRow, error) {
	lusdID := strings.TrimSpace(row[headerMap[lusdColID]])
	vorname := strings.TrimSpace(row[headerMap[lusdColVorname]])
	nachname := strings.TrimSpace(row[headerMap[lusdColNachname]])
	klasse := strings.TrimSpace(row[headerMap[lusdColKlasse]])
	geburtsdatum := parseLUSDGebDatum(row, headerMap)

	if vorname == "" || nachname == "" || klasse == "" {
		return parsedStudentRow{}, fmt.Errorf("zeile %d enthält ein leeres Pflichtfeld (Vorname/Nachname/Klasse)", lineNum)
	}

	return parsedStudentRow{
		LusdID:   lusdID,
		Vorname:  vorname,
		Nachname: nachname,
		Klasse:   klasse,
		GebDatum: geburtsdatum,
		LineNum:  lineNum,
	}, nil
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

	headerMap, err := lusdHeaderMap(headers)
	if err != nil {
		return nil, nil, err
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
			return nil, nil, fmt.Errorf("zeile %d der CSV-Datei ist nicht lesbar: %w", lineNum, err)
		}

		sRow, err := parseLUSDRow(row, headerMap, lineNum)
		if err != nil {
			return nil, nil, err
		}

		// Duplikate innerhalb der Datei: späterer Eintrag überschreibt den früheren.
		if sRow.LusdID != "" {
			if idx, exists := seenIndex[sRow.LusdID]; exists {
				parsedRows[idx] = sRow
				continue
			}
			seenIndex[sRow.LusdID] = len(parsedRows)
			lusdIDs = append(lusdIDs, sRow.LusdID)
		}
		parsedRows = append(parsedRows, sRow)
	}

	return parsedRows, lusdIDs, nil
}
