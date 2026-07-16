package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// InventurSessionDTO ist die Frontend-Sicht auf eine laufende Session.
type InventurSessionDTO struct {
	SessionID   string `json:"session_id"`
	Scope       string `json:"scope"`
	Label       string `json:"label"`
	GestartetAm string `json:"gestartet_am"`
	Erfasst     int    `json:"erfasst"`
	Erwartet    int    `json:"erwartet"`
}

// ListInventurSessionsHandler liefert alle laufenden Inventur-Sessions. Das Frontend
// zeigt sie an, damit eine Lehrkraft eine bereits laufende Inventur fortsetzt, statt
// (vergeblich) eine zweite im selben Scope zu starten.
//
// @Summary      List running inventory sessions
// @Tags         inventory
// @Produce      json
// @Success      200   {array}   InventurSessionDTO
// @Failure      500   {object}  map[string]string
// @Router       /inventur/sessions [get]
func (s *Server) ListInventurSessionsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		invRepo := repository.NewInventoryRepository(s.DB.Pool)

		sessions, err := invRepo.ListOffeneInventurSessions(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		out := make([]InventurSessionDTO, 0, len(sessions))
		for i := range sessions {
			erwartet, err := invRepo.ZaehleScope(ctx, sessions[i].SignatureID)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			out = append(out, InventurSessionDTO{
				SessionID:   sessions[i].ID,
				Scope:       sessions[i].ScopeType,
				Label:       sessions[i].ScopeLabel,
				GestartetAm: sessions[i].GestartetAm,
				Erfasst:     sessions[i].Erfasst,
				Erwartet:    erwartet,
			})
		}
		RespondJSON(w, http.StatusOK, out)
	}
}

// InventurAbortRequest benennt die abzubrechende Session.
type InventurAbortRequest struct {
	SessionID string `json:"session_id"`
}

// InventurAbortHandler verwirft eine Session ohne Verlustbuchung — für abgebrochene
// oder hängengebliebene Inventuren. Danach ist der Scope wieder für einen Neustart frei.
//
// @Summary      Abort an inventory session
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      InventurAbortRequest   true  "Session to abort"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /inventur/abort [post]
func (s *Server) InventurAbortHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req InventurAbortRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		if req.SessionID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("session_id fehlt"))
			return
		}

		ctx := r.Context()
		invRepo := repository.NewInventoryRepository(s.DB.Pool)
		if err := invRepo.AbortInventurSession(ctx, req.SessionID); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondSuccess(w)
	}
}
