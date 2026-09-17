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

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/qr"
)

// GenerateBarcodePNG erzeugt den Strichcode eines Ausweises, Etiketts oder Briefes als PNG.
// Code 128, oder QR, wenn isQR gesetzt ist; skaliert auf die gewünschte Größe.
//
// Bis zum 17.09.2026 stand hier `code39.Encode(strings.ToUpper(content), true, true)`. Das
// mittlere `true` heißt „Prüfzeichen anhängen", und dieses Zeichen steht IN den
// Strichcode-Daten: Ein Leser gibt es als Teil der Nummer zurück, solange er es nicht selbst
// prüft und entfernt (die Voreinstellung fast aller Geräte ist „nicht prüfen").
//
// Auf dem Ausweis stand also „A-10003" und im Strichcode darüber „A-100037", auf dem
// Buchetikett „B-10001" gegen „B-100016". Der Server sucht dann eine Nummer, die es nicht
// gibt — und weil ein unbekannter Scan nur eine leere Trefferliste erzeugt, sah es an der
// Theke aus, als täte der Scanner gar nichts. Gemessen mit zwei unabhängigen Erkennern und
// am fertigen PDF des Druck-Centers (Gate: frontend/e2e/barcode-lesbar.spec.js).
//
// Code 128 statt Code 39 ohne Prüfzeichen, aus drei Gründen: Es trägt kein Prüfzeichen im
// Text, es braucht für dieselbe Nummer rund die halbe Breite (auf einem Ausweisetikett der
// Unterschied zwischen „liest sofort" und „liest manchmal"), und es kennt Klein- wie
// Großbuchstaben — der Aufdruck ist damit Zeichen für Zeichen das, was in der Datenbank
// steht. Das `ToUpper` von früher war eine Eigenheit von Code 39 und fällt mit ihm weg.
//
// GELESEN werden Code 39 und die EAN-Arten weiterhin (frontend .../barcode_detector.js).
// Das allein genügte NICHT: Die Kamera erkannte den alten Code zwar, lieferte aber die
// Nummer samt Prüfzeichen, und die gibt es in keiner Tabelle. Karten und Etiketten aus
// der Zeit davor funktionieren, seit der Scan sie als zweiten Versuch ohne Prüfzeichen
// nachschlägt (pkg/code39, internal/service/omnibox_service.go, scanEinordnen.js).
// Bis zum 17.09.2026 stand hier nur der Satz, sie sollten weiter funktionieren.
func GenerateBarcodePNG(content string, isQR bool, width, height int) ([]byte, error) {
	var bc barcode.Barcode
	var err error

	if isQR {
		bc, err = qr.Encode(content, qr.M, qr.Auto)
	} else {
		bc, err = code128.Encode(content)
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
