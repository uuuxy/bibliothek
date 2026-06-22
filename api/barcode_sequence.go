package api

import (
	"fmt"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// NextBarcodeHandler returns the next available internal B-XXXXX barcode as JSON.
// This leverages the centralized SequenceRepository to safely find the highest
// sequence number and increment it.
func (s *Server) NextBarcodeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		seqRepo := repository.NewSequenceRepository(s.DB.Pool)
		
		// Determine the next B- barcode number dynamically from the DB
		startNum, err := seqRepo.GetNextSequence(ctx, "buecher_exemplare", "barcode_id", "B-")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		nextBarcode := fmt.Sprintf("B-%05d", startNum)

		RespondJSON(w, http.StatusOK, map[string]string{
			"next_barcode": nextBarcode,
		})
	}
}
