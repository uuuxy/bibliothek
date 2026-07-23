package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// InventurFinishRequest benennt die abzuschließende Session.
type InventurFinishRequest struct {
	SessionID string `json:"session_id"`
}

// InventurFinishResponse contains the outcome statistics of the completed inventory.
type InventurFinishResponse struct {
	VerlorenGemeldet int `json:"verloren_gemeldet"`
}

// InventurFinishHandler schließt eine Inventur-Session ab: Alle im Scope erwarteten,
// aber in DIESER Session nicht erfassten (und nicht verliehenen) Exemplare werden als
// Verlust markiert. Der Fortschritt paralleler Sessions bleibt unberührt.
//
// @Summary      Finalize an inventory session
// @Description  Marks scope items not scanned in this session as lost and closes it.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      InventurFinishRequest   true  "Session to finalize"
// @Success      200   {object}  InventurFinishResponse
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /inventur/finish [post]
func (s *Server) InventurFinishHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req InventurFinishRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		if req.SessionID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("session_id fehlt"))
			return
		}

		ctx := r.Context()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer db.SafeRollback(ctx, tx)

		invRepo := repository.NewInventoryRepository(tx)

		// Session in derselben Transaktion sperren/prüfen: verhindert doppelten
		// Abschluss und liefert die signature_id für den Scope des Verlust-Updates.
		session, err := invRepo.LadeInventurSession(ctx, req.SessionID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("keine laufende Inventur zu dieser Session"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		count, err := invRepo.FinishInventurSession(ctx, session.ID, session.Scope())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, InventurFinishResponse{VerlorenGemeldet: count})
	}
}
