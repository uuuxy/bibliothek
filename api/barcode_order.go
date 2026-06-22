package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// OrderRequest holds the input parameters for generating a new supplier order.
type OrderRequest struct {
	TitelID string `json:"titel_id"`
	Menge   int    `json:"menge"`
}

// SupplierOrderHandler processes new book orders.
// It generates sequential B- barcodes using the SequenceRepository,
// registers the new copies in the DB, and builds a print-ready PDF containing barcode sheets.
func (s *Server) SupplierOrderHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req OrderRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		if req.Menge <= 0 || req.Menge > 200 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("quantity must be between 1 and 200"))
			return
		}

		ctx := r.Context()

		// Begin transaction to ensure sequence and inserts are atomic
		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		// 1. Resolve master title details
		var titel, autor string
		err = tx.QueryRow(ctx, "SELECT titel, coalesce(autor, '') FROM buecher_titel WHERE id = $1", req.TitelID).Scan(&titel, &autor)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 2. Fetch the highest B-XXXXX barcode in the system using central repo
		seqRepo := repository.NewSequenceRepository(tx)
		startNum, err := seqRepo.GetNextSequence(ctx, "buecher_exemplare", "barcode_id", "B-")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 3. Register copies in DB (marked as not borrowable until delivery)
		newBarcodes := []string{}
		var copyRows [][]any
		for i := 0; i < req.Menge; i++ {
			barcodeID := fmt.Sprintf("B-%05d", startNum+i)
			copyRows = append(copyRows, []any{req.TitelID, barcodeID, "Bestellt (Lieferanten-Vorab-Barcode)", false})
			newBarcodes = append(newBarcodes, barcodeID)
		}

		// Use pgx.CopyFromRows to resolve N+1 queries when inserting multiple barcodes
		_, err = tx.CopyFrom(
			ctx,
			pgx.Identifier{"buecher_exemplare"},
			[]string{"titel_id", "barcode_id", "zustand_notiz", "ist_ausleihbar"},
			pgx.CopyFromRows(copyRows),
		)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		var labelItems []BarcodeLabelDetail
		for _, bc := range newBarcodes {
			labelItems = append(labelItems, BarcodeLabelDetail{
				BarcodeID: bc,
				Titel:     titel,
				Autor:     autor,
				ISBN:      "", // not used for barcode sheet
			})
		}

		// 4. Generate printable PDF label sheets
		pdf, err := GenerateLabelsPDF("zweckform_l4760", 1, false, labelItems)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=bestellung_barcodes_%d.pdf", startNum))
		if err := pdf.Output(w); err != nil {
			log.Printf("Barcode: PDF streaming failure: %v", err)
		}
	}
}
