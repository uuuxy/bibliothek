package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// calculateAbgaengerJahr errechnet das voraussichtliche Abgangsjahr eines Schülers
// anhand der Klassenbezeichnung (z. B. "5a", "9h", "10r", "11", "13").
//
// Regeln:
//   - Hauptschule (Suffix "h"): Abschluss nach Klasse 9
//   - Oberstufe (Klassenstufe >= 11): Abschluss nach Klasse 13
//   - Alle übrigen (Realschule "r", Gymnasium "g", unmarkiert): Abschluss nach Klasse 10
//
// Das Schuljahr endet im Juli; ab August läuft das neue Schuljahr, daher wird
// das Basisjahr um 1 erhöht, wenn wir uns ab August befinden.
func calculateAbgaengerJahr(klasse string) int {
	klasse = strings.ToLower(strings.TrimSpace(klasse))

	// Führende Ziffern extrahieren
	gradeStr := ""
	suffix := ""
	for i, c := range klasse {
		if c >= '0' && c <= '9' {
			gradeStr += string(c)
		} else {
			suffix = klasse[i:]
			break
		}
	}

	grade, err := strconv.Atoi(gradeStr)
	if err != nil || grade < 1 {
		return time.Now().Year() + 5 // Fallback
	}

	var maxGrade int
	switch {
	case strings.HasPrefix(suffix, "h"):
		maxGrade = 9 // Hauptschule → endet mit Klasse 9h
	case grade >= 11:
		maxGrade = 13 // Oberstufe → endet mit Klasse 13
	default:
		maxGrade = 10 // Gymnasium / Realschule → endet mit Klasse 10
	}

	yearsLeft := maxGrade - grade
	if yearsLeft < 0 {
		yearsLeft = 0
	}

	// Basisjahr: Schuljahresende liegt im Juli.
	// Ab August läuft das neue Schuljahr → aktueller Schüler schließt erst nächsten Sommer ab.
	now := time.Now()
	baseYear := now.Year()
	if now.Month() >= time.August {
		baseYear++
	}
	return baseYear + yearsLeft
}

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

// ListStudentsHandler returns all students, optionally filtered by klasse.
// @Summary      List students
// @Description  Retrieves students, optionally filtered by a specific school class, along with loan counts.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        klasse  query     string  false  "School class to filter by"
// @Success      200     {array}   map[string]any
// @Failure      500     {object}  map[string]string
// @Router       /schueler [get]
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

// CreateStudentRequest defines the payload for creating a new student.
type CreateStudentRequest struct {
	Vorname   string `json:"vorname"`
	Nachname  string `json:"nachname"`
	Klasse    string `json:"klasse"`
	BarcodeID string `json:"barcode_id"`
}

// CreateStudentHandler inserts a new student record into the database.
// @Summary      Create student
// @Description  Creates a new student profile in the library database.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        student  body      CreateStudentRequest  true  "Student creation payload"
// @Success      200      {object}  map[string]any
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /schueler [post]
func (s *Server) CreateStudentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateStudentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültiges JSON"))
			return
		}

		req.Vorname = strings.TrimSpace(req.Vorname)
		req.Nachname = strings.TrimSpace(req.Nachname)
		req.Klasse = strings.TrimSpace(req.Klasse)
		req.BarcodeID = strings.TrimSpace(req.BarcodeID)

		if req.Vorname == "" || req.Nachname == "" || req.Klasse == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("Vorname, Nachname und Klasse sind Pflichtfelder"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback(ctx)

		// 1. Resolve/generate barcode_id if not provided
		barcodeID := req.BarcodeID
		if barcodeID == "" {
			var lastBarcode string
			qLast := `
				SELECT barcode_id 
				FROM schueler 
				WHERE barcode_id LIKE 'S-%' 
				ORDER BY barcode_id DESC 
				LIMIT 1
				FOR UPDATE
			`
			err = tx.QueryRow(ctx, qLast).Scan(&lastBarcode)
			startNum := 10001
			if err == nil {
				re := regexp.MustCompile(`S-(\d+)`)
				matches := re.FindStringSubmatch(lastBarcode)
				if len(matches) > 1 {
					if parsed, err := strconv.Atoi(matches[1]); err == nil {
						startNum = parsed + 1
					}
				}
			}
			barcodeID = fmt.Sprintf("S-%05d", startNum)
		} else {
			// Check if barcode_id already exists
			var exists bool
			err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schueler WHERE barcode_id = $1)", barcodeID).Scan(&exists)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			if exists {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("Barcode-ID '%s' wird bereits verwendet", barcodeID))
				return
			}
		}

		// 2. Insert student
		abgaengerJahr := calculateAbgaengerJahr(req.Klasse)
		var studentID string
		qInsert := `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`
		err = tx.QueryRow(ctx, qInsert, barcodeID, req.Vorname, req.Nachname, req.Klasse, abgaengerJahr).Scan(&studentID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "success",
			"id":         studentID,
			"barcode_id": barcodeID,
		})
	}
}

// DeleteStudentHandler deletes a student after checking for outstanding loans and unpaid damage cases, logging it to the audit trail.
// @Summary      Delete student
// @Description  Transactionally deletes a student from the system, checks for active loans or unpaid damage fees, anonymizes historical loans, and writes to audit_log.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Student ID (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /schueler/{id} [delete]
func (s *Server) DeleteStudentHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}

		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende Schüler-ID"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// 1. Check if student exists
		var studentExists bool
		err := s.DB.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schueler WHERE id = $1)", id).Scan(&studentExists)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if !studentExists {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("Schüler nicht gefunden"))
			return
		}

		// 2. Check for active (unreturned) loans
		var activeLoansCount int
		qLoans := `
			SELECT COUNT(*) 
			FROM ausleihen 
			WHERE schueler_id = $1 AND rueckgabe_am IS NULL
		`
		err = s.DB.Pool.QueryRow(ctx, qLoans, id).Scan(&activeLoansCount)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if activeLoansCount > 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("Löschen nicht möglich: Schüler hat noch entliehene Bücher"))
			return
		}

		// 3. Check for unpaid damage cases (unpaid damages block deletion)
		var unpaidDamagesCount int
		qDamages := `
			SELECT COUNT(*) 
			FROM schadensfaelle 
			WHERE schueler_id = $1 AND ist_bezahlt = false
		`
		err = s.DB.Pool.QueryRow(ctx, qDamages, id).Scan(&unpaidDamagesCount)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if unpaidDamagesCount > 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("Löschen nicht möglich: Schüler hat noch unbezahlte Schadensfälle/Gebühren"))
			return
		}

		// 4. Perform transaction delete with audit log
		err = auditRepo.DeleteStudent(ctx, id, claims.UserID, "Manuelle Löschung")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
		})
	}
}

// PatchStudentHandler aktualisiert editierbare Felder eines Schülers (klasse, abgaenger_jahr).
// Wird für den manuellen Override des Abgangsjahrs und für Klassenänderungen verwendet.
func (s *Server) PatchStudentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende Schüler-ID"))
			return
		}

		var req struct {
			Klasse        *string `json:"klasse"`
			AbgaengerJahr *int    `json:"abgaenger_jahr"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("ungültiger Request-Body: %w", err))
			return
		}
		if req.Klasse == nil && req.AbgaengerJahr == nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("mindestens ein Feld (klasse oder abgaenger_jahr) muss angegeben werden"))
			return
		}
		if req.AbgaengerJahr != nil && (*req.AbgaengerJahr < 2000 || *req.AbgaengerJahr > 2100) {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("abgaenger_jahr muss zwischen 2000 und 2100 liegen"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Resolve new abgaenger_jahr: explicit override takes precedence, else recalculate from class
		var newJahr int
		if req.AbgaengerJahr != nil {
			newJahr = *req.AbgaengerJahr
		} else {
			newJahr = calculateAbgaengerJahr(*req.Klasse)
		}

		if req.Klasse != nil {
			// Update both klasse and abgaenger_jahr
			tag, err := s.DB.Pool.Exec(ctx,
				`UPDATE schueler SET klasse = $1, abgaenger_jahr = $2, aktualisiert_am = CURRENT_TIMESTAMP WHERE id = $3`,
				*req.Klasse, newJahr, id)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			if tag.RowsAffected() == 0 {
				apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("Schüler nicht gefunden"))
				return
			}
		} else {
			// Only update abgaenger_jahr
			tag, err := s.DB.Pool.Exec(ctx,
				`UPDATE schueler SET abgaenger_jahr = $1, aktualisiert_am = CURRENT_TIMESTAMP WHERE id = $2`,
				newJahr, id)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			if tag.RowsAffected() == 0 {
				apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("Schüler nicht gefunden"))
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":         "success",
			"abgaenger_jahr": newJahr,
		})
	}
}

// ImportStudentsLUSDHandler handles LUSD-compliant CSV uploads for admins.
func (s *Server) ImportStudentsLUSDHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Parse Multipart Form
		if err := r.ParseMultipartForm(5 << 20); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Hessen LUSD standard CSV uses semicolon (;)
		reader := csv.NewReader(strings.NewReader(string(content)))
		reader.Comma = ';'
		reader.LazyQuotes = true

		headers, err := reader.Read()
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("CSV-Header konnte nicht gelesen werden: %w", err))
			return
		}

		headerMap := make(map[string]int)
		for idx, h := range headers {
			headerMap[strings.ToLower(strings.TrimSpace(h))] = idx
		}

		// Resolve column indexes
		getColIdx := func(keys []string) int {
			for _, k := range keys {
				if idx, ok := headerMap[k]; ok {
					return idx
				}
			}
			return -1
		}

		lusdIDIdx := getColIdx([]string{"lusd_id", "schueler_id", "id", "lusd-id", "schüler-id", "schüler_id", "schuelerid", "schülerid", "lusd id", "schüler id", "schueler id"})
		vornameIdx := getColIdx([]string{"vorname", "first_name", "firstname", "rufname"})
		nachnameIdx := getColIdx([]string{"nachname", "last_name", "lastname", "name", "familienname"})
		klasseIdx := getColIdx([]string{"klasse", "class", "jahrgang", "klassenbezeichnung"})
		barcodeIdx := getColIdx([]string{"barcode_id", "barcode", "barcode-id"})

		// Validation
		if vornameIdx == -1 || nachnameIdx == -1 || klasseIdx == -1 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("CSV muss mindestens die Spalten 'Vorname', 'Nachname' und 'Klasse' enthalten"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
		defer cancel()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback(ctx)

		// Get next barcode sequence S-XXXXX helper
		var lastBarcode string
		qLast := `
			SELECT barcode_id 
			FROM schueler 
			WHERE barcode_id LIKE 'S-%' 
			ORDER BY barcode_id DESC 
			LIMIT 1
			FOR UPDATE
		`
		err = tx.QueryRow(ctx, qLast).Scan(&lastBarcode)
		startNum := 10001
		if err == nil {
			re := regexp.MustCompile(`S-(\d+)`)
			matches := re.FindStringSubmatch(lastBarcode)
			if len(matches) > 1 {
				if parsed, err := strconv.Atoi(matches[1]); err == nil {
					startNum = parsed + 1
				}
			}
		}

		importedCount := 0
		lineNum := 1

		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			lineNum++
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("Fehler in Zeile %d: %w", lineNum, err))
				return
			}

			if len(row) <= vornameIdx || len(row) <= nachnameIdx || len(row) <= klasseIdx {
				continue
			}

			vorname := strings.TrimSpace(row[vornameIdx])
			nachname := strings.TrimSpace(row[nachnameIdx])
			klasse := strings.TrimSpace(row[klasseIdx])

			if vorname == "" || nachname == "" || klasse == "" {
				continue // Skip invalid rows
			}

			var lusdID *string
			if lusdIDIdx != -1 && len(row) > lusdIDIdx {
				val := strings.TrimSpace(row[lusdIDIdx])
				if val != "" {
					lusdID = &val
				}
			}

			var barcodeID string
			if barcodeIdx != -1 && len(row) > barcodeIdx {
				barcodeID = strings.TrimSpace(row[barcodeIdx])
			}

			// Try to find student
			var existingID string
			found := false

			// 1. Try by lusdID
			if lusdID != nil {
				err = tx.QueryRow(ctx, "SELECT id FROM schueler WHERE lusd_id = $1 LIMIT 1", *lusdID).Scan(&existingID)
				if err == nil {
					found = true
				}
			}

			// 2. Try by barcodeID
			if !found && barcodeID != "" {
				err = tx.QueryRow(ctx, "SELECT id FROM schueler WHERE barcode_id = $1 LIMIT 1", barcodeID).Scan(&existingID)
				if err == nil {
					found = true
				}
			}

			// 3. Try by Name combination
			if !found {
				err = tx.QueryRow(ctx, "SELECT id FROM schueler WHERE lower(vorname) = lower($1) AND lower(nachname) = lower($2) LIMIT 1", vorname, nachname).Scan(&existingID)
				if err == nil {
					found = true
				}
			}

			if found {
				// Update student's class (Versetzung)
				qUpdate := `
					UPDATE schueler 
					SET klasse = $1, aktualisiert_am = CURRENT_TIMESTAMP
				`
				params := []any{klasse}
				paramCount := 2

				if lusdID != nil {
					qUpdate += fmt.Sprintf(", lusd_id = $%d", paramCount)
					params = append(params, *lusdID)
					paramCount++
				}

				qUpdate += fmt.Sprintf(" WHERE id = $%d", paramCount)
				params = append(params, existingID)

				_, err = tx.Exec(ctx, qUpdate, params...)
				if err != nil {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
					return
				}
			} else {
				// Generate new barcode if empty
				if barcodeID == "" {
					barcodeID = fmt.Sprintf("S-%05d", startNum)
					startNum++
				}

				defaultAbgaengerJahr := time.Now().Year() + 5
				qInsert := `
					INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id)
					VALUES ($1, $2, $3, $4, $5, $6)
				`
				_, err = tx.Exec(ctx, qInsert, barcodeID, vorname, nachname, klasse, defaultAbgaengerJahr, lusdID)
				if err != nil {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
					return
				}
			}
			importedCount++
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   "success",
			"imported": importedCount,
		})
	}
}
