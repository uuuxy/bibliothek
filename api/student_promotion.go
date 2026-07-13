package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"

	"github.com/jackc/pgx/v5"
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

		req, ok := parsePromoteRequest(w, r)
		if !ok {
			return
		}

		promoted, archived, ok := s.fuehreSchuljahreswechselAus(ctx, w, r, req, claims.UserID)
		if !ok {
			return
		}

		RespondJSON(w, http.StatusOK, PromoteStudentsResponse{
			PromotedCount: promoted,
			ArchivedCount: archived,
			DryRun:        req.DryRun,
		})
	}
}

// fuehreSchuljahreswechselAus wickelt den Versetzungs-Batch in einer einzigen strikten
// Transaktion ab: entweder wird JEDER Schüler versetzt/archiviert, oder — bei jedem
// Fehler — keiner (db.SafeRollback greift auf jedem Fehler- UND Panic-Pfad). Im
// Dry-Run-Modus wird bewusst NICHT committet (dieselbe Berechnung, null Seiteneffekte).
// ok=false: die Fehlerantwort wurde bereits geschrieben.
func (s *Server) fuehreSchuljahreswechselAus(ctx context.Context, w http.ResponseWriter, r *http.Request, req promoteStudentsRequest, userID string) (promoted, archived int, ok bool) {
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return 0, 0, false
	}
	defer db.SafeRollback(ctx, tx)

	if !req.DryRun {
		if !pruefeDoppellaufSchutz(ctx, tx, w) {
			return 0, 0, false
		}
	}

	if err := tx.QueryRow(ctx, promoteStudentsQuery).Scan(&promoted, &archived); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return 0, 0, false
	}

	if !req.DryRun {
		if !s.finalisiereSchuljahreswechsel(ctx, tx, w, r, userID, promoted, archived) {
			return 0, 0, false
		}
	}
	return promoted, archived, true
}

// parsePromoteRequest dekodiert den Request-Body und erzwingt die explizite Bestätigung
// (außer im Dry-Run). ok=false: die Fehlerantwort wurde bereits geschrieben.
func parsePromoteRequest(w http.ResponseWriter, r *http.Request) (promoteStudentsRequest, bool) {
	var req promoteStudentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültiger oder fehlender Request-Body"))
		return req, false
	}
	if !req.DryRun && !req.Confirm {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New(`bestätigung erforderlich: { "confirm": true } im Body senden`))
		return req, false
	}
	return req, true
}

// pruefeDoppellaufSchutz verhindert einen zweiten Schuljahreswechsel innerhalb von
// 10 Minuten (ein zweiter Lauf würde alle Schüler +2 versetzen). ok=false: die
// Fehlerantwort (500 oder 409) wurde bereits geschrieben.
func pruefeDoppellaufSchutz(ctx context.Context, tx pgx.Tx, w http.ResponseWriter) bool {
	var recentRuns int
	err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_logs
		WHERE aktion = 'SCHULJAHRESWECHSEL'
		  AND zeitstempel > NOW() - INTERVAL '10 minutes'
	`).Scan(&recentRuns)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}
	if recentRuns > 0 {
		apierrors.SendHTTPError(w, http.StatusConflict,
			errors.New("schuljahreswechsel wurde vor wenigen Minuten bereits durchgeführt — ein erneuter Lauf würde alle Schüler doppelt versetzen"))
		return false
	}
	return true
}

// finalisiereSchuljahreswechsel schreibt den Audit-Eintrag atomar mit dem Batch (er
// trägt zugleich den Doppellauf-Schutz) und committet die Transaktion. ok=false: die
// Fehlerantwort wurde bereits geschrieben.
func (s *Server) finalisiereSchuljahreswechsel(ctx context.Context, tx pgx.Tx, w http.ResponseWriter, r *http.Request, userID string, promoted, archived int) bool {
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse, zeitstempel)
		VALUES ($1, 'SCHULJAHRESWECHSEL', $2::jsonb, $3, CURRENT_TIMESTAMP)
	`, userID,
		fmt.Sprintf(`{"versetzt": %d, "abgaenger": %d}`, promoted, archived),
		getIP(r)); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}

	if err := tx.Commit(ctx); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}
	log.Printf("Schuljahreswechsel durchgeführt (Benutzer %s): %d Schüler versetzt, %d neue Abgänger", userID, promoted, archived)
	return true
}
