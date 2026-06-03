package api

// inventory.go — Handlers for physical inventory scanning and Fehlbestand reporting.
// Inventory scanning (ScanInventoryHandler) marks copies as checked during a stock-take.
// Fehlbestand (GetFehlbestandHandler) surfaces copies that are unexpectedly absent from the shelf.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
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
			apierrors.SendHTTPError(w, http.StatusConflict, fmt.Errorf("Exemplar %s ist ausgesondert und wird nicht inventarisiert", req.BarcodeID))
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

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(InventoryScanResponse{
			BarcodeID:       req.BarcodeID,
			Titel:           title,
			CoverURL:        coverURL,
			ImRegalErwartet: !isLent, // Should be on shelf if not currently checked out
			Status:          "Geprüft",
		})
	}
}

// FehlbestandEntry represents one copy missing from the expected shelf position.
type FehlbestandEntry struct {
	ID                 string     `json:"id"`
	BarcodeID          string     `json:"barcode_id"`
	ZustandNotiz       string     `json:"zustand_notiz"`
	InventurGeprueftAm *time.Time `json:"inventur_geprueft_am"`
	Titel              string     `json:"titel"`
	Autor              string     `json:"autor"`
	CoverURL           string     `json:"cover_url,omitempty"`
	ISBN               string     `json:"isbn,omitempty"`
}

// GetFehlbestandHandler returns copies that are expected on the shelf but have not been
// scanned during inventory for more than `tage` days (default: 30).
// Only active (non-ausgesondert, ausleihbar) copies that are not currently on loan are considered.
// @Summary      Get shelf discrepancies
// @Description  Lists physical copies overdue for inventory scanning (expected on shelf but not confirmed present).
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        tage  query     int  false  "Days since last scan before considered missing (default: 30, max: 3650)"
// @Success      200   {array}   FehlbestandEntry
// @Failure      500   {object}  map[string]string
// @Router       /inventur/fehlbestand [get]
func (s *Server) GetFehlbestandHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tage := 30
		if v := r.URL.Query().Get("tage"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 3650 {
				tage = n
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		rows, err := s.DB.Pool.Query(ctx, `
			SELECT e.id, e.barcode_id, coalesce(e.zustand_notiz, ''), e.inventur_geprueft_am,
			       t.titel, coalesce(t.autor, ''), coalesce(t.cover_url, ''), coalesce(t.isbn, '')
			FROM buecher_exemplare e
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE e.ist_ausgesondert = false
			  AND e.ist_ausleihbar = true
			  AND NOT EXISTS (
			      SELECT 1 FROM ausleihen a
			      WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
			  )
			  AND (
			      e.inventur_geprueft_am IS NULL
			      OR e.inventur_geprueft_am < CURRENT_TIMESTAMP - ($1 * INTERVAL '1 day')
			  )
			ORDER BY e.inventur_geprueft_am ASC NULLS FIRST, t.titel ASC
		`, tage)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		results := []FehlbestandEntry{}
		for rows.Next() {
			var e FehlbestandEntry
			if err := rows.Scan(&e.ID, &e.BarcodeID, &e.ZustandNotiz, &e.InventurGeprueftAm,
				&e.Titel, &e.Autor, &e.CoverURL, &e.ISBN); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			results = append(results, e)
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(results)
	}
}
