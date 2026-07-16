package api

import (
	"context"
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// InventurStartRequest holds the parameters needed to define the scope
// of a new physical stock-take (inventory).
type InventurStartRequest struct {
	Type        string `json:"type"` // "global" or "signature"
	SignatureID *int   `json:"signature_id,omitempty"`
}

// InventurStartResponse returns the new session and the number of copies expected.
type InventurStartResponse struct {
	SessionID string `json:"session_id"`
	Scope     string `json:"scope"`
	Label     string `json:"label"`
	Erwartet  int    `json:"erwartet"`
}

// validateInventurStart prüft Typ und (bei "signature") die erforderliche Signatur-ID.
func validateInventurStart(w http.ResponseWriter, req InventurStartRequest) bool {
	if req.Type != "global" && req.Type != "signature" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("invalid type, must be 'global' or 'signature'"))
		return false
	}
	if req.Type == "signature" && req.SignatureID == nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("signature_id is required when type is 'signature'"))
		return false
	}
	return true
}

// scopeLabelFuer bestimmt die Anzeigebezeichnung des Inventur-Scopes.
func scopeLabelFuer(ctx context.Context, invRepo *repository.InventoryRepository, req InventurStartRequest) (string, error) {
	if req.Type == "global" {
		return "Gesamtbestand", nil
	}
	return invRepo.SignaturBezeichnung(ctx, *req.SignatureID)
}

// InventurStartHandler eröffnet eine neue Inventur-Session für den gewählten Scope.
//
// Anders als früher wird KEIN globaler Zustand mehr zurückgesetzt: Jede Session hat
// ihren eigenen, session-gebundenen Fortschritt. Läuft für denselben Scope bereits
// eine Session, antwortet der Handler mit 409 — der bisherige stille globale Reset
// (der den Fortschritt eines parallel scannenden Kollegen löschte) entfällt.
//
// @Summary      Start an inventory session
// @Description  Opens a new inventory session for the chosen scope (global or per signature).
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      InventurStartRequest   true  "Scope configuration"
// @Success      200   {object}  InventurStartResponse
// @Failure      400   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /inventur/start [post]
func (s *Server) InventurStartHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req InventurStartRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		if !validateInventurStart(w, req) {
			return
		}

		ctx := r.Context()
		var benutzerID string
		if claims, ok := auth.GetClaims(ctx); ok {
			benutzerID = claims.UserID
		}

		invRepo := repository.NewInventoryRepository(s.DB.Pool)

		label, err := scopeLabelFuer(ctx, invRepo, req)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		session, err := invRepo.CreateInventurSession(ctx, req.Type, req.SignatureID, label, benutzerID)
		if err != nil {
			if errors.Is(err, repository.ErrInventurLaeuftBereits) {
				apierrors.SendHTTPError(w, http.StatusConflict, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		erwartet, err := invRepo.GetInventurSession(ctx, session.ID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, InventurStartResponse{
			SessionID: session.ID,
			Scope:     session.ScopeType,
			Label:     session.ScopeLabel,
			Erwartet:  erwartet.Erwartet,
		})
	}
}
