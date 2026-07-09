package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
)

// PromoteStudentsResponse liefert die Statistik des Schuljahreswechsels zurück.
// Bei DryRun=true wurde nichts geschrieben — die Zahlen sind die exakte Vorschau
// (identisches SQL, Transaktion wird zurückgerollt).
type PromoteStudentsResponse struct {
	PromotedCount int  `json:"promoted_count"`
	ArchivedCount int  `json:"archived_count"`
	DryRun        bool `json:"dry_run"`
}

// promoteStudentsRequest verlangt eine explizite Bestätigung im Body. Das ist eine
// zusätzliche serverseitige Sicherung gegen versehentliche oder automatisierte
// Aufrufe (z. B. ein wiederholter Retry) — bei einem irreversiblen Batch-Vorgang
// dieser Tragweite reicht eine reine Client-seitige Bestätigung nicht aus.
// DryRun führt dieselbe Berechnung in einer Transaktion aus und rollt sie zurück:
// exakte Vorschau ohne Seiteneffekte, ohne Confirm nutzbar.
type promoteStudentsRequest struct {
	Confirm bool `json:"confirm"`
	DryRun  bool `json:"dry_run"`
}

// promoteStudentsQuery zählt Klassenbezeichnungen um eine Stufe hoch und markiert
// Abschlussklassen als Abgänger.
//
// Wichtige Invarianten:
//   - klasse ist NOT NULL (schema.sql) — Abgänger bekommen 'ABG', exakt wie der
//     LUSD-Import-Pfad (computeLusdChanges), damit beide Wege dieselbe Konvention
//     schreiben.
//   - lpad erhält führende Nullen ('05a' → '06a'), ohne beim Stellenwechsel zu
//     kürzen ('09' → '10', greatest() verhindert lpad-Truncation).
const promoteStudentsQuery = `
	WITH parsed AS (
		SELECT id,
			   klasse,
			   substring(klasse from '^\d+') AS old_digits,
			   (substring(klasse from '^\d+')::int + 1) AS new_grade,
			   substring(klasse from '^\d+(.*)$') AS new_suffix
		FROM schueler
		WHERE ist_abgaenger = false
		  AND deleted_at IS NULL
		  AND klasse ~ '^\d+'
	),
	calculated AS (
		SELECT id,
			   (lpad(new_grade::text, greatest(length(old_digits), length(new_grade::text)), '0') || new_suffix) AS new_klasse,
			   CASE
				 WHEN new_grade = 10 AND new_suffix ILIKE '%h%' THEN true
				 WHEN new_grade = 11 AND new_suffix ILIKE '%r%' THEN true
				 WHEN new_grade >= 14 THEN true
				 ELSE false
			   END AS is_graduating
		FROM parsed
	),
	updated AS (
		UPDATE schueler s
		SET
			klasse = CASE WHEN c.is_graduating THEN 'ABG' ELSE c.new_klasse END,
			ist_abgaenger = c.is_graduating,
			ist_gesperrt = CASE WHEN c.is_graduating THEN true ELSE s.ist_gesperrt END,
			abgaenger_jahr = CASE
				WHEN c.is_graduating THEN EXTRACT(YEAR FROM CURRENT_DATE)
				ELSE s.abgaenger_jahr
			END,
			aktualisiert_am = CURRENT_TIMESTAMP
		FROM calculated c
		WHERE s.id = c.id
		RETURNING c.is_graduating
	)
	SELECT
		COUNT(*) FILTER (WHERE is_graduating = false) AS versetzt,
		COUNT(*) FILTER (WHERE is_graduating = true) AS abgaenger
	FROM updated;
`

// PromoteStudentsHandler führt den automatischen Schuljahreswechsel durch.
// @Summary      Automatische Versetzung (Schuljahreswechsel)
// @Description  Erhöht die Klassenstufe aller aktiven Schüler um 1. Markiert Abschlussklassen (9H, 10R, 13) automatisch als Abgänger. Erfordert { "confirm": true } im Body; { "dry_run": true } liefert eine exakte Vorschau ohne Änderungen.
// @Tags         schueler
// @Accept       json
// @Produce      json
// @Param        body body promoteStudentsRequest true "Bestätigung"
// @Success      200  {object}  PromoteStudentsResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /students/promote [post]
func (s *Server) PromoteStudentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		claims, ok := auth.GetClaims(ctx)
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}

		var req promoteStudentsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültiger oder fehlender Request-Body"))
			return
		}
		if !req.DryRun && !req.Confirm {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New(`Bestätigung erforderlich: { "confirm": true } im Body senden`))
			return
		}

		// Der gesamte Hochzähl-Vorgang läuft in einer einzigen strikten Transaktion:
		// entweder wird JEDER Schüler versetzt/archiviert, oder — bei jedem Fehler —
		// keiner. db.SafeRollback greift auf jedem Fehler- UND Panic-Pfad; nur der
		// explizite Commit ganz unten übernimmt final. Im Dry-Run-Modus wird bewusst
		// NICHT committet — dieselbe Berechnung, null Seiteneffekte.
		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer db.SafeRollback(ctx, tx)

		if !req.DryRun {
			// Doppellauf-Schutz: Ein zweiter Lauf kurz nach dem ersten würde alle
			// Schüler NOCHMAL versetzen (+2). Ein absichtlicher zweiter Lauf im
			// nächsten Jahr bleibt möglich.
			var recentRuns int
			err := tx.QueryRow(ctx, `
				SELECT COUNT(*) FROM audit_logs
				WHERE aktion = 'SCHULJAHRESWECHSEL'
				  AND zeitstempel > NOW() - INTERVAL '10 minutes'
			`).Scan(&recentRuns)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			if recentRuns > 0 {
				apierrors.SendHTTPError(w, http.StatusConflict,
					errors.New("Schuljahreswechsel wurde vor wenigen Minuten bereits durchgeführt — ein erneuter Lauf würde alle Schüler doppelt versetzen"))
				return
			}
		}

		var promoted, archived int
		if err := tx.QueryRow(ctx, promoteStudentsQuery).Scan(&promoted, &archived); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if !req.DryRun {
			// Audit-Eintrag atomar mit dem Batch — ein irreversibler Massenvorgang
			// gehört ins Audit-Log, nicht nur ins Server-Log. Er trägt zugleich den
			// Doppellauf-Schutz (siehe oben).
			if _, err := tx.Exec(ctx, `
				INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse, zeitstempel)
				VALUES ($1, 'SCHULJAHRESWECHSEL', $2::jsonb, $3, CURRENT_TIMESTAMP)
			`, claims.UserID,
				fmt.Sprintf(`{"versetzt": %d, "abgaenger": %d}`, promoted, archived),
				getIP(r)); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}

			if err := tx.Commit(ctx); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			log.Printf("Schuljahreswechsel durchgeführt (Benutzer %s): %d Schüler versetzt, %d neue Abgänger", claims.UserID, promoted, archived)
		}

		RespondJSON(w, http.StatusOK, PromoteStudentsResponse{
			PromotedCount: promoted,
			ArchivedCount: archived,
			DryRun:        req.DryRun,
		})
	}
}
