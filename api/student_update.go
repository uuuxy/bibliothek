package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
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
		_, _ = s.DB.Pool.Exec(ctx, "INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse) VALUES ($1, $2, $3::jsonb, $4)", claims.UserID, "DELETE_STUDENT", details, r.RemoteAddr)

		RespondJSON(w, http.StatusOK, map[string]any{
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
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("mindestens ein Feld muss angegeben werden"))
			return
		}
		if req.AbgaengerJahr != nil && (*req.AbgaengerJahr < 2000 || *req.AbgaengerJahr > 2100) {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("abgaenger_jahr muss zwischen 2000 und 2100 liegen"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		query := "UPDATE schueler SET aktualisiert_am = CURRENT_TIMESTAMP"
		args := []interface{}{}
		argId := 1

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
		_ = json.NewEncoder(w).Encode(response)
	}
}
