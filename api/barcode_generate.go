package api

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"net/http"
	"strconv"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/qr"
)

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

	// barcode.Scale verweigert Verkleinern unter die native Modulgröße
	// (lange Inhalte sprengen z. B. die üblichen 200px der Ausweiskarten).
	// Dann lieber größer ausliefern als mit 500 scheitern.
	if minW := bc.Bounds().Dx(); width < minW {
		width = minW
	}
	if minH := bc.Bounds().Dy(); height < minH {
		height = minH
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
		width, height := resolveBarcodeSize(r, isQR)

		pngBytes, err := GenerateBarcodePNG(content, isQR, width, height)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set(headerContentType, "image/png")
		w.Header().Set(headerCacheControl, "public, max-age=31536000") // Cache for 1 year
		httpresp.Write(w, pngBytes)
	}
}

// resolveBarcodeSize bestimmt die Zielgröße (Default 300×100, QR 200×200) und übernimmt
// gültige width/height-Query-Parameter.
func resolveBarcodeSize(r *http.Request, isQR bool) (width, height int) {
	width, height = 300, 100
	if isQR {
		width, height = 200, 200
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
	return width, height
}
