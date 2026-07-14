package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// resolveOrderTitel lädt Titel/Autor des Stammtitels. ok=false: die Fehlerantwort
// (404 bei unbekanntem Titel, sonst 500) wurde bereits geschrieben.
func resolveOrderTitel(ctx context.Context, tx pgx.Tx, w http.ResponseWriter, titelID string) (titel, autor string, ok bool) {
	err := tx.QueryRow(ctx, "SELECT titel, coalesce(autor, '') FROM buecher_titel WHERE id = $1", titelID).Scan(&titel, &autor)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, err)
			return "", "", false
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", "", false
	}
	return titel, autor, true
}

// insertVorabBarcodes registriert menge neue Exemplare mit fortlaufenden B-Barcodes
// (nicht ausleihbar bis zur Lieferung) via CopyFrom und liefert die erzeugten Barcodes.
func insertVorabBarcodes(ctx context.Context, tx pgx.Tx, titelID string, menge, startNum int) ([]string, error) {
	newBarcodes := []string{}
	var copyRows [][]any
	for i := 0; i < menge; i++ {
		barcodeID := fmt.Sprintf("B-%05d", startNum+i)
		copyRows = append(copyRows, []any{titelID, barcodeID, "Bestellt (Lieferanten-Vorab-Barcode)", false})
		newBarcodes = append(newBarcodes, barcodeID)
	}

	// Use pgx.CopyFromRows to resolve N+1 queries when inserting multiple barcodes
	if _, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"buecher_exemplare"},
		[]string{"titel_id", "barcode_id", "zustand_notiz", "ist_ausleihbar"},
		pgx.CopyFromRows(copyRows),
	); err != nil {
		return nil, err
	}
	return newBarcodes, nil
}

// buildBarcodeLabels erzeugt die Etikett-Details für ein Barcode-Blatt.
func buildBarcodeLabels(barcodes []string, titel, autor string) []BarcodeLabelDetail {
	var labelItems []BarcodeLabelDetail
	for _, bc := range barcodes {
		labelItems = append(labelItems, BarcodeLabelDetail{
			BarcodeID: bc,
			Titel:     titel,
			Autor:     autor,
			ISBN:      "", // not used for barcode sheet
		})
	}
	return labelItems
}

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

		barcodes, titel, autor, startNum, ok := s.reserviereVorabBarcodes(ctx, w, req)
		if !ok {
			return
		}

		// Generate printable PDF label sheets
		labelItems := buildBarcodeLabels(barcodes, titel, autor)
		pdf, err := GenerateLabelsPDF("zweckform_l4760", 1, false, labelItems)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf("attachment; filename=bestellung_barcodes_%d.pdf", startNum))
		if err := pdf.Output(w); err != nil {
			log.Printf("Barcode: PDF streaming failure: %v", err)
		}
	}
}

// reserviereVorabBarcodes wickelt die gesamte Bestell-Transaktion ab (Titel prüfen,
// B-Sequenz ziehen, Exemplare per CopyFrom anlegen, committen) und liefert die erzeugten
// Barcodes samt Titeldaten. ok=false bedeutet: die Fehlerantwort wurde bereits geschrieben.
func (s *Server) reserviereVorabBarcodes(ctx context.Context, w http.ResponseWriter, req OrderRequest) (barcodes []string, titel, autor string, startNum int, ok bool) {
	// Begin transaction to ensure sequence and inserts are atomic
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, "", "", 0, false
	}
	defer db.SafeRollback(ctx, tx)

	// 1. Resolve master title details
	titel, autor, ok = resolveOrderTitel(ctx, tx, w, req.TitelID)
	if !ok {
		return nil, "", "", 0, false
	}

	// 2. Fetch the highest B-XXXXX barcode in the system using central repo
	seqRepo := repository.NewSequenceRepository(tx)
	startNum, err = seqRepo.GetNextSequence(ctx, "buecher_exemplare", "barcode_id", "B-")
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, "", "", 0, false
	}

	// 3. Register copies in DB (marked as not borrowable until delivery)
	barcodes, err = insertVorabBarcodes(ctx, tx, req.TitelID, req.Menge, startNum)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, "", "", 0, false
	}

	if err := tx.Commit(ctx); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, "", "", 0, false
	}
	return barcodes, titel, autor, startNum, true
}
