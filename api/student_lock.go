package api

import (
	"net/http"
	"strings"

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
			IsLocked bool   `json:"is_locked"`
			Reason   string `json:"reason"`
		}
		if !DecodeAndValidate(w, r, &req) {
			// DecodeAndValidate handles writing its own response for now
			return nil
		}

		// Eine manuelle Sperre OHNE Grund ist genau die "Zombie-Sperre", die der
		// DB-Constraint chk_schueler_block_reason verhindern soll: Das Personal sähe nur das
		// rote Flag ohne Kontext. Daher ist der Grund beim Sperren Pflicht (beim Entsperren
		// irrelevant).
		reason := strings.TrimSpace(req.Reason)
		if req.IsLocked && reason == "" {
			return apierrors.BadRequest("Für eine manuelle Sperre ist ein Grund erforderlich.", nil)
		}

		ctx := r.Context()

		// block_reason konsistent zum Sperrzustand pflegen: beim Sperren den Grund setzen;
		// beim Entsperren nur räumen, wenn KEINE Systemsperre (ist_gesperrt) mehr besteht —
		// sonst bliebe deren Grund erhalten (chk_schueler_block_reason verlangt ihn dann).
		// `leser`, nicht die Sicht `schueler`: Die Handsperre gilt jedem Leser. Seit der
		// Leserdatei steht der Knopf auch in der Akte eines Kollegen, und über die Sicht
		// antwortete er mit „schüler nicht gefunden". Sie wirkt dort, wo sie muss — der
		// Ausleihpfad liest den Leser (GetLeserByID), nicht die Sicht.
		query := `
			UPDATE leser
			SET is_manually_blocked = $1,
			    block_reason = CASE
			        WHEN $1 = true      THEN $3
			        WHEN ist_gesperrt   THEN block_reason
			        ELSE NULL
			    END,
			    aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $2
			-- coalesce auf die Klasse: Sie ist seit Migration 123 nullbar (ein Kollege hat
			-- keine), und ein NULL in *string ist ein 500 („cannot scan NULL into *string").
			RETURNING id, vorname, nachname, coalesce(klasse, ''), is_manually_blocked
		`

		var student struct {
			ID                string `json:"id"`
			Vorname           string `json:"vorname"`
			Nachname          string `json:"nachname"`
			Klasse            string `json:"klasse"`
			IsManuallyBlocked bool   `json:"is_manually_blocked"`
		}

		err := s.DB.Pool.QueryRow(ctx, query, req.IsLocked, id, reason).Scan(
			&student.ID, &student.Vorname, &student.Nachname,
			&student.Klasse, &student.IsManuallyBlocked,
		)
		if err != nil {
			if err.Error() == "no rows in result set" {
				return apierrors.NotFound("leser nicht gefunden", err)
			}
			return apierrors.Internal("Fehler beim Aktualisieren der Sperre", err)
		}

		w.Header().Set("Content-Type", "application/json")
		httpresp.Encode(w, student)
		return nil
	})
}
