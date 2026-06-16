package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// SmartScanHandler implements GET /api/scan?barcode={code}
// It serves as a unified routing endpoint for barcode scans at the library desk.
func (s *Server) SmartScanHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		barcode := r.URL.Query().Get("barcode")
		if barcode == "" {
			http.Error(w, `{"error": "Barcode parameter is required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		// 1. Check if the barcode belongs to a student
		// We check barcode_id and optionally lusd_id
		var student struct {
			ID        string `json:"id"`
			BarcodeID string `json:"barcode_id"`
			Vorname   string `json:"vorname"`
			Nachname  string `json:"nachname"`
			Klasse    string `json:"klasse"`
		}

		err := s.DB.Pool.QueryRow(ctx, `
			SELECT id, barcode_id, vorname, nachname, klasse 
			FROM schueler 
			WHERE barcode_id = $1 OR lusd_id = $1 
			LIMIT 1
		`, barcode).Scan(&student.ID, &student.BarcodeID, &student.Vorname, &student.Nachname, &student.Klasse)

		if err == nil {
			// Student found
			json.NewEncoder(w).Encode(map[string]any{
				"type":    "student",
				"student": student,
			})
			return
		} else if err != pgx.ErrNoRows {
			http.Error(w, `{"error": "Database error while searching for student"}`, http.StatusInternalServerError)
			return
		}

		// 2. Check if the barcode belongs to a book
		var book struct {
			ID        string `json:"id"`
			TitelID   string `json:"titel_id"`
			BarcodeID string `json:"barcode_id"`
			Titel     string `json:"titel"`
			Autor     string `json:"autor"`
		}
		var currentStudentID *string

		err = s.DB.Pool.QueryRow(ctx, `
			SELECT e.id, e.titel_id, e.barcode_id, t.titel, COALESCE(t.autor, ''),
			       (SELECT schueler_id FROM ausleihen WHERE exemplar_id = e.id AND rueckgabe_am IS NULL LIMIT 1) as current_student_id
			FROM buecher_exemplare e
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE e.barcode_id = $1
			LIMIT 1
		`, barcode).Scan(&book.ID, &book.TitelID, &book.BarcodeID, &book.Titel, &book.Autor, &currentStudentID)

		if err == nil {
			// Book found
			status := "available"
			if currentStudentID != nil {
				status = "lent"
			}
			json.NewEncoder(w).Encode(map[string]any{
				"type":               "book",
				"book":               book,
				"status":             status,
				"current_student_id": currentStudentID,
			})
			return
		} else if err != pgx.ErrNoRows {
			http.Error(w, `{"error": "Database error while searching for book"}`, http.StatusInternalServerError)
			return
		}

		// 3. Fallback: Not found
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Barcode im System nicht gefunden",
		})
	}
}
