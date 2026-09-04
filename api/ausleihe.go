package api

import (
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	"errors"

	"context"
	"log"
	"net/http"
	"time"

	"bibliothek/internal/service"
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
	gesperrt, blockReason, errChk := s.checkAusleiheGesperrt(ctx, ausleiheID)
	if errChk != nil {
		if errors.Is(errChk, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("ausleihe nicht gefunden oder bereits zurückgegeben"))
			return
		}
		log.Printf("Fehler bei Sperr-Prüfung (Einzel-Verlängerung): %v", errChk)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
		return
	}
	if gesperrt {
		msg := "Verlängerung nicht möglich: Ausleihe gesperrt"
		// Der Sperr-Freitext ist Verwaltungsinformation (PII-Matrix Stufe 2) —
		// dieselbe view_students-Grenze wie bei /api/action (ohneSperrgrund).
		if blockReason != "" && s.BesitztRecht(r, "view_students") {
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

	// Gerechnet wird ab dem SPÄTEREN von altem Fristende und heute.
	//
	// Vorher stand hier rueckgabe_frist + Intervall, also immer ab dem alten Fristende. Bei
	// einer Ausleihe, die länger überfällig war als die Leihfrist, landete die neue Frist
	// damit WIEDER in der Vergangenheit: Das Buch blieb „Überfällig", die Zeile sah
	// unverändert aus, und die Erfolgsmeldung nannte ein Datum, das schon vorbei war. Aus
	// dem Betrieb gemeldet als „es kommt ein Datum in der Vergangenheit, sonst passiert
	// nichts" — eine Verlängerung, die nicht verlängert, ist keine.
	//
	// GREATEST statt schlicht CURRENT_DATE + Intervall: Wer VOR Fristablauf verlängert, soll
	// die verbleibenden Tage nicht verlieren. Für diesen Fall bleibt es beim bisherigen
	// Verhalten; nur der überfällige Fall rechnet ab heute.
	//
	// Damit liegt die neue Frist immer in der Zukunft, und die Mahn-Eskalation wird immer
	// zurückgesetzt. Das ist beabsichtigt: Wer verlängert, gewährt ausdrücklich neue Zeit —
	// ohne den Reset übersprünge dasselbe Buch beim nächsten Überziehen die 1. Mahnstufe
	// und eskalierte sofort in Stufe 2 (Rechnung).
	q := `
			UPDATE ausleihen
			SET rueckgabe_frist = GREATEST(rueckgabe_frist, CURRENT_TIMESTAMP) + ($2 * INTERVAL '1 day'),
			    mahnstufe = 0,
			    letztes_mahndatum = NULL
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
func (s *Server) OverrideDueDateHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("Sitzungs-Information fehlt oder ist abgelaufen"))
			return
		}

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
			// Tagesende in der Schulzeitzone (Berlin), nicht roh in UTC: sonst wäre die
			// überschriebene Frist 1–2 h später fällig als eine regulär berechnete zum
			// selben Datum (dieselbe EINZIGE Definition wie service.CalculateDueDate).
			newDate = service.TagesEndeInSchulzeitzone(newDate)
		}

		ctx := r.Context()

		// Sanktions-Konsistenz mit ExtendLoanHandler und GlobalExtendLMFHandler: Ist das
		// Buch an einen gesperrten Schüler verliehen, darf die Frist nicht in die ZUKUNFT
		// verschoben werden — das machte die Ausleihe wieder "nicht überfällig", setzte
		// die Mahn-Eskalation zurück und hebelte so die Auto-Sperre aus (die aus überfälligen
		// Ausleihen berechnet wird). Genau diese Umgehung war bis 18.08.2026 über diesen
		// Endpunkt möglich, während die beiden Verlängerungs-Endpunkte sie ausdrücklich
		// verbieten. Ein VORGEZOGENES Datum (Rückruf) bleibt auch bei Sperre erlaubt:
		// Es hebt die Sanktion nicht auf, im Gegenteil.
		if newDate.After(time.Now()) {
			gesperrt, _, errChk := s.checkAusleiheGesperrt(ctx, ausleiheID)
			if errChk != nil {
				if errors.Is(errChk, pgx.ErrNoRows) {
					apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("ausleihe nicht gefunden oder bereits zurückgegeben"))
					return
				}
				log.Printf("Fehler bei Sperr-Prüfung (Frist-Überschreibung): %v", errChk)
				apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
				return
			}
			if gesperrt {
				apierrors.SendHTTPError(w, http.StatusForbidden,
					errors.New("gesperrte Ausleihe: die Frist kann nicht in die Zukunft verschoben werden — die Sperre soll zur Rückgabe zwingen"))
				return
			}
		}

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

		// Ein manuell überschriebenes Fälligkeitsdatum ist ein Eingriff in eine Sanktion
		// (Mahnfristen, Auto-Sperre) — er gehört revisionssicher protokolliert, genau wie
		// der Checkout-Override (OVERRIDE_BLOCK). Best effort: Der Eingriff gilt, auch wenn
		// das Log klemmt, aber der Fehlversuch steht wenigstens im Server-Log.
		if logErr := auditRepo.LogAdminAktion(ctx, claims.UserID, "FRIST_OVERRIDE", "", map[string]any{
			"ausleihe_id": id,
			"neue_frist":  newFrist.Format(time.RFC3339),
		}); logErr != nil {
			log.Printf("Audit für Frist-Überschreibung fehlgeschlagen: %v", logErr)
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
		// Tagesende in der Schulzeitzone (Berlin) — dieselbe Definition wie jede andere
		// Frist (service.TagesEndeInSchulzeitzone), nicht roh 23:59:59 UTC.
		newDate = service.TagesEndeInSchulzeitzone(newDate)

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
			  -- Über den Normal-Schlüssel (Migration 079/087): „5a" aus einem Formular und
			  -- „05A" als registrierte Anzeigeform meinen dieselbe Klasse.
			  AND klassen_normkey(s.klasse) = klassen_normkey($2)
			  AND t.ist_lernmittel
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

// checkAusleiheGesperrt prüft, ob die angegebene Ausleihe an einen gesperrten Schüler vergeben ist.
func (s *Server) checkAusleiheGesperrt(ctx context.Context, ausleiheID string) (bool, string, error) {
	var gesperrt bool
	var blockReason string
	err := s.DB.Pool.QueryRow(ctx, `
		SELECT COALESCE(s.ist_gesperrt, false) OR COALESCE(s.is_manually_blocked, false),
		       COALESCE(s.block_reason, '')
		FROM ausleihen a
		LEFT JOIN schueler s ON s.id = a.schueler_id
		WHERE a.id = $1 AND a.rueckgabe_am IS NULL
	`, ausleiheID).Scan(&gesperrt, &blockReason)
	return gesperrt, blockReason, err
}
