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

// BarcodeBild liefert den Parameter content als Strichcode oder, mit qr=true, als QR-Code im
// Format PNG.
func (t *Tueren) BarcodeBild() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inhalt := r.URL.Query().Get("content")
		if inhalt == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Der Inhalt des Barcodes fehlt", nil)
			return
		}

		alsQR := r.URL.Query().Get("qr") == "true"
		breite, hoehe := leseBarcodeGroesse(r, alsQR)

		bild, err := strichcode.PNG(inhalt, alsQR, breite, hoehe)
		if errors.Is(err, strichcode.ErrZuGross) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Der Barcode wäre zu groß", err)
			return
		}
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set(tuer.HeaderContentType, "image/png")
		// Kein eigener Cache-Control-Kopf: Es gilt no-store wie für jede Antwort unter /api/
		// (internal/middleware/sicherheitskopfzeilen.go). Die Adresse trägt den Inhalt, beim
		// Ausweis die Ausweisnummer, und der Browser legte sie mit dem Bild ab; die Prüfung der
		// Antworten auf Personendaten führt die Route deshalb eigens (adresseTraegtPersonendaten).
		httpresp.Schreibe(w, bild)
	}
}

// leseBarcodeGroesse liest Breite und Höhe aus den Parametern width und height; ohne gültige
// Angabe gilt 300×100, für einen QR-Code 200×200.
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
