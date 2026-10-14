package druck

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/buchetikett"
	"bibliothek/pdf"
)

// leseEtikettParameter liest Format, Startposition und QR-Flag aus den Query-Parametern
// (mit denselben Defaults wie bisher).
func leseEtikettParameter(r *http.Request) (formatID string, start int, alsQR bool) {
	formatID = r.URL.Query().Get("format")
	if formatID == "" {
		formatID = "avery_3475" // default as before
	}

	start = 1
	if startText := r.URL.Query().Get("start"); startText != "" {
		if zahl, err := strconv.Atoi(startText); err == nil && zahl > 0 {
			start = zahl
		}
	}

	alsQR = r.URL.Query().Get("qr") == "true"
	return formatID, start, alsQR
}

// TitelEtiketten returns a handler that generates an A4 PDF containing 3x8 Avery labels
// for all copies of a given book title.
func (t *Tueren) TitelEtiketten() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("id is required"))
			return
		}

		ctx := r.Context()
		formatID, start, alsQR := leseEtikettParameter(r)

		etiketten, err := buchetikett.LadeDesTitels(ctx, t.DB.Pool, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if len(etiketten) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("keine exemplare für diesen titel vorhanden"))
			return
		}

		bogen, err := pdf.GenerateLabelsPDF(formatID, start, alsQR, buchetikett.PDFEtiketten(etiketten, buchetikett.LadeKopf(ctx, t.DB.Pool)))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler bei der pdf generierung: %w", err))
			return
		}

		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, fmt.Sprintf("inline; filename=\"etiketten_%s.pdf\"", id))

		if err := bogen.Output(w); err != nil {
			log.Printf("Fehler beim Senden des PDFs: %v", err)
		}
	}
}

// etikettenbogenAnfrage represents a request to generate a PDF label sheet.
type etikettenbogenAnfrage struct {
	FormatID      string                `json:"formatId"`
	Startposition int                   `json:"startPosition"`
	AlsQR         bool                  `json:"isQR"`
	Etiketten     []buchetikett.Etikett `json:"items"`
}

// Etikettenbogen generates an A4 PDF containing labels dynamically.
func (t *Tueren) Etikettenbogen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req etikettenbogenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}

		if len(req.Etiketten) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("keine exemplare angegeben"))
			return
		}

		ctx := r.Context()
		buchetikett.ErgaenzeServerfelder(ctx, t.DB.Pool, req.Etiketten)

		bogen, err := pdf.GenerateLabelsPDF(req.FormatID, req.Startposition, req.AlsQR, buchetikett.PDFEtiketten(req.Etiketten, buchetikett.LadeKopf(ctx, t.DB.Pool)))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler bei der pdf generierung: %w", err))
			return
		}

		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, "inline; filename=\"etiketten_custom.pdf\"")

		if err := bogen.Output(w); err != nil {
			log.Printf("Fehler beim Senden des PDFs: %v", err)
		}
	}
}
