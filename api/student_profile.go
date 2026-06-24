package api

import (
	"bibliothek/apierrors"
	"bibliothek/repository"
	"fmt"
	"log"
	"net/http"
)

// StudentProfileResponse returns master data (with photo_url) and currently borrowed books.
type StudentProfileResponse struct {
	ID                string                    `json:"id"`
	BarcodeID         string                    `json:"barcode_id"`
	Vorname           string                    `json:"vorname"`
	Nachname          string                    `json:"nachname"`
	Klasse            string                    `json:"klasse"`
	AbgaengerJahr     int                       `json:"abgaenger_jahr"`
	IstGesperrt       bool                      `json:"ist_gesperrt"`
	FotoURL           string                    `json:"foto_url"`
	Geburtsdatum      *string                   `json:"geburtsdatum,omitempty"`
	LusdID            *string                   `json:"lusd_id,omitempty"`
	Status            string                    `json:"status,omitempty"`
	HasOpenDamages    bool                      `json:"has_open_damages"`
	Strasse           string                    `json:"strasse"`
	Hausnummer        string                    `json:"hausnummer"`
	Plz               string                    `json:"plz"`
	Ort               string                    `json:"ort"`
	ElternEmail       string                    `json:"eltern_email"`
	EntlieheneBuecher []repository.BorrowedBook `json:"entliehene_buecher"`
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
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("missing student ID parameter", nil)
		}

		ctx := r.Context()

		// 1. Resolve student details from DB
		student, err := studentRepo.GetByID(ctx, id)
		if err != nil {
			return apierrors.Internal("Fehler beim Laden des Schülers", err)
		}
		if student == nil {
			return apierrors.NotFound("student record not found", nil)
		}

		// 2. Resolve photo URL if an encrypted photo exists in the DB
		fotoURL := ""
		if student.BarcodeID != "" {
			hasPhoto, err := studentRepo.HasPhoto(ctx, student.ID)
			if err == nil && hasPhoto {
				fotoURL = fmt.Sprintf("/api/schueler/%s/photo", student.BarcodeID)
			}
		}

		// 3. Retrieve currently active loans for this student
		borrowedBooks, err := studentRepo.GetActiveBorrowedBooks(ctx, id)
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der ausgeliehenen Bücher", err)
		}
		if borrowedBooks == nil {
			borrowedBooks = []repository.BorrowedBook{}
		}

		statusStr := "aktiv"
		if student.IstGesperrt {
			statusStr = "gesperrt"
		}
		if student.IstAbgaenger {
			statusStr = "abgaenger"
		}

		// 3.5 Check for open damages
		hasOpenDamages, err := studentRepo.HasOpenDamages(ctx, student.ID)
		if err != nil {
			log.Printf("student-profile: Prüfung auf offene Schadensfälle fehlgeschlagen: %v", err)
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
			HasOpenDamages:    hasOpenDamages,
			Strasse:           student.Strasse,
			Hausnummer:        student.Hausnummer,
			Plz:               student.Plz,
			Ort:               student.Ort,
			ElternEmail:       student.ElternEmail,
			EntlieheneBuecher: borrowedBooks,
		}

		RespondJSON(w, http.StatusOK, resp)
		return nil
	})
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
func (s *Server) GetClassesHandler(studentRepo repository.StudentRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		classes, err := studentRepo.GetDistinctClasses(r.Context())
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Klassen", err)
		}

		RespondJSON(w, http.StatusOK, classes)
		return nil
	})
}
