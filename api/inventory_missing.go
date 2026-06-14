package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"bibliothek/apierrors"
)

// FehlbestandResponse encapsulates the data with pagination metadata.
type FehlbestandResponse struct {
	Data       []FehlbestandEntry `json:"data"`
	TotalCount int                `json:"total_count"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
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

		page := 1
		if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p >= 1 {
			page = p
		}

		limit := 50
		if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l >= 1 && l <= 500 {
			limit = l
		}

		offset := (page - 1) * limit

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		rows, err := s.DB.Pool.Query(ctx, `
			SELECT e.id, e.barcode_id, coalesce(e.zustand_notiz, ''), e.inventur_geprueft_am,
			       t.titel, coalesce(t.autor, ''), coalesce(t.cover_url, ''), coalesce(t.isbn, ''),
			       COUNT(*) OVER() AS total_count
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
			LIMIT $2 OFFSET $3
		`, tage, limit, offset)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		results := []FehlbestandEntry{}
		totalCount := 0

		for rows.Next() {
			var e FehlbestandEntry
			if err := rows.Scan(
				&e.ID, &e.BarcodeID, &e.ZustandNotiz, &e.InventurGeprueftAm,
				&e.Titel, &e.Autor, &e.CoverURL, &e.ISBN,
				&totalCount,
			); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			results = append(results, e)
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		response := FehlbestandResponse{
			Data:       results,
			TotalCount: totalCount,
			Page:       page,
			Limit:      limit,
		}

		RespondJSON(w, http.StatusOK, response)
	}
}
