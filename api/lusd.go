package api

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pkg/closeutil"
)

// LusdPreviewResult contains the statistics for the frontend preview.
type LusdPreviewResult struct {
	NewStudents     int `json:"new_students"`
	ClassChanges    int `json:"class_changes"`
	Graduates       int `json:"graduates"` // Abgänger (missing in CSV but in DB)
	TotalCsvRecords int `json:"total_csv_records"`
}

// lusdRecord represents a parsed row from the CSV.
type lusdRecord struct {
	LusdID       string
	Vorname      string
	Nachname     string
	Klasse       string
	Geburtsdatum string
}

// parseLusdCSV reads the multipart file and extracts the relevant fields.
// It uses case-insensitive matching to find the column indices for LUSD-ID, Vorname, Nachname, Klasse, and Geburtsdatum.
func parseLusdCSV(r *http.Request) ([]lusdRecord, error) {
	file, _, err := r.FormFile("csvFile")
	if err != nil {
		return nil, fmt.Errorf("CSV-Datei fehlt: %w", err)
	}
	defer closeutil.LogClose(file, "lusd upload")

	reader := csv.NewReader(file)
	reader.Comma = ';' // Typical for German CSVs, fallback to ',' below if needed
	reader.LazyQuotes = true
	// Read header
	header, err := reader.Read()
	// Try comma if semicolon failed to produce multiple columns
	if err != nil || len(header) < 3 {
		if _, serr := file.Seek(0, io.SeekStart); serr != nil {
			return nil, fmt.Errorf("CSV konnte für erneutes Einlesen nicht zurückgespult werden: %w", serr)
		}
		reader = csv.NewReader(file)
		reader.Comma = ','
		reader.LazyQuotes = true
		header, err = reader.Read()
		if err != nil {
			return nil, fmt.Errorf("fehler beim Lesen der CSV-Kopfzeile: %w", err)
		}
	}

	// Map headers to indices
	colIdx := map[string]int{"id": -1, "vorname": -1, "nachname": -1, "klasse": -1, "geburtsdatum": -1}
	for i, h := range header {
		lowerH := strings.ToLower(strings.TrimSpace(h))
		if strings.Contains(lowerH, "id") && colIdx["id"] == -1 {
			colIdx["id"] = i
		} else if strings.Contains(lowerH, "vorname") && colIdx["vorname"] == -1 {
			colIdx["vorname"] = i
		} else if strings.Contains(lowerH, "name") && !strings.Contains(lowerH, "vorname") && colIdx["nachname"] == -1 {
			colIdx["nachname"] = i
		} else if strings.Contains(lowerH, "klasse") && colIdx["klasse"] == -1 {
			colIdx["klasse"] = i
		} else if strings.Contains(lowerH, "geburt") && colIdx["geburtsdatum"] == -1 {
			colIdx["geburtsdatum"] = i
		}
	}

	if colIdx["id"] == -1 || colIdx["vorname"] == -1 || colIdx["nachname"] == -1 {
		return nil, errors.New("fehlende Pflichtspalten in der CSV. Benötigt: ID, Vorname, Name")
	}

	records := make([]lusdRecord, 0)
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // Skip bad rows
		}
		rec := lusdRecord{
			LusdID:   strings.TrimSpace(row[colIdx["id"]]),
			Vorname:  strings.TrimSpace(row[colIdx["vorname"]]),
			Nachname: strings.TrimSpace(row[colIdx["nachname"]]),
		}
		if colIdx["klasse"] != -1 && len(row) > colIdx["klasse"] {
			rec.Klasse = strings.TrimSpace(row[colIdx["klasse"]])
		}
		if colIdx["geburtsdatum"] != -1 && len(row) > colIdx["geburtsdatum"] {
			rec.Geburtsdatum = strings.TrimSpace(row[colIdx["geburtsdatum"]])
			// Convert DD.MM.YYYY to YYYY-MM-DD for postgres
			if parts := strings.Split(rec.Geburtsdatum, "."); len(parts) == 3 {
				rec.Geburtsdatum = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
			}
		}
		if rec.LusdID != "" && rec.Vorname != "" && rec.Nachname != "" {
			records = append(records, rec)
		}
	}
	return records, nil
}

// computeLusdChanges compares the CSV records with the database inside a transaction
// and either returns the preview stats or actually applies the changes.
func (s *Server) computeLusdChanges(ctx context.Context, records []lusdRecord, apply bool) (*LusdPreviewResult, error) {
	// Start transaction. Everything is done within a single TX to ensure atomicity.
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	// Always rollback on panic or early return. If we apply, we commit at the very end.
	defer db.SafeRollback(ctx, tx)

	// Read all current students. deleted_at IS NULL ist zwingend: sonst „matcht"
	// eine soft-gelöschte, aber noch auf der LUSD-Liste stehende Zeile, der aktive
	// Schüler würde nie neu angelegt und bliebe unsichtbar. Re-Insert einer zuvor
	// gelöschten lusd_id ist dank partiellem Unique-Index (Migration 035) sicher.
	rows, err := tx.Query(ctx, "SELECT id, lusd_id, klasse FROM schueler WHERE ist_abgaenger = false AND deleted_at IS NULL")
	if err != nil {
		return nil, err
	}
	dbStudents := make(map[string]struct {
		ID     string
		Klasse string
	})
	for rows.Next() {
		var id string
		var lusdID *string
		var klasse string
		if err := rows.Scan(&id, &lusdID, &klasse); err != nil {
			rows.Close()
			return nil, err
		}
		if lusdID != nil && *lusdID != "" {
			dbStudents[*lusdID] = struct {
				ID     string
				Klasse string
			}{id, klasse}
		}
	}
	rows.Close()

	res := &LusdPreviewResult{TotalCsvRecords: len(records)}
	csvLusdIDs := make(map[string]bool)

	// Process CSV records (Updates and Inserts)
	for _, rec := range records {
		csvLusdIDs[rec.LusdID] = true
		if dbRec, exists := dbStudents[rec.LusdID]; exists {
			if dbRec.Klasse != rec.Klasse {
				res.ClassChanges++
				if apply {
					_, err := tx.Exec(ctx, "UPDATE schueler SET klasse = $1, aktualisiert_am = NOW() WHERE id = $2", rec.Klasse, dbRec.ID)
					if err != nil {
						return nil, err
					}
				}
			}
		} else {
			res.NewStudents++
			if apply {
				barcode := fmt.Sprintf("S-%05d%04d", time.Now().Unix()%100000, time.Now().Nanosecond()%10000) // Temporary unique short barcode
				year := time.Now().Year() + 5                                                                 // Default abgang
				geb := interface{}(rec.Geburtsdatum)
				if rec.Geburtsdatum == "" {
					geb = nil
				}
				_, err := tx.Exec(ctx,
					"INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id, geburtsdatum) VALUES ($1, $2, $3, $4, $5, $6, $7)",
					barcode, rec.Vorname, rec.Nachname, rec.Klasse, year, rec.LusdID, geb)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// Process missing records (Abgänger)
	for lusdID, dbRec := range dbStudents {
		if !csvLusdIDs[lusdID] {
			res.Graduates++
			if apply {
				// Check if they have active loans
				var pending int
				err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL", dbRec.ID).Scan(&pending)
				if err != nil {
					return nil, err
				}
				if pending > 0 {
					// Mark as inactive/abgänger
					_, err = tx.Exec(ctx, "UPDATE schueler SET ist_abgaenger = true, ist_gesperrt = true, aktualisiert_am = NOW() WHERE id = $1", dbRec.ID)
				} else {
					// DSGVO compliant anonymization
					_, err = tx.Exec(ctx, "UPDATE schueler SET vorname = 'Abgänger', nachname = 'Anonymisiert', klasse = 'ABG', ist_abgaenger = true, ist_gesperrt = true, aktualisiert_am = NOW() WHERE id = $1", dbRec.ID)
				}
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if apply {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// PostLusdPreviewHandler parses the CSV and returns a preview of changes.
func (s *Server) PostLusdPreviewHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		records, err := parseLusdCSV(r)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		res, err := s.computeLusdChanges(r.Context(), records, false)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, res)
	}
}

// PostLusdImportHandler parses the CSV and applies the changes transactionally.
func (s *Server) PostLusdImportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		records, err := parseLusdCSV(r)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		res, err := s.computeLusdChanges(r.Context(), records, true)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, res)
	}
}
