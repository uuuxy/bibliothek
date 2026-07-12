package api

import (
	"context"
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/repository"
)

// validateInventurStart prüft Typ und (bei "signature") die erforderliche Signatur-ID.
// ok=false: die Fehlerantwort wurde bereits geschrieben.
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

// setzeInventurUmfang setzt den Inventur-Status zurück und markiert den gewählten
// Umfang (global oder je Signatur) als 'ausstehend'; liefert die Anzahl erwarteter
// Exemplare.
func setzeInventurUmfang(ctx context.Context, invRepo *repository.InventoryRepository, req InventurStartRequest) (int, error) {
	// 1. Reset all old states globally
	if err := invRepo.ResetInventoryStatus(ctx); err != nil {
		return 0, err
	}

	// 2. Set 'ausstehend' for the targeted scope
	if req.Type == "global" {
		return invRepo.SetInventoryScopeGlobal(ctx)
	}
	return invRepo.SetInventoryScopeSignature(ctx, *req.SignatureID)
}

// InventurStartRequest holds the parameters needed to define the scope
// of a new physical stock-take (inventory).
type InventurStartRequest struct {
	Type        string `json:"type"` // "global" or "signature"
	SignatureID *int   `json:"signature_id,omitempty"`
}

// InventurStartResponse returns the number of copies expected for the started inventory.
type InventurStartResponse struct {
	Scope    string `json:"scope"`
	Erwartet int    `json:"erwartet"`
}

// InventurStartHandler sets the scope for a new inventory session.
// @Summary      Start an inventory session
// @Description  Resets old inventory states and sets 'ausstehend' for the chosen scope via the central InventoryRepository.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      InventurStartRequest   true  "Scope configuration"
// @Success      200   {object}  InventurStartResponse
// @Failure      400   {object}  map[string]string
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

		// Begin transaction to ensure reset and scope setting is atomic
		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer db.SafeRollback(ctx, tx)

		invRepo := repository.NewInventoryRepository(tx)
		count, err := setzeInventurUmfang(ctx, invRepo, req)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, InventurStartResponse{
			Scope:    req.Type,
			Erwartet: count,
		})
	}
}
