package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/buchetikett"
	"bibliothek/pdf"
	"bibliothek/repository"
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

// queryLabelItems lädt alle Exemplare (Barcode, Titel, Autor, Anschaffungsjahr, Signatur) eines Titels.
func (s *Server) queryLabelItems(ctx context.Context, id string) ([]buchetikett.BarcodeLabelDetail, error) {
	zeilen, err := repository.EtikettDatenDesTitels(ctx, s.DB.Pool, id)
	if err != nil {
		return nil, err
	}
	return buchetikett.EtikettenAus(zeilen), nil
}

// LabelsHandler returns a handler that generates an A4 PDF containing 3x8 Avery labels
// for all copies of a given book title.
func (s *Server) LabelsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("id is required"))
			return
		}

		ctx := r.Context()
		formatId, startPos, isQR := parseLabelParams(r)

		items, err := s.queryLabelItems(ctx, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if len(items) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("keine exemplare für diesen titel vorhanden"))
			return
		}

		bogen, err := pdf.GenerateLabelsPDF(formatId, startPos, isQR, buchetikett.BuchEtiketten(items, buchetikett.LadeEtikettKopf(ctx, s.DB.Pool)))
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

// ergaenzeServerfelder füllt die Etikettenfelder nach, die NICHT aus der Anfrage
// stammen: Anschaffungsjahr und Signatur.
//
// Der Nachdruck-Dialog schickt nur Barcode, Titel und Autor — beides stünde sonst nie
// auf einem nachgedruckten Etikett. Es dem Client mitzugeben wäre der falsche Weg: Es
// ist Serverwissen, und ein Feld, das die Oberfläche mitschicken MUSS, wird irgendwann
// vergessen (dieselbe Klasse Fehler hat hier schon Buchdaten still verschluckt).
//
// Genau das ist der Signatur passiert: Diese Nachfüllung gab es zuerst nur fürs Jahr,
// also trug dasselbe Buch je nach Druckweg einen anderen Aufkleber — über
// GET /api/buecher/titel/{id}/etiketten mit Signatur, über POST /api/print/labels ohne.
// Wer hier ein Feld ergänzt, ergänzt es auch in queryLabelItems und
// ladeBestellEtiketten; TestEtikettenWegeDruckenDasselbe hält die Wege am fertigen PDF
// zusammen.
//
// Eine Abfrage für alle Barcodes, kein N+1. Unbekannte Barcodes bleiben unangetastet —
// beim Vorab-Druck für eine Bestellung existieren die Exemplare noch gar nicht, dort
// ist das der richtige Zustand und kein Fehler.
func (s *Server) ergaenzeServerfelder(ctx context.Context, items []buchetikett.BarcodeLabelDetail) {
	barcodes := make([]string, 0, len(items))
	for i := range items {
		if items[i].BarcodeID != "" {
			barcodes = append(barcodes, items[i].BarcodeID)
		}
	}
	if len(barcodes) == 0 {
		return
	}

	bekannt, err := repository.EtikettServerfelderZuBarcodes(ctx, s.DB.Pool, barcodes)
	if err != nil {
		if errors.Is(err, repository.ErrZeileUnlesbar) {
			log.Printf("Etiketten: Serverfelder unvollständig gelesen: %v", err)
			return
		}
		log.Printf("Etiketten: Serverfelder nicht ermittelbar, drucke ohne: %v", err)
		return
	}

	for i := range items {
		if felder, ok := bekannt[items[i].BarcodeID]; ok {
			items[i].AnschaffungsJahr = felder.Jahr
			items[i].Signatur = felder.Signatur
			items[i].Topf = felder.Topf
		}
	}
}

// PrintLabelsRequest represents a request to generate a PDF label sheet.
type PrintLabelsRequest struct {
	FormatID      string                           `json:"formatId"`
	StartPosition int                              `json:"startPosition"`
	IsQR          bool                             `json:"isQR"`
	Items         []buchetikett.BarcodeLabelDetail `json:"items"`
}

// PrintLabelsHandler generates an A4 PDF containing labels dynamically.
func (s *Server) PrintLabelsHandler() http.HandlerFunc {
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
		s.ergaenzeServerfelder(ctx, req.Items)

		bogen, err := pdf.GenerateLabelsPDF(req.FormatID, req.StartPosition, req.IsQR, buchetikett.BuchEtiketten(req.Items, buchetikett.LadeEtikettKopf(ctx, s.DB.Pool)))
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
