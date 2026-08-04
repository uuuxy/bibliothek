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

// AbgeschlosseneInventurDTO ist die Frontend-Sicht auf eine fertige Inventur. Sie
// trägt nur, was für die Auswahl nötig ist: wann, welcher Bereich, wie viele Verluste.
type AbgeschlosseneInventurDTO struct {
	SessionID       string `json:"session_id"`
	Label           string `json:"label"`
	AbgeschlossenAm string `json:"abgeschlossen_am"`
	Erfasst         int    `json:"erfasst"`
	Verluste        int    `json:"verluste"`
}

// ListAbgeschlosseneInventurenHandler liefert die zuletzt abgeschlossenen Inventuren.
//
// Sie sind der fehlende Schlüssel zu GET /api/inventur/fehlbestand: Der Endpunkt
// verlangt eine session_id, und die war aus der Oberfläche nicht zu bekommen, weil die
// Session-Liste nur laufende Inventuren zeigt. Der Fehlbestand lag damit ausschließlich
// im Arbeitsspeicher des Browsers, der die Inventur abgeschlossen hatte — nach einem
// Neuladen oder an einem zweiten Arbeitsplatz war er weg, obwohl die Daten in
// inventur_verluste dauerhaft gespeichert sind.
//
// @Summary      List completed inventory sessions
// @Tags         inventory
// @Produce      json
// @Success      200   {array}   AbgeschlosseneInventurDTO
// @Failure      500   {object}  map[string]string
// @Router       /inventur/abgeschlossen [get]
func (s *Server) ListAbgeschlosseneInventurenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		invRepo := repository.NewInventoryRepository(s.DB.Pool)

		sessions, err := invRepo.ListAbgeschlosseneInventurSessions(r.Context(), 10)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		out := make([]AbgeschlosseneInventurDTO, 0, len(sessions))
		for i := range sessions {
			abgeschlossen := ""
			if sessions[i].AbgeschlossenAm != nil {
				abgeschlossen = *sessions[i].AbgeschlossenAm
			}
			out = append(out, AbgeschlosseneInventurDTO{
				SessionID:       sessions[i].ID,
				Label:           sessions[i].ScopeLabel,
				AbgeschlossenAm: abgeschlossen,
				Erfasst:         sessions[i].Erfasst,
				Verluste:        sessions[i].Verluste,
			})
		}
		RespondJSON(w, http.StatusOK, out)
	}
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
			erwartet, err := invRepo.ZaehleScope(ctx, sessions[i].Scope())
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
