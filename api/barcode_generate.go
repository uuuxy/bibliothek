package api

import (
	"errors"
	"net/http"
	"strconv"

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
	"bibliothek/pkg/strichcode"
)

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

		pngBytes, err := strichcode.PNG(content, isQR, width, height)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set(headerContentType, "image/png")
		// Kein eigener Cache-Control-Kopf: no-store wie jede Antwort unter /api/
		// (internal/middleware/security.go). Bis zum 30.09.2026 stand hier „public,
		// max-age=31536000". Die Adresse trägt den Inhalt, beim Ausweis also die
		// Ausweisnummer, und der Browser legte sie mit dem Bild ein Jahr lang ab. Das
		// PII-Antwort-Gate führt Ausweisnummern als Stufe 1 und prüft diese Route eigens
		// (adresseTraegtPersonendaten).
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
