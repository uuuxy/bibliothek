package api

import (
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/repository"
)

// InventurFinishResponse contains the outcome statistics of the completed inventory.
type InventurFinishResponse struct {
	VerlorenGemeldet int `json:"verloren_gemeldet"`
}

// InventurFinishHandler concludes an inventory session.
// @Summary      Finalize inventory
// @Description  Marks all 'ausstehend' books as 'verloren' and resets inventory states using the central InventoryRepository.
// @Tags         inventory
// @Produce      json
// @Success      200   {object}  InventurFinishResponse
// @Failure      500   {object}  map[string]string
// @Router       /inventur/finish [post]
func (s *Server) InventurFinishHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Begin transaction to ensure the marking and resetting is atomic
		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer db.SafeRollback(ctx, tx)

		invRepo := repository.NewInventoryRepository(tx)

		// Mark remaining 'ausstehend' items as lost and reset the global state
		count, err := invRepo.MarkRemainingAsLostAndReset(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, InventurFinishResponse{
			VerlorenGemeldet: count,
		})
	}
}
