package api

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

type parsedStudentRow struct {
	LusdID      string
	Vorname     string
	Nachname    string
	Klasse      string
	GebDatum    *time.Time
	Strasse     string
	Hausnummer  string
	PLZ         string
	Ort         string
	ElternEmail string
	LineNum     int
}

const (
	lusdColID           = "lusd_id"
	lusdColVorname      = "vorname"
	lusdColNachname     = "nachname"
	lusdColKlasse       = "klasse"
	lusdColGeburtsdatum = "geburtsdatum"
	// Optionale Kontakt-/Adressspalten. Fehlen sie im Export, bleibt der Import
	// gültig; die Felder sind dann leer. Zweck: Schadens-Rechnung (Anschrift) und
	// Eltern-Mahnung (E-Mail). Header müssen exakt (case-insensitiv) so heißen.
	lusdColStrasse     = "strasse"
	lusdColHausnummer  = "hausnummer"
	lusdColPLZ         = "plz"
	lusdColOrt         = "ort"
	lusdColElternEmail = "eltern_email"
)

// Header-Erkennung (Normalisierung, Alias-Tabelle, Pflichtspalten) liegt in
// lusd_header.go.

// spaltenWert liest eine optionale Spalte getrimmt aus; fehlt sie oder ist die
// Zeile zu kurz, wird "" zurückgegeben.
func spaltenWert(row []string, headerMap map[string]int, col string) string {
	idx, ok := headerMap[col]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
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
		LusdID:      lusdID,
		Vorname:     vorname,
		Nachname:    nachname,
		Klasse:      klasse,
		GebDatum:    geburtsdatum,
		Strasse:     spaltenWert(row, headerMap, lusdColStrasse),
		Hausnummer:  spaltenWert(row, headerMap, lusdColHausnummer),
		PLZ:         spaltenWert(row, headerMap, lusdColPLZ),
		Ort:         spaltenWert(row, headerMap, lusdColOrt),
		ElternEmail: spaltenWert(row, headerMap, lusdColElternEmail),
		LineNum:     lineNum,
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
