package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// UpdateBarcodeRequest holds the payload for updating a copy's barcode.
type UpdateBarcodeRequest struct {
	Barcode string `json:"barcode"`
}

// UpdateCopyBarcodeHandler updates the barcode of a physical book copy.
// @Summary      Update copy barcode
// @Description  Updates the barcode of a physical book copy, replacing placeholders like AUTO-.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string                true  "Book copy ID (UUID)"
// @Param        body  body      UpdateBarcodeRequest  true  "New barcode payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /buecher/exemplare/{id}/barcode [put]
func (s *Server) UpdateCopyBarcodeHandler(bookRepo repository.BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		var req UpdateBarcodeRequest
		if !DecodeJSON(w, r, &req) {
			return
		}

		if req.Barcode == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("barcode cannot be empty"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := bookRepo.UpdateCopyBarcode(ctx, id, req.Barcode); err != nil {
			if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
				apierrors.SendHTTPError(w, http.StatusConflict, errors.New("dieser Barcode wird bereits von einem anderen Exemplar verwendet"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondSuccess(w)
	}
}
