package api

import (
	"context"
	"encoding/json"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// BorrowedBook represents a currently checked out book copy detail for the student.
type BorrowedBook struct {
	ID             string    `json:"id"`
	AusleiheID     string    `json:"ausleihe_id"`
	BarcodeID      string    `json:"barcode_id"`
	Titel          string    `json:"titel"`
	Autor          string    `json:"autor"`
	CoverURL       string    `json:"cover_url,omitempty"`
	AusgeliehenAm  time.Time `json:"ausgeliehen_am"`
	RueckgabeFrist time.Time `json:"rueckgabe_frist"`
}

// StudentProfileResponse returns master data (with photo_url) and currently borrowed books.
type StudentProfileResponse struct {
	ID                string         `json:"id"`
	BarcodeID         string         `json:"barcode_id"`
	Vorname           string         `json:"vorname"`
	Nachname          string         `json:"nachname"`
	Klasse            string         `json:"klasse"`
	AbgaengerJahr     int            `json:"abgaenger_jahr"`
	IstGesperrt       bool           `json:"ist_gesperrt"`
	FotoURL           string         `json:"foto_url"`
	EntlieheneBuecher []BorrowedBook `json:"entliehene_buecher"`
}

// GetStudentProfileHandler returns a student's master data, passport photo URL (if uploaded),
// and a list of currently borrowed books with their loan and due dates.
// @Summary      Get student profile details
// @Description  Retrieves the complete profile for a student by their ID, including active loans and avatar photo URL if present.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Student ID (UUID)"
// @Success      200  {object}  StudentProfileResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /schueler/{id} [get]
func (s *Server) GetStudentProfileHandler(
	studentRepo repository.StudentRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing student ID parameter"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// 1. Resolve student details from DB
		student, err := studentRepo.GetByID(ctx, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if student == nil {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("student record not found"))
			return
		}

		// 2. Resolve photo URL if the webcam snapshot exists on disk
		fotoURL := ""
		if student.BarcodeID != "" {
			safeBarcodeID := filepath.Base(student.BarcodeID)
			filePath := filepath.Join("uploads", "fotos", fmt.Sprintf("%s.jpg", safeBarcodeID))

			fotoBytes, err := os.ReadFile(filePath)
			if err == nil {
				// Base64 encoden
				encoded := base64.StdEncoding.EncodeToString(fotoBytes)
				fotoURL = fmt.Sprintf("data:image/jpeg;base64,%s", encoded)
			} else {
				// Fallback auf den URL-Pfad (der Client lädt das Bild dann selbst)
				fotoURL = fmt.Sprintf("/uploads/fotos/%s.jpg", safeBarcodeID)
			}
		}

		// 3. Retrieve currently active loans for this student
		query := `
			SELECT 
				e.id, 
				a.id AS ausleihe_id,
				e.barcode_id, 
				t.titel, 
				coalesce(t.autor, ''), 
				coalesce(t.cover_url, ''),
				a.ausgeliehen_am, 
				a.rueckgabe_frist
			FROM ausleihen a
			JOIN buecher_exemplare e ON a.exemplar_id = e.id
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE a.schueler_id = $1 AND a.rueckgabe_am IS NULL
			ORDER BY a.ausgeliehen_am DESC
		`
		rows, err := s.DB.Pool.Query(ctx, query, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		borrowedBooks := make([]BorrowedBook, 0)
		for rows.Next() {
			var b BorrowedBook
			err := rows.Scan(
				&b.ID,
				&b.AusleiheID,
				&b.BarcodeID,
				&b.Titel,
				&b.Autor,
				&b.CoverURL,
				&b.AusgeliehenAm,
				&b.RueckgabeFrist,
			)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			borrowedBooks = append(borrowedBooks, b)
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 4. Construct response and stream as JSON
		resp := StudentProfileResponse{
			ID:                student.ID,
			BarcodeID:         student.BarcodeID,
			Vorname:           student.Vorname,
			Nachname:          student.Nachname,
			Klasse:            student.Klasse,
			AbgaengerJahr:     student.AbgaengerJahr,
			IstGesperrt:       student.IstGesperrt,
			FotoURL:           fotoURL,
			EntlieheneBuecher: borrowedBooks,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// GetClassesHandler returns a list of all distinct classes in the database.
// @Summary      Get list of classes
// @Description  Retrieves all unique school class names currently assigned to students.
// @Tags         students
// @Accept       json
// @Produce      json
// @Success      200  {array}   string
// @Failure      500  {object}  map[string]string
// @Router       /klassen [get]
func (s *Server) GetClassesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		rows, err := s.DB.Pool.Query(ctx, "SELECT DISTINCT klasse FROM schueler WHERE klasse != '' ORDER BY klasse")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		classes := []string{}
		for rows.Next() {
			var k string
			if err := rows.Scan(&k); err == nil {
				classes = append(classes, k)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(classes)
	}
}
