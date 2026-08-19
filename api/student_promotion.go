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

// schuljahreswechselLockKey ist der feste Advisory-Lock-Schlüssel, der gleichzeitige
// Versetzungsläufe hart serialisiert (siehe fuehreSchuljahreswechselAus). Ein beliebiger,
// aber im Projekt eindeutiger Wert — er teilt keinen Namensraum mit den tabellenbasierten
// Sequenz-Locks (advisoryLockKey in repository/sequence_repo.go).
const schuljahreswechselLockKey int64 = 748_2026

// PromoteStudentsResponse liefert die Statistik des Schuljahreswechsels zurück.
// Bei DryRun=true wurde nichts geschrieben — die Zahlen sind die exakte Vorschau
// (identisches SQL, Transaktion wird zurückgerollt).
type PromoteStudentsResponse struct {
	PromotedCount int  `json:"promoted_count"`
	ArchivedCount int  `json:"archived_count"`
	DryRun        bool `json:"dry_run"`
	// Klassenlehrer-Zuordnungen, die mit versetzt / als Abschlussklasse entfernt
	// wurden (Befund F3: bis 18.08.2026 blieb die Zuordnung beim Jahreswechsel
	// auf den alten Klassennamen stehen — die Mahnliste der neuen 6a ging an
	// niemanden, ohne Fehlermeldung).
	MappingVersetzt int `json:"mapping_versetzt"`
	MappingEntfernt int `json:"mapping_entfernt"`
	// Zuordnungen, deren Zielname schon belegt war (z. B. '9a' und '09a' laufen
	// beide auf '10a'): Sie bleiben unverändert stehen und werden hier genannt,
	// statt den ganzen Lauf scheitern zu lassen.
	MappingKonflikte []string `json:"mapping_konflikte,omitempty"`
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
			-- Abgänger werden gesperrt → chk_schueler_block_reason verlangt einen Grund.
			-- Ein vorhandener (z. B. manueller) Grund bleibt erhalten.
			block_reason = CASE WHEN c.is_graduating
			                    THEN COALESCE(NULLIF(s.block_reason, ''), 'Automatische Abgänger-Sperre (Schuljahreswechsel)')
			                    ELSE s.block_reason END,
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

		resp, ok := s.fuehreSchuljahreswechselAus(ctx, w, r, req, claims.UserID)
		if !ok {
			return
		}
		resp.DryRun = req.DryRun
		RespondJSON(w, http.StatusOK, resp)
	}
}

// fuehreSchuljahreswechselAus wickelt den Versetzungs-Batch in einer einzigen strikten
// Transaktion ab: entweder wird JEDER Schüler versetzt/archiviert, oder — bei jedem
// Fehler — keiner (db.SafeRollback greift auf jedem Fehler- UND Panic-Pfad). Im
// Dry-Run-Modus wird bewusst NICHT committet (dieselbe Berechnung, null Seiteneffekte).
// ok=false: die Fehlerantwort wurde bereits geschrieben.
func (s *Server) fuehreSchuljahreswechselAus(ctx context.Context, w http.ResponseWriter, r *http.Request, req promoteStudentsRequest, userID string) (resp PromoteStudentsResponse, ok bool) {
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return resp, false
	}
	defer db.SafeRollback(ctx, tx)

	if !req.DryRun {
		// HARTE Serialisierung gegen den Doppellauf. Der COUNT-Check unten allein ist
		// TOCTOU: Zwei gleichzeitige Klicks (oder zwei Admins) lesen beide 0, bevor einer
		// committet — und versetzen die GANZE Schule ein zweites Mal (+2 Stufen,
		// irreversibel). Der Advisory-Lock (transaktionsgebunden, gibt beim Commit/
		// Rollback automatisch frei) zwingt den zweiten Lauf zu warten, bis der erste
		// fertig ist; danach sieht dessen COUNT den Audit-Eintrag und antwortet 409.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, schuljahreswechselLockKey); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return resp, false
		}
		if !pruefeDoppellaufSchutz(ctx, tx, w) {
			return resp, false
		}
	}

	if err := tx.QueryRow(ctx, promoteStudentsQuery).Scan(&resp.PromotedCount, &resp.ArchivedCount); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return resp, false
	}

	// Die Klassenlehrer-Zuordnung wandert MIT der Kohorte (dieselbe Transaktion,
	// dieselbe Rechenregel). Die Klassen-Bücherlisten (class_books) bleiben
	// BEWUSST unangetastet: Sie hängen am Stufen-Namen ("die 8a liest Markl 2"
	// gilt für jede künftige 8a), nicht an den Menschen — zwei Bedeutungen von
	// "Klasse", die der Jahreswechsel nicht verwechseln darf (Befund F3).
	if err := versetzeKlassenlehrerZuordnung(ctx, tx, &resp); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return resp, false
	}

	if !req.DryRun {
		if !s.finalisiereSchuljahreswechsel(ctx, tx, w, r, userID, resp.PromotedCount, resp.ArchivedCount) {
			return resp, false
		}
	}
	return resp, true
}

// versetzeKlassenlehrerZuordnung zählt die Klassennamen der Lehrer-Zuordnung mit
// derselben Regel hoch wie promoteStudentsQuery die Schüler. Absteigend nach
// Stufe, damit '09a' erst nach dem Wegzug von '10a' auf den freien Namen rücken
// kann; Abschlussklassen-Zeilen werden entfernt (die Kohorte verlässt die
// Schule). Kollidiert ein Zielname trotzdem (z. B. '9a' UND '09a' laufen beide
// auf '10a'), bleibt die Zeile stehen und wird im Ergebnis genannt — ein
// Namenskonflikt darf den Schuljahreswechsel nicht scheitern lassen.
func versetzeKlassenlehrerZuordnung(ctx context.Context, tx pgx.Tx, resp *PromoteStudentsResponse) error {
	rows, err := tx.Query(ctx, `
		SELECT klasse,
		       lpad((substring(klasse from '^\d+')::int + 1)::text,
		            greatest(length(substring(klasse from '^\d+')), length((substring(klasse from '^\d+')::int + 1)::text)), '0')
		         || substring(klasse from '^\d+(.*)$') AS neue_klasse,
		       CASE
		         WHEN (substring(klasse from '^\d+')::int + 1) = 10 AND substring(klasse from '^\d+(.*)$') ILIKE '%h%' THEN true
		         WHEN (substring(klasse from '^\d+')::int + 1) = 11 AND substring(klasse from '^\d+(.*)$') ILIKE '%r%' THEN true
		         WHEN (substring(klasse from '^\d+')::int + 1) >= 14 THEN true
		         ELSE false
		       END AS abschluss
		FROM klassen_lehrer_mapping
		WHERE klasse ~ '^\d+'
		ORDER BY substring(klasse from '^\d+')::int DESC, klasse DESC`)
	if err != nil {
		return err
	}
	type zeile struct {
		alt, neu  string
		abschluss bool
	}
	var zeilen []zeile
	for rows.Next() {
		var z zeile
		if err := rows.Scan(&z.alt, &z.neu, &z.abschluss); err != nil {
			rows.Close()
			return err
		}
		zeilen = append(zeilen, z)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, z := range zeilen {
		if z.abschluss {
			if _, err := tx.Exec(ctx, `DELETE FROM klassen_lehrer_mapping WHERE klasse = $1`, z.alt); err != nil {
				return err
			}
			resp.MappingEntfernt++
			continue
		}
		var belegt bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM klassen_lehrer_mapping WHERE klasse = $1)`, z.neu).Scan(&belegt); err != nil {
			return err
		}
		if belegt {
			resp.MappingKonflikte = append(resp.MappingKonflikte, z.alt+" -> "+z.neu)
			continue
		}
		if _, err := tx.Exec(ctx, `UPDATE klassen_lehrer_mapping SET klasse = $1 WHERE klasse = $2`, z.neu, z.alt); err != nil {
			return err
		}
		resp.MappingVersetzt++
	}
	return nil
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
// 12 Stunden (ein zweiter Lauf würde alle Schüler +2 versetzen). ok=false: die
// Fehlerantwort (500 oder 409) wurde bereits geschrieben.
//
// Das Fenster ist bewusst großzügig: Der Schuljahreswechsel ist ein Batch, der einmal
// im Jahr läuft. Das frühere 10-Minuten-Fenster ließ genau den wahrscheinlichsten
// Fehler durch — der Admin bemerkt etwas, korrigiert, und klickt nach einer
// Viertelstunde erneut, wodurch die ganze Schule ein zweites Mal befördert wird. Ein
// legitimer Wiederholungslauf nach einem FEHLGESCHLAGENEN Wechsel bleibt möglich: Der
// Audit-Eintrag wird atomar mit dem Batch committet (finalisiereSchuljahreswechsel),
// ein abgebrochener Lauf hinterlässt also keinen Eintrag und keine Sperre.
func pruefeDoppellaufSchutz(ctx context.Context, tx pgx.Tx, w http.ResponseWriter) bool {
	var recentRuns int
	err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_logs
		WHERE aktion = 'SCHULJAHRESWECHSEL'
		  AND zeitstempel > NOW() - INTERVAL '12 hours'
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
