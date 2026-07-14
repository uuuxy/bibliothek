package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/db"
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

// CreateStudentRequest defines the payload for creating a new student.
type CreateStudentRequest struct {
	Vorname      string  `json:"vorname" validate:"required"`
	Nachname     string  `json:"nachname" validate:"required"`
	Klasse       string  `json:"klasse" validate:"required"`
	BarcodeID    string  `json:"barcode_id"`
	Geburtsdatum *string `json:"geburtsdatum"` // Format: YYYY-MM-DD
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
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		req.Vorname = strings.TrimSpace(req.Vorname)
		req.Nachname = strings.TrimSpace(req.Nachname)
		req.Klasse = strings.TrimSpace(req.Klasse)
		req.BarcodeID = strings.TrimSpace(req.BarcodeID)

		ctx := r.Context()

		parsedGebdatum, ok := parseCreateGeburtsdatum(w, req.Geburtsdatum)
		if !ok {
			return
		}

		studentID, barcodeID, ok := s.legeSchuelerAn(ctx, w, req, parsedGebdatum)
		if !ok {
			return
		}

		RespondJSON(w, http.StatusCreated, map[string]any{
			"status":     "success",
			"id":         studentID,
			"barcode_id": barcodeID,
		})
	}
}

// legeSchuelerAn wickelt die Neuanlage in einer Transaktion ab (Duplikatsprüfung,
// Barcode-Auflösung/-Generierung, Insert, Commit) und liefert die neue Schüler- und
// Barcode-ID. ok=false: die Fehlerantwort wurde bereits geschrieben.
func (s *Server) legeSchuelerAn(ctx context.Context, w http.ResponseWriter, req CreateStudentRequest, parsedGebdatum *time.Time) (studentID, barcodeID string, ok bool) {
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", "", false
	}
	defer db.SafeRollback(ctx, tx)

	// 1. Notfall-Wachhund: Duplikatsprüfung (Vorname, Nachname, Geburtsdatum)
	isDuplicate, err := pruefeSchuelerDuplikat(ctx, tx, req.Vorname, req.Nachname, parsedGebdatum)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", "", false
	}
	if isDuplicate {
		apierrors.SendHTTPError(w, http.StatusConflict, errors.New("achtung: Ein Schüler mit diesem Namen und Geburtsdatum existiert bereits im System"))
		return "", "", false
	}

	// 2. Resolve/generate barcode_id if not provided
	barcodeID, ok = resolveNeueBarcodeID(ctx, tx, w, req.BarcodeID)
	if !ok {
		return "", "", false
	}

	// 3. Insert student
	abgaengerJahr := calculateAbgaengerJahr(req.Klasse)
	qInsert := `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, geburtsdatum, abgaenger_jahr)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	if err := tx.QueryRow(ctx, qInsert, barcodeID, req.Vorname, req.Nachname, req.Klasse, parsedGebdatum, abgaengerJahr).Scan(&studentID); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", "", false
	}

	if err := tx.Commit(ctx); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", "", false
	}
	return studentID, barcodeID, true
}

// parseCreateGeburtsdatum parst das optionale Geburtsdatum (YYYY-MM-DD) aus dem
// Anlage-Request. ok=false bedeutet: die Fehlerantwort wurde bereits geschrieben.
func parseCreateGeburtsdatum(w http.ResponseWriter, raw *string) (*time.Time, bool) {
	if raw == nil || *raw == "" {
		return nil, true
	}
	t, err := time.Parse(dateFormatISO, *raw)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültiges Format für Geburtsdatum, erwartet YYYY-MM-DD"))
		return nil, false
	}
	return &t, true
}

// pruefeSchuelerDuplikat erkennt einen bereits existierenden Schüler anhand von
// Vor-/Nachname und Geburtsdatum (case-insensitive, ohne soft-gelöschte Datensätze).
func pruefeSchuelerDuplikat(ctx context.Context, tx pgx.Tx, vorname, nachname string, gebdatum *time.Time) (bool, error) {
	var isDuplicate bool
	q := `SELECT EXISTS(SELECT 1 FROM schueler WHERE lower(vorname) = lower($1) AND lower(nachname) = lower($2) AND coalesce(geburtsdatum, '1900-01-01'::DATE) = coalesce($3::DATE, '1900-01-01'::DATE) AND deleted_at IS NULL)`
	err := tx.QueryRow(ctx, q, vorname, nachname, gebdatum).Scan(&isDuplicate)
	return isDuplicate, err
}

// resolveNeueBarcodeID liefert die zu verwendende Barcode-ID: entweder die vom Client
// gewünschte (nach Eindeutigkeitsprüfung) oder eine neu generierte S-Nummer aus der
// zentralen Sequenz. ok=false bedeutet: die Fehlerantwort wurde bereits geschrieben.
func resolveNeueBarcodeID(ctx context.Context, tx pgx.Tx, w http.ResponseWriter, requested string) (string, bool) {
	if requested == "" {
		// Use central repository for sequence generation
		seqRepo := repository.NewSequenceRepository(tx)
		startNum, err := seqRepo.GetNextSequence(ctx, "schueler", "barcode_id", "S-")
		if err != nil {
			db.SafeRollback(ctx, tx)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return "", false
		}
		return fmt.Sprintf("S-%05d", startNum), true
	}

	// Check if barcode_id already exists
	var exists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schueler WHERE barcode_id = $1)", requested).Scan(&exists); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", false
	}
	if exists {
		apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("Barcode-ID '%s' wird bereits verwendet", requested))
		return "", false
	}
	return requested, true
}
