package api

import (
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
)

// LockStudentHandler provides a dedicated secure endpoint to block or unblock a student's checkouts.
func (s *Server) LockStudentHandler() http.HandlerFunc {
	// Wir nutzen hier das neue apierrors.Wrap, um Errors einfach durchzureichen
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("fehlende Schüler-ID", nil)
		}

		var req struct {
			IsLocked bool `json:"is_locked"`
		}
		if !DecodeAndValidate(w, r, &req) {
			// DecodeAndValidate handles writing its own response for now
			return nil
		}

		ctx := r.Context()

		query := `
			UPDATE schueler 
			SET is_manually_blocked = $1, 
			    aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $2
			RETURNING id, vorname, nachname, klasse, is_manually_blocked
		`

		var student struct {
			ID                string `json:"id"`
			Vorname           string `json:"vorname"`
			Nachname          string `json:"nachname"`
			Klasse            string `json:"klasse"`
			IsManuallyBlocked bool   `json:"is_manually_blocked"`
		}

		err := s.DB.Pool.QueryRow(ctx, query, req.IsLocked, id).Scan(
			&student.ID, &student.Vorname, &student.Nachname,
			&student.Klasse, &student.IsManuallyBlocked,
		)
		if err != nil {
			if err.Error() == "no rows in result set" {
				return apierrors.NotFound("schüler nicht gefunden", err)
			}
			return apierrors.Internal("Fehler beim Aktualisieren der Sperre", err)
		}

		w.Header().Set("Content-Type", "application/json")
		httpresp.Encode(w, student)
		return nil
	})
}
