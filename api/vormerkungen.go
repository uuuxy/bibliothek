package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// CreateVormerkungRequest is the body for POST /api/vormerkungen.
type CreateVormerkungRequest struct {
	TitelID    string `json:"titel_id" validate:"required"`
	Notiz      string `json:"notiz,omitempty"`
	SchuelerID string `json:"schueler_id,omitempty"`
}

// ListVormerkungHandler handles GET /api/vormerkungen?titel_id=...
func (s *Server) ListVormerkungHandler(vormerkungRepo repository.VormerkungRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()

		titelID := r.URL.Query().Get("titel_id")
		schuelerID := r.URL.Query().Get("schueler_id")

		result, err := vormerkungRepo.List(ctx, titelID, schuelerID)
		if err != nil {
			return apierrors.Internal("Fehler beim Abrufen der Vormerkungen", err)
		}
		if result == nil {
			result = []repository.Vormerkung{}
		}

		RespondJSON(w, http.StatusOK, result)
		return nil
	})
}

// CreateVormerkungHandler handles POST /api/vormerkungen.
func (s *Server) CreateVormerkungHandler(vormerkungRepo repository.VormerkungRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req CreateVormerkungRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}

		ctx := r.Context()

		id, err := vormerkungRepo.Create(ctx, req.TitelID, req.Notiz, req.SchuelerID)
		if err != nil {
			// Fachlicher Konflikt (409): Der Schüler hat den Titel bereits selbst ausgeliehen
			// und darf ihn nicht zusätzlich vormerken — kein Serverfehler.
			if errors.Is(err, repository.ErrTitelBereitsAusgeliehen) {
				return apierrors.Conflict(err.Error(), err)
			}
			return apierrors.Internal("Fehler beim Erstellen der Vormerkung", err)
		}

		RespondJSON(w, http.StatusCreated, map[string]string{"id": id})
		return nil
	})
}

// DeleteVormerkungHandler handles DELETE /api/vormerkungen/{id}.
func (s *Server) DeleteVormerkungHandler(vormerkungRepo repository.VormerkungRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("ID fehlt", nil)
		}

		ctx := r.Context()

		if err := vormerkungRepo.Delete(ctx, id); err != nil {
			return apierrors.Internal("Fehler beim Löschen der Vormerkung", err)
		}

		w.Header().Set("Content-Type", "application/json")
		httpresp.Write(w, []byte(`{"status":"gelöscht"}`))
		return nil
	})
}
