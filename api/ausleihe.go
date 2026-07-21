package api

import (
	"bibliothek/apierrors"
	"bibliothek/db"
	"errors"

	"log"
	"net/http"
	"time"

	"bibliothek/pkg/lmf"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// GlobalExtendLMFRequest holds the JSON payload for extending LMF loans by class.
type GlobalExtendLMFRequest struct {
	Klasse              string `json:"klasse"`
	NeuesRueckgabeDatum string `json:"neues_rueckgabe_datum"` // Expected format "2006-01-02"
}

// OverrideDueDateRequest holds the JSON payload for manually overriding a due date.
type OverrideDueDateRequest struct {
	FaelligAm string `json:"faellig_am" validate:"required"` // ISO 8601 or YYYY-MM-DD
}

// ExtendLoanHandler extends the due date of a single loan by the standard book interval (e.g. 28 days).
func (s *Server) ExtendLoanHandler(settingsRepo repository.SystemSettingsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.handleExtendLoan(w, r, settingsRepo)
	}
}

// handleExtendLoan verlängert eine einzelne Ausleihe. Als Top-Level-Methode ausgelagert,
// damit die Frühabbrüche nicht zusätzlich als Closure-Verschachtelung zählen (S3776).
func (s *Server) handleExtendLoan(w http.ResponseWriter, r *http.Request, settingsRepo repository.SystemSettingsRepository) {
	ausleiheID := r.PathValue("ausleihe_id")
	if ausleiheID == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende ausleihe_id"))
		return
	}

	ctx := r.Context()

	// Sanktions-Konsistenz: Ist das Buch an einen gesperrten Schüler verliehen, darf die
	// Frist nicht verlängert werden — die Sperre soll zur Rückgabe zwingen, nicht durch
	// eine Verlängerung ausgehebelt werden. Lehrer-/Handapparat-Ausleihen (kein
	// schueler_id → LEFT JOIN liefert NULL) sind nicht betroffen.
	var gesperrt bool
	var blockReason string
	if errChk := s.DB.Pool.QueryRow(ctx, `
			SELECT COALESCE(s.ist_gesperrt, false) OR COALESCE(s.is_manually_blocked, false),
			       COALESCE(s.block_reason, '')
			FROM ausleihen a
			LEFT JOIN schueler s ON s.id = a.schueler_id
			WHERE a.id = $1 AND a.rueckgabe_am IS NULL
		`, ausleiheID).Scan(&gesperrt, &blockReason); errChk != nil {
		if errChk == pgx.ErrNoRows {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("ausleihe nicht gefunden oder bereits zurückgegeben"))
			return
		}
		log.Printf("Fehler bei Sperr-Prüfung (Einzel-Verlängerung): %v", errChk)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return
	}
	if gesperrt {
		msg := "Verlängerung nicht möglich: Ausleihe gesperrt"
		if blockReason != "" {
			msg += " (" + blockReason + ")"
		}
		apierrors.SendHTTPError(w, http.StatusForbidden, errors.New(msg))
		return
	}

	// Retrieve standard extension interval
	settings, err := settingsRepo.GetSettings(ctx)
	extensionDays := 28 // Default if not configured
	if err == nil && settings.FristBuchTage > 0 {
		extensionDays = settings.FristBuchTage
	}

	// Die Verlängerung setzt die Mahn-Eskalation zurück, SOBALD die neue Frist wieder in
	// der Zukunft liegt. Ohne das würde ein nach der 1. Mahnung verlängertes und erneut
	// überzogenes Buch die 1. Stufe überspringen und sofort in Stufe 2 (Rechnung)
	// eskalieren. Bleibt die Frist trotz Verlängerung in der Vergangenheit (stark
	// überzogene Ausleihe), bleibt die Mahnstufe erhalten — dann ist die Eskalation
	// berechtigt weiterzuführen, nicht zurückzusetzen.
	q := `
			UPDATE ausleihen
			SET rueckgabe_frist = rueckgabe_frist + ($2 * INTERVAL '1 day'),
			    mahnstufe = CASE WHEN rueckgabe_frist + ($2 * INTERVAL '1 day') > CURRENT_TIMESTAMP THEN 0 ELSE mahnstufe END,
			    letztes_mahndatum = CASE WHEN rueckgabe_frist + ($2 * INTERVAL '1 day') > CURRENT_TIMESTAMP THEN NULL ELSE letztes_mahndatum END
			WHERE id = $1 AND rueckgabe_am IS NULL
			RETURNING id, rueckgabe_frist
		`

	var id string
	var newFrist time.Time
	err = s.DB.Pool.QueryRow(ctx, q, ausleiheID, extensionDays).Scan(&id, &newFrist)
	if err != nil {
		if err == pgx.ErrNoRows {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("ausleihe nicht gefunden oder bereits zurückgegeben"))
			return
		}
		log.Printf("Fehler bei Einzel-Verlaengerung: %v", err)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return
	}

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":               true,
		"neues_rueckgabe_datum": newFrist,
	})
}

// OverrideDueDateHandler manually overrides the due date of an active loan.
func (s *Server) OverrideDueDateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ausleiheID := r.PathValue("id")
		if ausleiheID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende Ausleihe-ID"))
			return
		}

		var req OverrideDueDateRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		newDate, err := time.Parse(time.RFC3339, req.FaelligAm)
		if err != nil {
			// Fallback auf einfaches Datum YYYY-MM-DD
			newDate, err = time.Parse("2006-01-02", req.FaelligAm)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültiges Datumsformat (erwartet ISO 8601 oder YYYY-MM-DD)"))
				return
			}
			newDate = time.Date(newDate.Year(), newDate.Month(), newDate.Day(), 23, 59, 59, 0, newDate.Location())
		}

		ctx := r.Context()
		// Wie bei der regulären Verlängerung: Eine neue Frist in der Zukunft macht die
		// Ausleihe wieder "nicht überfällig" und setzt die Mahn-Eskalation zurück. Ein
		// vorgezogenes Datum (Rückruf) lässt die Mahnstufe unberührt.
		q := `
			UPDATE ausleihen
			SET rueckgabe_frist = $1,
			    mahnstufe = CASE WHEN $1 > CURRENT_TIMESTAMP THEN 0 ELSE mahnstufe END,
			    letztes_mahndatum = CASE WHEN $1 > CURRENT_TIMESTAMP THEN NULL ELSE letztes_mahndatum END
			WHERE id = $2 AND rueckgabe_am IS NULL
			RETURNING id, rueckgabe_frist
		`

		var id string
		var newFrist time.Time
		err = s.DB.Pool.QueryRow(ctx, q, newDate, ausleiheID).Scan(&id, &newFrist)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("ausleihe nicht gefunden oder bereits zurückgegeben"))
				return
			}
			log.Printf("Fehler bei manueller Frist-Überschreibung: %v", err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
			return
		}

		RespondJSON(w, http.StatusOK, map[string]interface{}{
			"success":    true,
			"faellig_am": newFrist,
		})
	}
}

// GlobalExtendLMFHandler performs a mass-extension for all LMF media for a specific class.
// It executes a single SQL transaction to ensure consistency.
func (s *Server) GlobalExtendLMFHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req GlobalExtendLMFRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		if req.Klasse == "" || req.NeuesRueckgabeDatum == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("klasse und neues_rueckgabe_datum sind erforderlich"))
			return
		}

		newDate, err := time.Parse("2006-01-02", req.NeuesRueckgabeDatum)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültiges Datumsformat (erwartet YYYY-MM-DD)"))
			return
		}
		// Set to the end of the day
		newDate = time.Date(newDate.Year(), newDate.Month(), newDate.Day(), 23, 59, 59, 0, newDate.Location())

		ctx := r.Context()
		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			log.Printf("Fehler beim Starten der Transaktion: %v", err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
			return
		}
		defer db.SafeRollback(ctx, tx)

		// Mass-Verlängerung setzt zugleich die Mahn-Eskalation der betroffenen Ausleihen
		// zurück (sofern die neue Frist in der Zukunft liegt) — sonst würde ein ganzer
		// Klassensatz nach der Verlängerung fälschlich auf der alten Mahnstufe weiterlaufen.
		q := `
			UPDATE ausleihen a
			SET rueckgabe_frist = $1,
			    mahnstufe = CASE WHEN $1 > CURRENT_TIMESTAMP THEN 0 ELSE a.mahnstufe END,
			    letztes_mahndatum = CASE WHEN $1 > CURRENT_TIMESTAMP THEN NULL ELSE a.letztes_mahndatum END
			FROM schueler s, buecher_exemplare e, buecher_titel t
			WHERE a.schueler_id = s.id
			  AND a.exemplar_id = e.id
			  AND e.titel_id = t.id
			  AND a.rueckgabe_am IS NULL
			  AND s.deleted_at IS NULL
			  -- Gesperrte Schüler von der Massen-Verlängerung ausnehmen: die Sperre soll zur
			  -- Rückgabe zwingen, nicht durch eine Fristverlängerung ausgehebelt werden.
			  AND s.ist_gesperrt = false
			  AND COALESCE(s.is_manually_blocked, false) = false
			  AND s.klasse = $2
			  AND ` + lmf.SQLBedingung("t.titel") + `
		`
		tag, err := tx.Exec(ctx, q, newDate, req.Klasse)
		if err != nil {
			log.Printf("Fehler beim globalen Verlängern: %v", err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("fehler beim Ausführen des Updates"))
			return
		}

		if err := tx.Commit(ctx); err != nil {
			log.Printf("Fehler beim Commit der Transaktion: %v", err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
			return
		}

		RespondJSON(w, http.StatusOK, map[string]interface{}{
			"success":       true,
			"updated_count": tag.RowsAffected(),
		})
	}
}
