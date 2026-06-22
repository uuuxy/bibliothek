package api

import (
	"context"
	"net/http"

	"bibliothek/apierrors"
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
		_, _ = w.Write([]byte(`{"status":"gelöscht"}`))
		return nil
	})
}

// checkVormerkung returns the earliest pending reservation for a given titel_id, or nil if none.
// Since it's used internally by Server, we pass the context and the repo.
// Note: Some places in api/ might call s.checkVormerkung(ctx, titelID), which we now must refactor slightly,
// but since we are doing dependency injection, it's safer to just let the repo handle it.
func (s *Server) checkVormerkung(ctx context.Context, titelID string) (*repository.Vormerkung, error) {
	// Temporarily create a repository instance here if called internally where repo is not injected.
	repo := repository.NewVormerkungRepository(s.DB.Pool)
	return repo.GetEarliestPending(ctx, titelID)
}
