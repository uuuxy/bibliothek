package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"

	"github.com/jackc/pgx/v5"
)

type InventurStartRequest struct {
	Type        string `json:"type"` // "global" or "signature"
	SignatureID *int   `json:"signature_id,omitempty"`
}

type InventurStartResponse struct {
	Scope    string `json:"scope"`
	Erwartet int    `json:"erwartet"`
}

// InventurStartHandler sets the scope for a new inventory session.
// @Summary      Start an inventory session
// @Description  Resets old inventory states and sets 'ausstehend' for the chosen scope.
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
		if !DecodeJSON(w, r, &req) {
			return
		}

		if req.Type != "global" && req.Type != "signature" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("invalid type, must be 'global' or 'signature'"))
			return
		}

		if req.Type == "signature" && req.SignatureID == nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("signature_id is required when type is 'signature'"))
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

		// 1. Reset all old states globally
		_, err = tx.Exec(ctx, "UPDATE buecher_exemplare SET inventur_status = NULL")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 2. Set 'ausstehend' for the targeted scope
		var query string
		var count int
		if req.Type == "global" {
			query = `
				WITH updated AS (
					UPDATE buecher_exemplare
					SET inventur_status = 'ausstehend'
					WHERE ist_ausgesondert = false AND ist_ausleihbar = true
					RETURNING id
				)
				SELECT count(*) FROM updated
			`
			err = tx.QueryRow(ctx, query).Scan(&count)
		} else {
			query = `
				WITH updated AS (
					UPDATE buecher_exemplare e
					SET inventur_status = 'ausstehend'
					FROM buecher_titel t
					WHERE e.titel_id = t.id 
					  AND e.ist_ausgesondert = false 
					  AND e.ist_ausleihbar = true 
					  AND t.signature_id = $1
					RETURNING e.id
				)
				SELECT count(*) FROM updated
			`
			err = tx.QueryRow(ctx, query, *req.SignatureID).Scan(&count)
		}

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

// InventurScanRequest is the payload for checking in an item during inventory.
type InventurScanRequest struct {
	BarcodeID string `json:"barcode_id"`
}

type InventurScanResponse struct {
	BarcodeID       string   `json:"barcode_id"`
	Titel           string   `json:"titel"`
	CoverURL        string   `json:"cover_url,omitempty"`
	Status          string   `json:"status"` // e.g. "erfasst"
	Warnungen       []string `json:"warnungen,omitempty"`
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
		if !DecodeJSON(w, r, &req) {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var copyID, title, coverURL string
		var isAusgesondert, isLent bool
		var inventurStatus *string

		query := `
			SELECT e.id, t.titel, coalesce(t.cover_url, ''), e.ist_ausgesondert, e.inventur_status, EXISTS (
				SELECT 1 FROM ausleihen a 
				WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
			) AS is_lent
			FROM buecher_exemplare e
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE e.barcode_id = $1
			LIMIT 1
		`
		err := s.DB.Pool.QueryRow(ctx, query, req.BarcodeID).Scan(&copyID, &title, &coverURL, &isAusgesondert, &inventurStatus, &isLent)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if isAusgesondert {
			apierrors.SendHTTPError(w, http.StatusConflict, fmt.Errorf("exemplar %s ist bereits ausgesondert", req.BarcodeID))
			return
		}

		var warnungen []string
		if isLent {
			warnungen = append(warnungen, "Buch ist laut System aktuell ausgeliehen.")
		}
		if inventurStatus == nil || *inventurStatus != "ausstehend" {
			warnungen = append(warnungen, "Buch gehört nicht zum aktuell gestarteten Inventur-Scope.")
		}

		updateQuery := `
			UPDATE buecher_exemplare
			SET inventur_status = 'erfasst',
			    inventur_geprueft_am = CURRENT_TIMESTAMP,
			    aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $1
		`
		_, err = s.DB.Pool.Exec(ctx, updateQuery, copyID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, InventurScanResponse{
			BarcodeID: req.BarcodeID,
			Titel:     title,
			CoverURL:  coverURL,
			Status:    "erfasst",
			Warnungen: warnungen,
		})
	}
}

type InventurFinishResponse struct {
	VerlorenGemeldet int `json:"verloren_gemeldet"`
}

// InventurFinishHandler concludes an inventory session.
// @Summary      Finalize inventory
// @Description  Marks all 'ausstehend' books as 'verloren' and resets inventory states.
// @Tags         inventory
// @Produce      json
// @Success      200   {object}  InventurFinishResponse
// @Failure      500   {object}  map[string]string
// @Router       /inventur/finish [post]
func (s *Server) InventurFinishHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		// Mark remaining 'ausstehend' items as lost
		query := `
			WITH updated AS (
				UPDATE buecher_exemplare
				SET ist_ausleihbar = false,
				    ist_ausgesondert = true,
				    zustand_notiz = 'Verlust bei Inventur',
				    aktualisiert_am = CURRENT_TIMESTAMP
				WHERE inventur_status = 'ausstehend'
				RETURNING titel_id
			)
			SELECT titel_id FROM updated
		`
		rows, err := tx.Query(ctx, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		
		titelIDs := make(map[string]bool)
		count := 0
		for rows.Next() {
			var tID string
			if err := rows.Scan(&tID); err == nil {
				titelIDs[tID] = true
				count++
			}
		}
		rows.Close()

		// Update total verfuegbar count for affected titles
		for tID := range titelIDs {
			_, _ = tx.Exec(ctx, "SELECT update_verfuegbar_count($1)", tID)
		}

		// Reset all inventur_status to NULL globally
		_, err = tx.Exec(ctx, "UPDATE buecher_exemplare SET inventur_status = NULL")
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
