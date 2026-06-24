package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

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

		ctx := r.Context()

		// 1. Check if student exists
		var studentExists bool
		err := s.DB.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schueler WHERE id = $1)", id).Scan(&studentExists)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if !studentExists {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("schüler nicht gefunden"))
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
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("löschen nicht möglich: Schüler hat noch entliehene Bücher"))
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
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("löschen nicht möglich: Schüler hat noch unbezahlte Schadensfälle/Gebühren"))
			return
		}

		// 4. Perform transaction delete with audit log
		err = auditRepo.DeleteStudent(ctx, id, claims.UserID, "Manuelle Löschung")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Admin audit log
		details := fmt.Sprintf(`{"student_id":"%s"}`, id)
		logExec(s.DB.Pool.Exec(ctx, "INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse) VALUES ($1, $2, $3::jsonb, $4)", claims.UserID, "DELETE_STUDENT", details, getIP(r)))

		RespondJSON(w, http.StatusOK, map[string]any{
			"status": "success",
		})
	}
}

// PatchStudentHandler aktualisiert editierbare Felder eines Schülers (klasse, abgaenger_jahr).
// Wird nun auch für das Bearbeiten aller Stammdaten in der UI genutzt.
func (s *Server) PatchStudentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende Schüler-ID"))
			return
		}

		var req struct {
			Vorname           *string `json:"vorname"`
			Nachname          *string `json:"nachname"`
			Klasse            *string `json:"klasse"`
			LusdID            *string `json:"lusd_id"`
			BarcodeID         *string `json:"barcode_id"`
			AbgaengerJahr     *int    `json:"abgaenger_jahr"`
			Geburtsdatum      *string `json:"geburtsdatum"`
			IsManuallyBlocked *bool   `json:"is_manually_blocked"`
			BlockReason       *string `json:"block_reason"`
			Strasse           *string `json:"strasse"`
			Hausnummer        *string `json:"hausnummer"`
			Plz               *string `json:"plz"`
			Ort               *string `json:"ort"`
			ElternEmail       *string `json:"eltern_email"`
		}
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		ctx := r.Context()

		query := "UPDATE schueler SET aktualisiert_am = CURRENT_TIMESTAMP"
		args := []interface{}{}
		argId := 1

		if req.Vorname != nil {
			query += fmt.Sprintf(", vorname = $%d", argId)
			args = append(args, *req.Vorname)
			argId++
		}
		if req.Nachname != nil {
			query += fmt.Sprintf(", nachname = $%d", argId)
			args = append(args, *req.Nachname)
			argId++
		}
		if req.LusdID != nil {
			query += fmt.Sprintf(", lusd_id = $%d", argId)
			args = append(args, *req.LusdID)
			argId++
		}
		if req.BarcodeID != nil {
			query += fmt.Sprintf(", barcode_id = $%d", argId)
			args = append(args, *req.BarcodeID)
			argId++
		}
		if req.Klasse != nil {
			query += fmt.Sprintf(", klasse = $%d", argId)
			args = append(args, *req.Klasse)
			argId++

			// Resolve new abgaenger_jahr if not explicitly provided
			if req.AbgaengerJahr == nil {
				newJahr := calculateAbgaengerJahr(*req.Klasse)
				req.AbgaengerJahr = &newJahr
			}
		}

		if req.AbgaengerJahr != nil {
			query += fmt.Sprintf(", abgaenger_jahr = $%d", argId)
			args = append(args, *req.AbgaengerJahr)
			argId++
		}

		if req.Geburtsdatum != nil {
			var parsedDate *time.Time
			if *req.Geburtsdatum != "" {
				t, err := time.Parse("2006-01-02", *req.Geburtsdatum)
				if err != nil {
					apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("ungültiges Datumsformat für Geburtsdatum: %q — erwartet YYYY-MM-DD", *req.Geburtsdatum))
					return
				}
				parsedDate = &t
			}
			query += fmt.Sprintf(", geburtsdatum = $%d", argId)
			args = append(args, parsedDate)
			argId++
		}

		if req.IsManuallyBlocked != nil {
			query += fmt.Sprintf(", is_manually_blocked = $%d", argId)
			args = append(args, *req.IsManuallyBlocked)
			argId++
		}

		if req.BlockReason != nil {
			query += fmt.Sprintf(", block_reason = $%d", argId)
			args = append(args, *req.BlockReason)
			argId++
		}

		// Postanschrift & Elternkontakt (Stammdaten): nur aktualisieren, wenn das Feld
		// im Payload vorhanden ist (nil = nicht mitgeschickt → unverändert lassen).
		if req.Strasse != nil {
			query += fmt.Sprintf(", strasse = $%d", argId)
			args = append(args, *req.Strasse)
			argId++
		}
		if req.Hausnummer != nil {
			query += fmt.Sprintf(", hausnummer = $%d", argId)
			args = append(args, *req.Hausnummer)
			argId++
		}
		if req.Plz != nil {
			query += fmt.Sprintf(", plz = $%d", argId)
			args = append(args, *req.Plz)
			argId++
		}
		if req.Ort != nil {
			query += fmt.Sprintf(", ort = $%d", argId)
			args = append(args, *req.Ort)
			argId++
		}
		if req.ElternEmail != nil {
			query += fmt.Sprintf(", eltern_email = $%d", argId)
			args = append(args, *req.ElternEmail)
			argId++
		}

		// Empty PATCH (no updatable field provided): reject as 400 instead of running a
		// no-op UPDATE whose RowsAffected==0 would be misreported as 404 for an existing student.
		if argId == 1 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("keine zu aktualisierenden Felder angegeben"))
			return
		}

		query += fmt.Sprintf(" WHERE id = $%d", argId)
		args = append(args, id)

		tag, err := s.DB.Pool.Exec(ctx, query, args...)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if tag.RowsAffected() == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("schüler nicht gefunden"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{"status": "success"}
		if req.AbgaengerJahr != nil {
			response["abgaenger_jahr"] = *req.AbgaengerJahr
		}
		httpresp.Encode(w, response)
	}
}
