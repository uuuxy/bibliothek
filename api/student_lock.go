package api

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// sperrStand ist der Zustand eines Lesers, bevor die Sperre umgeschaltet wird.
type sperrStand struct {
	art                     string
	geloescht, anonymisiert bool
	vomProgramm, vonHand    bool
	grund                   string
}

// LockStudentHandler sperrt einen Leser von Hand oder hebt jede Sperre an ihm auf — die
// eine Tür zu diesem Zustand (schueler_sperre_eine_tuer_pg_test.go).
//
// Aufheben nimmt seit dem 24.09.2026 BEIDE Sperren weg: die von Hand und die, die das
// Programm den Ehemaligen setzt (ist_gesperrt: Versetzung, LUSD-Import). Beide lassen an der
// Theke nur die Rückgabe zu (service.pruefeAusleihSperren); aufgehoben werden sie hier, in
// der Akte oder aus dem Dialog der Theke — wie in Littera in den Leserdaten. Bis dahin löste
// der Knopf nur die Sperre von Hand, und die der Ehemaligen fiel erst, wenn der nächste
// LUSD-Import den Schüler wiederfand, mit offenem Buch nicht einmal dann.
//
// Nicht umschalten lässt sich: ein Leser im Papierkorb (zurück nur über Wiederherstellen),
// ein anonymisierter Datensatz (keine Person mehr) und die Sperre eines Kollegen — ein
// Kollege wird nie gesperrt (entschieden am 16.09.2026, bestätigt am 24.09.2026).
//
// Jede Änderung steht im Protokoll (LESER_GESPERRT, LESER_ENTSPERRT). Das Übergehen an der
// Theke stand dort seit jeher (OVERRIDE_BLOCK); das Sperren und Entsperren selbst bis zum
// 24.09.2026 nicht, obwohl FACHKONZEPT §10 es behauptete.
func (s *Server) LockStudentHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			return apierrors.Unauthorized("missing session information", nil)
		}
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
		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			return apierrors.Internal("Fehler beim Aktualisieren der Sperre", err)
		}
		defer db.SafeRollback(ctx, tx)

		// `leser`, nicht die Sicht `schueler`: Die Akte eines Kollegen ruft die Tür auch, und
		// über die Sicht antwortete sie mit „schüler nicht gefunden".
		var alt sperrStand
		err = tx.QueryRow(ctx, `
			SELECT art, deleted_at IS NOT NULL, anonymized_at IS NOT NULL,
			       ist_gesperrt, coalesce(is_manually_blocked, false), coalesce(block_reason, '')
			FROM leser WHERE id = $1 FOR UPDATE`, id).Scan(
			&alt.art, &alt.geloescht, &alt.anonymisiert, &alt.vomProgramm, &alt.vonHand, &alt.grund)
		if errors.Is(err, pgx.ErrNoRows) {
			return apierrors.NotFound("leser nicht gefunden", err)
		}
		if err != nil {
			return apierrors.Internal("Fehler beim Aktualisieren der Sperre", err)
		}
		switch {
		case alt.geloescht:
			return apierrors.Conflict("Der Leser liegt im Papierkorb. Erst wiederherstellen.", nil)
		case alt.anonymisiert:
			return apierrors.Conflict("Ein anonymisierter Datensatz bleibt gesperrt.", nil)
		case req.IsLocked && alt.art != "schueler":
			return apierrors.BadRequest("Kollegen werden nicht gesperrt.", nil)
		}

		// Sperren setzt die Sperre von Hand samt Grund. Aufheben nimmt beide Sperren und den
		// Grund weg; chk_schueler_block_reason verlangt ihn nur, solange eine besteht.
		var student struct {
			ID                string `json:"id"`
			Vorname           string `json:"vorname"`
			Nachname          string `json:"nachname"`
			Klasse            string `json:"klasse"`
			IsManuallyBlocked bool   `json:"is_manually_blocked"`
			IstGesperrt       bool   `json:"ist_gesperrt"`
		}
		// coalesce auf die Klasse: Sie ist seit Migration 123 nullbar (ein Kollege hat
		// keine), und ein NULL in *string ist ein 500 („cannot scan NULL into *string").
		err = tx.QueryRow(ctx, `
			UPDATE leser
			SET is_manually_blocked = $1,
			    ist_gesperrt = ist_gesperrt AND $1,
			    block_reason = CASE WHEN $1 THEN $3 ELSE NULL END,
			    aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $2
			RETURNING id, vorname, nachname, coalesce(klasse, ''), is_manually_blocked, ist_gesperrt`,
			req.IsLocked, id, reason).Scan(
			&student.ID, &student.Vorname, &student.Nachname,
			&student.Klasse, &student.IsManuallyBlocked, &student.IstGesperrt)
		if err != nil {
			return apierrors.Internal("Fehler beim Aktualisieren der Sperre", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return apierrors.Internal("Fehler beim Aktualisieren der Sperre", err)
		}

		protokolliereSperre(r, auditRepo, claims.UserID, id, req.IsLocked, reason, alt)

		w.Header().Set("Content-Type", "application/json")
		httpresp.Encode(w, student)
		return nil
	})
}

// protokolliereSperre schreibt das Sperren und das Aufheben ins Protokoll. Best effort wie
// FRIST_OVERRIDE: Die Änderung gilt, auch wenn das Log klemmt; der Fehlversuch steht im
// Server-Log. Aufheben ohne bestehende Sperre ändert nichts und schreibt nichts.
func protokolliereSperre(r *http.Request, auditRepo repository.AuditRepository, adminID, id string, sperren bool, grund string, alt sperrStand) {
	aktion, details := "LESER_GESPERRT", map[string]any{"schueler_id": id, "grund": grund}
	if !sperren {
		if !alt.vonHand && !alt.vomProgramm {
			return
		}
		aktion, details = "LESER_ENTSPERRT", map[string]any{
			"schueler_id":  id,
			"grund":        alt.grund,
			"von_hand":     alt.vonHand,
			"vom_programm": alt.vomProgramm,
		}
	}
	if err := auditRepo.LogAdminAktion(r.Context(), adminID, aktion, getIP(r), details); err != nil {
		log.Printf("Audit für %s fehlgeschlagen: %v", aktion, err)
	}
}
