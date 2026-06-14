package api

import (
	"context"
	"net/http"
	"time"

	"bibliothek/apierrors"
)

// FinalizeInventoryRequest is the payload for concluding an inventory session.
type FinalizeInventoryRequest struct {
	Tage int `json:"tage"`
}

// FinalizeInventoryResponse yields the number of items marked as lost.
type FinalizeInventoryResponse struct {
	VerlorenGemeldet int `json:"verloren_gemeldet"`
}

// FinalizeInventoryHandler marks all missing books as lost.
// @Summary      Finalize inventory and book losses
// @Description  Marks all expected-but-missing books (based on the same logic as Fehlbestand) as 'verloren', rendering them non-borrowable and ausgesondert.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      FinalizeInventoryRequest   true  "Finalize configuration"
// @Success      200   {object}  FinalizeInventoryResponse
// @Failure      500   {object}  map[string]string
// @Router       /inventur/finalize [post]
func (s *Server) FinalizeInventoryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req FinalizeInventoryRequest
		if !DecodeJSON(w, r, &req) {
			return
		}

		tage := req.Tage
		if tage < 1 || tage > 3650 {
			tage = 30 // default fallback
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		// Begin transaction for safety, though a single UPDATE also works.
		// We use a single UPDATE query to mark them as lost.
		// This uses exactly the same WHERE logic as GetFehlbestandHandler.
		query := `
			WITH updated AS (
				UPDATE buecher_exemplare e
				SET ist_ausleihbar = false,
				    ist_ausgesondert = true,
				    zustand_notiz = 'Verlust bei Inventur',
				    aktualisiert_am = CURRENT_TIMESTAMP
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
				RETURNING e.id
			)
			SELECT count(*) FROM updated;
		`

		var count int
		err := s.DB.Pool.QueryRow(ctx, query, tage).Scan(&count)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, FinalizeInventoryResponse{
			VerlorenGemeldet: count,
		})
	}
}
