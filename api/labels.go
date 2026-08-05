package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"bibliothek/apierrors"
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
func (s *Server) queryLabelItems(ctx context.Context, id string) ([]BarcodeLabelDetail, error) {
	// erworben_am ist NOT NULL mit Vorgabe CURRENT_DATE — to_char liefert also immer
	// vier Ziffern und nie NULL.
	query := `
		SELECT e.barcode_id, t.titel, coalesce(t.autor, ''), to_char(e.erworben_am, 'YYYY'), coalesce(t.signatur, '')
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE e.titel_id = $1
		ORDER BY e.barcode_id
	`
	rows, err := s.DB.Pool.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("fehler beim laden der exemplare: %w", err)
	}
	defer rows.Close()

	var items []BarcodeLabelDetail
	for rows.Next() {
		var item BarcodeLabelDetail
		if err := rows.Scan(&item.BarcodeID, &item.Titel, &item.Autor, &item.AnschaffungsJahr, &item.Signatur); err == nil {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("datenbankfehler: %w", err)
	}
	return items, nil
}

// etikettKopf lädt Schulname und Eigentumsvermerk aus den Systemeinstellungen.
//
// Fehlt der Schulname, bleibt die Zeile LEER statt auf einen erfundenen Wert
// zurückzufallen: Ein Etikett, das die falsche Schule nennt, führt ein gefundenes Buch
// in die Irre. Der Eigentumsvermerk hat dagegen eine sinnvolle Vorgabe, weil er für
// alle Bücher desselben Trägers gleich lautet.
func (s *Server) etikettKopf(ctx context.Context) EtikettKopf {
	kopf := EtikettKopf{Eigentumsvermerk: repository.StandardEigentumsvermerk}
	settings, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
	if err != nil {
		log.Printf("Etiketten: Einstellungen nicht lesbar, drucke ohne Schulnamen: %v", err)
		return kopf
	}
	kopf.Schulname = settings.SchuleName
	if settings.EtikettEigentumsvermerk != "" {
		kopf.Eigentumsvermerk = settings.EtikettEigentumsvermerk
	}
	return kopf
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

		pdf, err := GenerateLabelsPDF(formatId, startPos, isQR, items, s.etikettKopf(ctx))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler bei der pdf generierung: %w", err))
			return
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf("inline; filename=\"etiketten_%s.pdf\"", id))

		if err := pdf.Output(w); err != nil {
			log.Printf("Fehler beim Senden des PDFs: %v", err)
		}
	}
}

// serverEtikettFelder sind die Etikettenangaben, die der Server selbst kennt und
// deshalb niemals vom Client entgegennimmt.
type serverEtikettFelder struct {
	jahr     string
	signatur string
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
func (s *Server) ergaenzeServerfelder(ctx context.Context, items []BarcodeLabelDetail) {
	barcodes := make([]string, 0, len(items))
	for i := range items {
		if items[i].BarcodeID != "" {
			barcodes = append(barcodes, items[i].BarcodeID)
		}
	}
	if len(barcodes) == 0 {
		return
	}

	rows, err := s.DB.Pool.Query(ctx, `
		SELECT e.barcode_id, to_char(e.erworben_am, 'YYYY'), coalesce(t.signatur, '')
		FROM buecher_exemplare e
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE e.barcode_id = ANY($1)
	`, barcodes)
	if err != nil {
		log.Printf("Etiketten: Serverfelder nicht ermittelbar, drucke ohne: %v", err)
		return
	}
	defer rows.Close()

	bekannt := make(map[string]serverEtikettFelder, len(barcodes))
	for rows.Next() {
		var barcode string
		var felder serverEtikettFelder
		if err := rows.Scan(&barcode, &felder.jahr, &felder.signatur); err == nil {
			bekannt[barcode] = felder
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("Etiketten: Serverfelder unvollständig gelesen: %v", err)
		return
	}

	for i := range items {
		if felder, ok := bekannt[items[i].BarcodeID]; ok {
			items[i].AnschaffungsJahr = felder.jahr
			items[i].Signatur = felder.signatur
		}
	}
}

// PrintLabelsRequest represents a request to generate a PDF label sheet.
type PrintLabelsRequest struct {
	FormatID      string               `json:"formatId"`
	StartPosition int                  `json:"startPosition"`
	IsQR          bool                 `json:"isQR"`
	Items         []BarcodeLabelDetail `json:"items"`
}

// PrintLabelsHandler generates an A4 PDF containing labels dynamically.
func (s *Server) PrintLabelsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req PrintLabelsRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		if len(req.Items) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("keine exemplare angegeben"))
			return
		}

		ctx := r.Context()
		s.ergaenzeServerfelder(ctx, req.Items)

		pdf, err := GenerateLabelsPDF(req.FormatID, req.StartPosition, req.IsQR, req.Items, s.etikettKopf(ctx))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler bei der pdf generierung: %w", err))
			return
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, "inline; filename=\"etiketten_custom.pdf\"")

		if err := pdf.Output(w); err != nil {
			log.Printf("Fehler beim Senden des PDFs: %v", err)
		}
	}
}
