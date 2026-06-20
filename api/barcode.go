package api

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"bibliothek/apierrors"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/qr"
	"github.com/jackc/pgx/v5"
)

// OrderRequest holds the input parameters for generating a new supplier order.
type OrderRequest struct {
	TitelID string `json:"titel_id"`
	Menge   int    `json:"menge"`
}

// GenerateBarcodePNG creates a high-resolution PNG barcode image from a string.
// Supports Code39 and QR-code. Scales the output to the specified dimensions.
func GenerateBarcodePNG(content string, isQR bool, width, height int) ([]byte, error) {
	var bc barcode.Barcode
	var err error

	if isQR {
		bc, err = qr.Encode(content, qr.M, qr.Auto)
	} else {
		// Code39 is case-sensitive, capitalize content for compatibility
		bc, err = code39.Encode(strings.ToUpper(content), true, true)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode barcode: %w", err)
	}

	scaled, err := barcode.Scale(bc, width, height)
	if err != nil {
		return nil, fmt.Errorf("failed to scale barcode: %w", err)
	}

	// Convert the scaled barcode to standard 8-bit RGBA image
	// to avoid 16-bit PNG depth which gofpdf PNG parser doesn't support.
	bounds := scaled.Bounds()
	rgbaImg := image.NewRGBA(bounds)
	draw.Draw(rgbaImg, bounds, scaled, bounds.Min, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, rgbaImg); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// BarcodeHandler handles on-demand PNG barcode and QR code generation.
func (s *Server) BarcodeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		content := r.URL.Query().Get("content")
		if content == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing content parameter"))
			return
		}

		isQR := r.URL.Query().Get("qr") == "true"

		// Set default size metrics
		width := 300
		height := 100
		if isQR {
			width = 200
			height = 200
		}

		if wStr := r.URL.Query().Get("width"); wStr != "" {
			if parsed, err := strconv.Atoi(wStr); err == nil {
				width = parsed
			}
		}
		if hStr := r.URL.Query().Get("height"); hStr != "" {
			if parsed, err := strconv.Atoi(hStr); err == nil {
				height = parsed
			}
		}

		pngBytes, err := GenerateBarcodePNG(content, isQR, width, height)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
		_, _ = w.Write(pngBytes)
	}
}

// SupplierOrderHandler processes new book orders. Generates sequential B- barcodes,
// registers the new copies in the DB, and builds a print-ready PDF containing barcode sheets.
func (s *Server) SupplierOrderHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req OrderRequest
		if !DecodeJSON(w, r, &req) {
			return
		}

		if req.Menge <= 0 || req.Menge > 200 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("quantity must be between 1 and 200"))
			return
		}

		ctx := r.Context()

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

		// 2. Fetch the highest B-XXXXX barcode in the system to calculate the next sequence
		var lastBarcode string
		qLast := `
			SELECT barcode_id 
			FROM buecher_exemplare 
			WHERE barcode_id LIKE 'B-%' 
			ORDER BY barcode_id DESC 
			LIMIT 1
		`
		err = tx.QueryRow(ctx, qLast).Scan(&lastBarcode)

		startNum := 10001
		if err == nil {
			re := regexp.MustCompile(`B-(\d+)`)
			matches := re.FindStringSubmatch(lastBarcode)
			if len(matches) > 1 {
				if parsed, err := strconv.Atoi(matches[1]); err == nil {
					startNum = parsed + 1
				}
			}
		}

		// 3. Register copies in DB (marked as not borrowable until delivery)
		newBarcodes := []string{}
		qInsert := `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, zustand_notiz, ist_ausleihbar)
			VALUES ($1, $2, 'Bestellt (Lieferanten-Vorab-Barcode)', false)
		`
		for i := 0; i < req.Menge; i++ {
			barcodeID := fmt.Sprintf("B-%05d", startNum+i)
			_, err = tx.Exec(ctx, qInsert, req.TitelID, barcodeID)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			newBarcodes = append(newBarcodes, barcodeID)
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
		pdf, err := GenerateLabelsPDF("zweckform_l4760", 1, true, labelItems)
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

// NextBarcodeHandler returns the next available internal B-XXXXX barcode as JSON.
func (s *Server) NextBarcodeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var lastBarcode string
		qLast := `
			SELECT barcode_id 
			FROM buecher_exemplare 
			WHERE barcode_id LIKE 'B-%' 
			ORDER BY barcode_id DESC 
			LIMIT 1
		`
		err := s.DB.Pool.QueryRow(ctx, qLast).Scan(&lastBarcode)

		startNum := 10001
		if err == nil {
			re := regexp.MustCompile(`B-(\d+)`)
			matches := re.FindStringSubmatch(lastBarcode)
			if len(matches) > 1 {
				if parsed, err := strconv.Atoi(matches[1]); err == nil {
					startNum = parsed + 1
				}
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		nextBarcode := fmt.Sprintf("B-%05d", startNum)

		RespondJSON(w, http.StatusOK, map[string]string{
			"next_barcode": nextBarcode,
		})
	}
}

// PrintErsatzEtikettHandler generates an A6 PDF label for a single given exemplar.
func (s *Server) PrintErsatzEtikettHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing exemplar ID parameter"))
			return
		}

		ctx := r.Context()

		var label BarcodeLabelDetail
		query := `
			SELECT e.barcode_id, t.titel, coalesce(t.autor, ''), coalesce(t.isbn, '')
			FROM buecher_exemplare e
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE e.id = $1
		`
		err := s.DB.Pool.QueryRow(ctx, query, id).Scan(&label.BarcodeID, &label.Titel, &label.Autor, &label.ISBN)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("exemplar nicht gefunden"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		pdfBytes, err := GenerateSingleLabelPDFA6(label)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		filename := fmt.Sprintf("Ersatz_Etikett_%s.pdf", label.BarcodeID)
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
		_, _ = w.Write(pdfBytes)
	}
}
