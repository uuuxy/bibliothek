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
	"bibliothek/pkg/leserart"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// meldungSperreFehler geht hinaus, wenn das Umschalten an der Datenbank scheitert.
const meldungSperreFehler = "Fehler beim Aktualisieren der Sperre"

// LockStudentHandler sperrt einen Leser von Hand oder hebt jede Sperre an ihm auf — die
// eine Tür zu diesem Zustand (schueler_sperre_eine_tuer_pg_test.go).
//
// Aufheben nimmt beide Sperren weg: die von Hand und die, die das Programm den Ehemaligen
// setzt (ist_gesperrt: Versetzung, LUSD-Import). Beide lassen an der Theke nur die Rückgabe
// zu (service.pruefeAusleihSperren); aufgehoben werden sie hier, in der Akte oder aus dem
// Dialog der Theke — wie in Littera in den Leserdaten. Sonst fiele die Sperre der
// Ehemaligen erst, wenn der nächste LUSD-Import den Schüler wiederfände.
//
// Nicht umschalten lässt sich: ein Leser im Papierkorb (zurück nur über Wiederherstellen),
// ein anonymisierter Datensatz (keine Person mehr) und die Sperre eines Kollegen — ein
// Kollege wird nie gesperrt.
//
// Jede Änderung steht im Protokoll (LESER_GESPERRT, LESER_ENTSPERRT), wie das Übergehen an
// der Theke (OVERRIDE_BLOCK).
func (s *Server) LockStudentHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		return s.handleLockStudent(w, r, auditRepo)
	})
}

// handleLockStudent schaltet die Sperre in einer Transaktion um und schreibt den Eintrag ins
// Protokoll.
func (s *Server) handleLockStudent(w http.ResponseWriter, r *http.Request, auditRepo repository.AuditRepository) error {
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
		return apierrors.Internal(meldungSperreFehler, err)
	}
	defer db.SafeRollback(ctx, tx)

	// `leser`, nicht die Sicht `schueler`: Die Akte eines Kollegen ruft die Tür auch, und
	// über die Sicht antwortete sie mit „schüler nicht gefunden".
	alt, err := repository.SperreLeserZeileMitStand(ctx, tx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return apierrors.NotFound("leser nicht gefunden", err)
	}
	if err != nil {
		return apierrors.Internal(meldungSperreFehler, err)
	}
	switch {
	case alt.Geloescht:
		return apierrors.Conflict("Der Leser liegt im Papierkorb. Erst wiederherstellen.", nil)
	case alt.Anonymisiert:
		return apierrors.Conflict("Ein anonymisierter Datensatz bleibt gesperrt.", nil)
	case req.IsLocked && !leserart.IstSchueler(alt.Art):
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
	neu, err := repository.SetzeSperreVonHand(ctx, tx, id, req.IsLocked, reason)
	if err != nil {
		return apierrors.Internal(meldungSperreFehler, err)
	}
	student.ID, student.Vorname, student.Nachname, student.Klasse = neu.ID, neu.Vorname, neu.Nachname, neu.Klasse
	student.IsManuallyBlocked, student.IstGesperrt = neu.VonHand, neu.Gesperrt
	if err := tx.Commit(ctx); err != nil {
		return apierrors.Internal(meldungSperreFehler, err)
	}

	protokolliereSperre(r, auditRepo, claims.UserID, id, req.IsLocked, reason, alt)

	w.Header().Set("Content-Type", "application/json")
	httpresp.Encode(w, student)
	return nil
}

// protokolliereSperre schreibt das Sperren und das Aufheben ins Protokoll. Best effort wie
// FRIST_OVERRIDE: Die Änderung gilt, auch wenn das Log klemmt; der Fehlversuch steht im
// Server-Log. Aufheben ohne bestehende Sperre ändert nichts und schreibt nichts.
func protokolliereSperre(r *http.Request, auditRepo repository.AuditRepository, adminID, id string, sperren bool, grund string, alt repository.LeserSperrStand) {
	aktion, details := "LESER_GESPERRT", map[string]any{"schueler_id": id, "grund": grund}
	if !sperren {
		if !alt.VonHand && !alt.VomProgramm {
			return
		}
		aktion, details = "LESER_ENTSPERRT", map[string]any{
			"schueler_id":  id,
			"grund":        alt.Grund,
			"von_hand":     alt.VonHand,
			"vom_programm": alt.VomProgramm,
		}
	}
	if err := auditRepo.LogAdminAktion(r.Context(), adminID, aktion, getIP(r), details); err != nil {
		log.Printf("Audit für %s fehlgeschlagen: %v", aktion, err)
	}
}
