package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"bibliothek/apierrors"
	"bibliothek/repository"
	"github.com/jackc/pgx/v5"
)

// BorrowedBook represents a currently checked out book copy detail for the student.
type BorrowedBook struct {
	ID             string    `json:"id"`
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
			filePath := filepath.Join("uploads", "fotos", fmt.Sprintf("%s.jpg", student.BarcodeID))
			if _, err := os.Stat(filePath); err == nil {
				fotoURL = fmt.Sprintf("/uploads/fotos/%s.jpg", student.BarcodeID)
			}
		}

		// 3. Retrieve currently active loans for this student
		query := `
			SELECT 
				e.id, 
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

		var borrowedBooks []BorrowedBook
		for rows.Next() {
			var b BorrowedBook
			err := rows.Scan(
				&b.ID,
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

// ListStudentsHandler returns all students, optionally filtered by klasse.
func (s *Server) ListStudentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		klasse := r.URL.Query().Get("klasse")

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var rows pgx.Rows
		var err error
		if klasse != "" {
			rows, err = s.DB.Pool.Query(ctx, `
				SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt,
					(SELECT COUNT(*) FROM ausleihen a WHERE a.schueler_id = schueler.id AND a.rueckgabe_am IS NULL) as ausgeliehen_anzahl,
					(SELECT COUNT(*) FROM ausleihen a WHERE a.schueler_id = schueler.id AND a.rueckgabe_am IS NULL AND a.rueckgabe_frist < CURRENT_TIMESTAMP) as ueberfaellig_anzahl
				FROM schueler 
				WHERE klasse = $1 
				ORDER BY nachname, vorname
			`, klasse)
		} else {
			rows, err = s.DB.Pool.Query(ctx, `
				SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt,
					(SELECT COUNT(*) FROM ausleihen a WHERE a.schueler_id = schueler.id AND a.rueckgabe_am IS NULL) as ausgeliehen_anzahl,
					(SELECT COUNT(*) FROM ausleihen a WHERE a.schueler_id = schueler.id AND a.rueckgabe_am IS NULL AND a.rueckgabe_frist < CURRENT_TIMESTAMP) as ueberfaellig_anzahl
				FROM schueler 
				ORDER BY klasse, nachname, vorname 
				LIMIT 500
			`)
		}

		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		students := []map[string]any{}
		for rows.Next() {
			var id, barcode, vorname, nachname, kl string
			var abgaengerJahr int
			var gesperrt bool
			var ausgeliehenAnzahl, ueberfaelligAnzahl int
			if err := rows.Scan(&id, &barcode, &vorname, &nachname, &kl, &abgaengerJahr, &gesperrt, &ausgeliehenAnzahl, &ueberfaelligAnzahl); err == nil {
				fotoURL := ""
				if barcode != "" {
					filePath := filepath.Join("uploads", "fotos", fmt.Sprintf("%s.jpg", barcode))
					if _, err := os.Stat(filePath); err == nil {
						fotoURL = fmt.Sprintf("/uploads/fotos/%s.jpg", barcode)
					}
				}
				students = append(students, map[string]any{
					"id":                 id,
					"barcode_id":         barcode,
					"vorname":            vorname,
					"nachname":           nachname,
					"klasse":             kl,
					"abgaenger_jahr":     abgaengerJahr,
					"ist_gesperrt":       gesperrt,
					"ausgeliehen_count":  ausgeliehenAnzahl,
					"ueberfaellig_count": ueberfaelligAnzahl,
					"foto_url":           fotoURL,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(students)
	}
}
