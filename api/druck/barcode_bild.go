package druck

import (
	"errors"
	"net/http"
	"strconv"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
	"bibliothek/pkg/strichcode"
)

// BarcodeBild handles on-demand PNG barcode and QR code generation.
func (t *Tueren) BarcodeBild() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inhalt := r.URL.Query().Get("content")
		if inhalt == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing content parameter"))
			return
		}

		alsQR := r.URL.Query().Get("qr") == "true"
		breite, hoehe := leseBarcodeGroesse(r, alsQR)

		bild, err := strichcode.PNG(inhalt, alsQR, breite, hoehe)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set(tuer.HeaderContentType, "image/png")
		// Kein eigener Cache-Control-Kopf: no-store wie jede Antwort unter /api/
		// (internal/middleware/sicherheitskopfzeilen.go). Bis zum 30.09.2026 stand hier „public,
		// max-age=31536000". Die Adresse trägt den Inhalt, beim Ausweis also die
		// Ausweisnummer, und der Browser legte sie mit dem Bild ein Jahr lang ab. Das
		// PII-Antwort-Gate führt Ausweisnummern als Stufe 1 und prüft diese Route eigens
		// (adresseTraegtPersonendaten).
		httpresp.Schreibe(w, bild)
	}
}

// leseBarcodeGroesse bestimmt die Zielgröße (Default 300×100, QR 200×200) und übernimmt
// gültige width/height-Query-Parameter.
func leseBarcodeGroesse(r *http.Request, alsQR bool) (breite, hoehe int) {
	breite, hoehe = 300, 100
	if alsQR {
		breite, hoehe = 200, 200
	}
	if breiteText := r.URL.Query().Get("width"); breiteText != "" {
		if zahl, err := strconv.Atoi(breiteText); err == nil {
			breite = zahl
		}
	}
	if hoeheText := r.URL.Query().Get("height"); hoeheText != "" {
		if zahl, err := strconv.Atoi(hoeheText); err == nil {
			hoehe = zahl
		}
	}
	return breite, hoehe
}
