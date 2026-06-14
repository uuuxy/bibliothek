package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"

	"github.com/jackc/pgx/v5"
)

// InventoryScanRequest is the payload for checking in an item during inventory.
type InventoryScanRequest struct {
	BarcodeID string `json:"barcode_id"`
}

// InventoryScanResponse yields the inventory status of the physical copy.
type InventoryScanResponse struct {
	BarcodeID       string `json:"barcode_id"`
	Titel           string `json:"titel"`
	CoverURL        string `json:"cover_url,omitempty"`
	ImRegalErwartet bool   `json:"im_regal_erwartet"`
	Status          string `json:"status"` // always "Geprüft" on success
}

// ScanInventoryHandler registers copy scans in active inventory lists.
// @Summary      Scan a copy during inventory
// @Description  Records that a physical copy was physically present during a stock-take and updates inventur_geprueft_am.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      InventoryScanRequest   true  "Barcode to check in"
// @Success      200   {object}  InventoryScanResponse
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /inventur/scan [post]
func (s *Server) ScanInventoryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req InventoryScanRequest
		if !DecodeJSON(w, r, &req) {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var copyID, title, coverURL string
		var isLent, isAusgesondert bool

		// Check copy metadata and current loan status
		query := `
			SELECT e.id, t.titel, coalesce(t.cover_url, ''), e.ist_ausgesondert, EXISTS (
				SELECT 1 FROM ausleihen a 
				WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
			) AS is_lent
			FROM buecher_exemplare e
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE e.barcode_id = $1
			LIMIT 1
		`
		err := s.DB.Pool.QueryRow(ctx, query, req.BarcodeID).Scan(&copyID, &title, &coverURL, &isAusgesondert, &isLent)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if isAusgesondert {
			apierrors.SendHTTPError(w, http.StatusConflict, fmt.Errorf("exemplar %s ist ausgesondert und wird nicht inventarisiert", req.BarcodeID))
			return
		}

		// Update inventory audit timestamp on copy
		updateQuery := `
			UPDATE buecher_exemplare
			SET inventur_geprueft_am = CURRENT_TIMESTAMP,
			    aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $1
		`
		_, err = s.DB.Pool.Exec(ctx, updateQuery, copyID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, InventoryScanResponse{
			BarcodeID:       req.BarcodeID,
			Titel:           title,
			CoverURL:        coverURL,
			ImRegalErwartet: !isLent, // Should be on shelf if not currently checked out
			Status:          "Geprüft",
		})
	}
}

// @Summary      Mark missing title copies as lost
// @Description  Marks a batch of copy IDs for a specific title as lost/ausgesondert
// @Tags         Inventur
// @Accept       json
// @Produce      json
// @Param        id path string true "Title ID"
// @Param        body body object{exemplar_ids=[]string} true "Copy IDs to mark as lost"
// @Success      200 {object} object
// @Router       /inventur/titel/{id}/verlust-batch [post]
func (s *Server) TitleVerlustBatchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		titelID := r.PathValue("id")
		if titelID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing title id"))
			return
		}

		var req struct {
			ExemplarIDs []string `json:"exemplar_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("invalid json body"))
			return
		}

		if len(req.ExemplarIDs) == 0 {
			RespondJSON(w, http.StatusOK, map[string]any{"updated": 0})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		// Create the array of IDs
		ids := make([]any, len(req.ExemplarIDs))
		for i, id := range req.ExemplarIDs {
			ids[i] = id
		}

		// Prepare the query using ANY($1::uuid[])
		query := `
			UPDATE buecher_exemplare
			SET ist_ausleihbar = false,
			    ist_ausgesondert = true,
			    zustand_notiz = 'Verloren (Inventur Regal-Scan)',
			    inventur_geprueft_am = CURRENT_TIMESTAMP
			WHERE titel_id = $1 AND id = ANY($2::uuid[])
		`

		tag, err := tx.Exec(ctx, query, titelID, req.ExemplarIDs)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Update total verfuegbar count for this title
		_, _ = tx.Exec(ctx, "SELECT update_verfuegbar_count($1)", titelID)

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, map[string]any{
			"updated": tag.RowsAffected(),
		})
	}
}
