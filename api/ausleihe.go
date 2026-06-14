package api

import (
	"bibliothek/apierrors"
	"errors"

	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// GlobalExtendLMFRequest holds the JSON payload for extending LMF loans by class.
type GlobalExtendLMFRequest struct {
	Klasse              string `json:"klasse"`
	NeuesRueckgabeDatum string `json:"neues_rueckgabe_datum"` // Expected format "2006-01-02"
}

// ExtendLoanHandler extends the due date of a single loan by the standard book interval (e.g. 28 days).
func (s *Server) ExtendLoanHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ausleiheID := r.PathValue("ausleihe_id")
		if ausleiheID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("Fehlende ausleihe_id"))
			return
		}

		ctx := r.Context()

		// Retrieve standard extension interval
		settings, err := s.querySettings(ctx)
		extensionDays := 28 // Default if not configured
		if err == nil && settings.FristBuchTage > 0 {
			extensionDays = settings.FristBuchTage
		}

		q := `
			UPDATE ausleihen 
			SET rueckgabe_frist = rueckgabe_frist + ($2 * INTERVAL '1 day')
			WHERE id = $1 AND rueckgabe_am IS NULL
			RETURNING id, rueckgabe_frist
		`

		var id string
		var newFrist time.Time
		err = s.DB.Pool.QueryRow(ctx, q, ausleiheID, extensionDays).Scan(&id, &newFrist)
		if err != nil {
			if err == pgx.ErrNoRows {
				apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("Ausleihe nicht gefunden oder bereits zurückgegeben"))
				return
			}
			log.Printf("Fehler bei Einzel-Verlaengerung: %v", err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Interner Serverfehler"))
			return
		}

		RespondJSON(w, http.StatusOK, map[string]interface{}{
			"success":               true,
			"neues_rueckgabe_datum": newFrist,
		})
	}
}

// GlobalExtendLMFHandler performs a mass-extension for all LMF media for a specific class.
// It executes a single SQL transaction to ensure consistency.
func (s *Server) GlobalExtendLMFHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req GlobalExtendLMFRequest
		if !DecodeJSON(w, r, &req) {
			return
		}

		if req.Klasse == "" || req.NeuesRueckgabeDatum == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("klasse und neues_rueckgabe_datum sind erforderlich"))
			return
		}

		newDate, err := time.Parse("2006-01-02", req.NeuesRueckgabeDatum)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("Ungültiges Datumsformat (erwartet YYYY-MM-DD)"))
			return
		}
		// Set to the end of the day
		newDate = time.Date(newDate.Year(), newDate.Month(), newDate.Day(), 23, 59, 59, 0, newDate.Location())

		ctx := r.Context()
		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			log.Printf("Fehler beim Starten der Transaktion: %v", err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Interner Serverfehler"))
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		q := `
			UPDATE ausleihen a
			SET rueckgabe_frist = $1
			FROM schueler s, buecher_exemplare e, buecher_titel t
			WHERE a.schueler_id = s.id
			  AND a.exemplar_id = e.id
			  AND e.titel_id = t.id
			  AND a.rueckgabe_am IS NULL
			  AND s.klasse = $2
			  AND t.titel ILIKE 'LMF-%'
		`
		tag, err := tx.Exec(ctx, q, newDate, req.Klasse)
		if err != nil {
			log.Printf("Fehler beim globalen Verlängern: %v", err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Fehler beim Ausführen des Updates"))
			return
		}

		if err := tx.Commit(ctx); err != nil {
			log.Printf("Fehler beim Commit der Transaktion: %v", err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Interner Serverfehler"))
			return
		}

		RespondJSON(w, http.StatusOK, map[string]interface{}{
			"success":       true,
			"updated_count": tag.RowsAffected(),
		})
	}
}
