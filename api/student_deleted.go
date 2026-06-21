package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
)

// GetDeletedStudentsHandler liefert eine Liste aller weichgelöschten Schüler für den Papierkorb.
func (s *Server) GetDeletedStudentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		rows, err := s.DB.Pool.Query(ctx, `
			SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, deleted_at
			FROM schueler 
			WHERE deleted_at IS NOT NULL
			ORDER BY deleted_at DESC
		`)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		students := []map[string]any{}
		for rows.Next() {
			var id, barcode, vorname, nachname, kl string
			var abgaengerJahr *int
			var gesperrt bool
			var deletedAt string

			if err := rows.Scan(&id, &barcode, &vorname, &nachname, &kl, &abgaengerJahr, &gesperrt, &deletedAt); err == nil {
				students = append(students, map[string]any{
					"id":             id,
					"barcode_id":     barcode,
					"vorname":        vorname,
					"nachname":       nachname,
					"klasse":         kl,
					"abgaenger_jahr": abgaengerJahr,
					"ist_gesperrt":   gesperrt,
					"deleted_at":     deletedAt,
				})
			}
		}

		RespondJSON(w, http.StatusOK, students)
	}
}

// RestoreStudentHandler stellt einen weichgelöschten Schüler wieder her.
func (s *Server) RestoreStudentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende Schüler-ID"))
			return
		}

		ctx := r.Context()

		// Führe das Restore durch (deleted_at = NULL)
		tag, err := s.DB.Pool.Exec(ctx, "UPDATE schueler SET deleted_at = NULL, aktualisiert_am = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NOT NULL", id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if tag.RowsAffected() == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("schüler nicht im Papierkorb gefunden"))
			return
		}

		// Optional: Audit-Log für Restore anlegen
		if claims, ok := auth.GetClaims(ctx); ok {
			_, _ = s.DB.Pool.Exec(ctx, "INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse) VALUES ($1, $2, $3::jsonb, $4)", claims.UserID, "RESTORE_STUDENT", `{"student_id":"`+id+`"}`, r.RemoteAddr)
		}

		RespondJSON(w, http.StatusOK, map[string]any{
			"status":  "success",
			"message": "Schüler erfolgreich wiederhergestellt",
		})
	}
}
