package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// ladeExemplarFuerScan lädt die für die Inventur-Logik nötigen Exemplardetails.
// ok=false: die Fehlerantwort (404 bei unbekanntem Barcode, sonst 500) wurde bereits geschrieben.
func ladeExemplarFuerScan(ctx context.Context, invRepo *repository.InventoryRepository, w http.ResponseWriter, barcodeID string) (*repository.InventoryScanResult, bool) {
	res, err := invRepo.GetExemplarForInventoryScan(ctx, barcodeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, err)
			return nil, false
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, false
	}
	return res, true
}

// inventurWarnungen sammelt nicht-blockierende Hinweise zu einem gescannten Exemplar
// (aktuell ausgeliehen bzw. außerhalb des laufenden Inventur-Scopes).
func inventurWarnungen(isLent bool, inventurStatus *string) []string {
	var warnungen []string
	if isLent {
		warnungen = append(warnungen, "Buch ist laut System aktuell ausgeliehen.")
	}
	if inventurStatus == nil || *inventurStatus != "ausstehend" {
		warnungen = append(warnungen, "Buch gehört nicht zum aktuell gestarteten Inventur-Scope.")
	}
	return warnungen
}

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
		res, ok := ladeExemplarFuerScan(ctx, invRepo, w, req.BarcodeID)
		if !ok {
			return
		}

		// 2. Validate current state
		if res.IsAusgesondert {
			apierrors.SendHTTPError(w, http.StatusConflict, fmt.Errorf("exemplar %s ist bereits ausgesondert", req.BarcodeID))
			return
		}

		warnungen := inventurWarnungen(res.IsLent, res.InventurStatus)

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
