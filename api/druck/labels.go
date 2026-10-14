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

// parseLabelParams liest Format, Startposition und QR-Flag aus den Query-Parametern
// (mit denselben Defaults wie bisher).
func parseLabelParams(r *http.Request) (formatId string, startPos int, isQR bool) {
	formatId = r.URL.Query().Get("format")
	if formatId == "" {
		formatId = "avery_3475" // default as before
	}

	startPos = 1
	if startParam := r.URL.Query().Get("start"); startParam != "" {
		if parsed, err := strconv.Atoi(startParam); err == nil && parsed > 0 {
			startPos = parsed
		}
	}

	isQR = r.URL.Query().Get("qr") == "true"
	return formatId, startPos, isQR
}

// LabelsHandler returns a handler that generates an A4 PDF containing 3x8 Avery labels
// for all copies of a given book title.
func (s *Tueren) LabelsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("id is required"))
			return
		}

		ctx := r.Context()
		formatId, startPos, isQR := parseLabelParams(r)

		items, err := buchetikett.LadeDesTitels(ctx, s.DB.Pool, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if len(items) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("keine exemplare für diesen titel vorhanden"))
			return
		}

		bogen, err := pdf.GenerateLabelsPDF(formatId, startPos, isQR, buchetikett.PDFEtiketten(items, buchetikett.LadeKopf(ctx, s.DB.Pool)))
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

// PrintLabelsRequest represents a request to generate a PDF label sheet.
type PrintLabelsRequest struct {
	FormatID      string                `json:"formatId"`
	StartPosition int                   `json:"startPosition"`
	IsQR          bool                  `json:"isQR"`
	Items         []buchetikett.Etikett `json:"items"`
}

// PrintLabelsHandler generates an A4 PDF containing labels dynamically.
func (s *Tueren) PrintLabelsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req PrintLabelsRequest
		if !tuer.DecodeAndValidate(w, r, &req) {
			return
		}

		if len(req.Items) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("keine exemplare angegeben"))
			return
		}

		ctx := r.Context()
		buchetikett.ErgaenzeServerfelder(ctx, s.DB.Pool, req.Items)

		bogen, err := pdf.GenerateLabelsPDF(req.FormatID, req.StartPosition, req.IsQR, buchetikett.PDFEtiketten(req.Items, buchetikett.LadeKopf(ctx, s.DB.Pool)))
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
