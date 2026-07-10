package api

import (
	"errors"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
)

// GetDeletedStudentsHandler liefert eine Liste aller weichgelöschten Schüler für den Papierkorb.
func (s *Server) GetDeletedStudentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		rows, err := s.DB.Pool.Query(ctx, `
			SELECT id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''),
			       coalesce(klasse, ''), abgaenger_jahr, coalesce(ist_gesperrt, false), deleted_at
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
			// timestamptz braucht time.Time — ein String-Scan brach hier
			// die Iteration ab und machte den Papierkorb zum 500er.
			var deletedAt time.Time

			if err := rows.Scan(&id, &barcode, &vorname, &nachname, &kl, &abgaengerJahr, &gesperrt, &deletedAt); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
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
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
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
			logExec(s.DB.Pool.Exec(ctx, "INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse) VALUES ($1, $2, $3::jsonb, $4)", claims.UserID, "RESTORE_STUDENT", `{"student_id":"`+id+`"}`, getIP(r)))
		}

		RespondJSON(w, http.StatusOK, map[string]any{
			"status":  "success",
			"message": "Schüler erfolgreich wiederhergestellt",
		})
	}
}
