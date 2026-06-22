package api

import (
	"errors"
	"fmt"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// InventurScanRequest is the payload for checking in an item during inventory.
type InventurScanRequest struct {
	BarcodeID string `json:"barcode_id"`
}

// InventurScanResponse provides feedback to the frontend after a scan attempt.
type InventurScanResponse struct {
	BarcodeID string   `json:"barcode_id"`
	Titel     string   `json:"titel"`
	CoverURL  string   `json:"cover_url,omitempty"`
	Status    string   `json:"status"` // e.g. "erfasst"
	Warnungen []string `json:"warnungen,omitempty"`
}

// InventurScanHandler registers copy scans in the active inventory list.
// @Summary      Scan a copy during inventory
// @Description  Records that a physical copy was physically present during a stock-take.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      InventurScanRequest   true  "Barcode to check in"
// @Success      200   {object}  InventurScanResponse
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /inventur/scan [post]
func (s *Server) InventurScanHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req InventurScanRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		ctx := r.Context()
		invRepo := repository.NewInventoryRepository(s.DB.Pool)

		// 1. Fetch details required for inventory logic
		res, err := invRepo.GetExemplarForInventoryScan(ctx, req.BarcodeID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 2. Validate current state
		if res.IsAusgesondert {
			apierrors.SendHTTPError(w, http.StatusConflict, fmt.Errorf("exemplar %s ist bereits ausgesondert", req.BarcodeID))
			return
		}

		var warnungen []string
		if res.IsLent {
			warnungen = append(warnungen, "Buch ist laut System aktuell ausgeliehen.")
		}
		if res.InventurStatus == nil || *res.InventurStatus != "ausstehend" {
			warnungen = append(warnungen, "Buch gehört nicht zum aktuell gestarteten Inventur-Scope.")
		}

		// 3. Register the scan
		if err := invRepo.MarkExemplarScanned(ctx, res.CopyID); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, InventurScanResponse{
			BarcodeID: req.BarcodeID,
			Titel:     res.Title,
			CoverURL:  res.CoverURL,
			Status:    "erfasst",
			Warnungen: warnungen,
		})
	}
}
