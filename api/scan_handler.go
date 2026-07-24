package api

import (
	"bibliothek/pkg/httpresp"
	"context"
	"errors"
	"net/http"

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

		ctx := r.Context()
		w.Header().Set("Content-Type", "application/json")

		// 1. Check if the barcode belongs to a student
		studentResp, err := s.tryScanStudent(ctx, barcode)
		if err != nil {
			http.Error(w, `{"error": "Database error while searching for student"}`, http.StatusInternalServerError)
			return
		}
		if studentResp != nil {
			httpresp.Encode(w, studentResp)
			return
		}

		// 2. Check if the barcode belongs to a book
		bookResp, err := s.tryScanBook(ctx, barcode)
		if err != nil {
			http.Error(w, `{"error": "Database error while searching for book"}`, http.StatusInternalServerError)
			return
		}
		if bookResp != nil {
			httpresp.Encode(w, bookResp)
			return
		}

		// 3. Fallback: Not found
		w.WriteHeader(http.StatusNotFound)
		httpresp.Encode(w, map[string]string{
			"error": "Barcode im System nicht gefunden",
		})
	}
}

func (s *Server) tryScanStudent(ctx context.Context, barcode string) (map[string]any, error) {
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
		WHERE (barcode_id = $1 OR lusd_id = $1) AND deleted_at IS NULL
		LIMIT 1
	`, barcode).Scan(&student.ID, &student.BarcodeID, &student.Vorname, &student.Nachname, &student.Klasse)

	if err == nil {
		return map[string]any{
			"type":    "student",
			"student": student,
		}, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // Not found
	}

	return nil, err // DB error
}

func (s *Server) tryScanBook(ctx context.Context, barcode string) (map[string]any, error) {
	var book struct {
		ID        string `json:"id"`
		TitelID   string `json:"titel_id"`
		BarcodeID string `json:"barcode_id"`
		Titel     string `json:"titel"`
		Autor     string `json:"autor"`
	}
	var currentStudentID *string
	var currentStudentBarcode *string

	err := s.DB.Pool.QueryRow(ctx, `
		SELECT e.id, e.titel_id, e.barcode_id, t.titel, COALESCE(t.autor, ''),
		       (SELECT schueler_id FROM ausleihen WHERE exemplar_id = e.id AND rueckgabe_am IS NULL LIMIT 1) as current_student_id,
		       (SELECT s.barcode_id FROM ausleihen a JOIN schueler s ON a.schueler_id = s.id WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL AND s.deleted_at IS NULL LIMIT 1) as current_student_barcode
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE e.barcode_id = $1
		LIMIT 1
	`, barcode).Scan(&book.ID, &book.TitelID, &book.BarcodeID, &book.Titel, &book.Autor, &currentStudentID, &currentStudentBarcode)

	if err == nil {
		status := "available"
		if currentStudentID != nil {
			status = "lent"
		}
		return map[string]any{
			"type":                    "book",
			"book":                    book,
			"status":                  status,
			"current_student_id":      currentStudentID,
			"current_student_barcode": currentStudentBarcode,
		}, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // Not found
	}

	return nil, err // DB error
}
