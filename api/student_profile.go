package api

import (
	"bibliothek/apierrors"
	"bibliothek/repository"
	"errors"
	"fmt"
	"net/http"
	"time"
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
	Geburtsdatum      *string        `json:"geburtsdatum,omitempty"`
	LusdID            *string        `json:"lusd_id,omitempty"`
	Status            string         `json:"status,omitempty"`
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

		ctx := r.Context()

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

		// 2. Resolve photo URL if an encrypted photo exists in the DB
		fotoURL := ""
		if student.BarcodeID != "" {
			var hasPhoto bool
			err := s.DB.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schueler_fotos WHERE schueler_id = $1)", student.ID).Scan(&hasPhoto)
			if err == nil && hasPhoto {
				fotoURL = fmt.Sprintf("/api/schueler/%s/photo", student.BarcodeID)
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

		statusStr := "aktiv"
		if student.IstGesperrt {
			statusStr = "gesperrt"
		}
		if student.IstAbgaenger {
			statusStr = "abgaenger"
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
			Geburtsdatum:      student.Geburtsdatum,
			LusdID:            student.LusdID,
			Status:            statusStr,
			EntlieheneBuecher: borrowedBooks,
		}

		RespondJSON(w, http.StatusOK, resp)
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
		ctx := r.Context()

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

		RespondJSON(w, http.StatusOK, classes)
	}
}
