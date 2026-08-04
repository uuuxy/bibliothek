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

// queryLabelItems lädt alle Exemplare (Barcode, Titel, Autor, Anschaffungsjahr) eines Titels.
func (s *Server) queryLabelItems(ctx context.Context, id string) ([]BarcodeLabelDetail, error) {
	// erworben_am ist NOT NULL mit Vorgabe CURRENT_DATE — to_char liefert also immer
	// vier Ziffern und nie NULL.
	query := `
		SELECT e.barcode_id, t.titel, coalesce(t.autor, ''), to_char(e.erworben_am, 'YYYY')
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
		if err := rows.Scan(&item.BarcodeID, &item.Titel, &item.Autor, &item.AnschaffungsJahr); err == nil {
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

// ergaenzeAnschaffungsjahr füllt das Anschaffungsjahr aus der Datenbank nach.
//
// Der Nachdruck-Dialog schickt nur Barcode, Titel und Autor — das Jahr stünde sonst nie
// auf einem nachgedruckten Etikett. Es dem Client mitzugeben wäre der falsche Weg: Es
// ist Serverwissen, und ein Feld, das die Oberfläche mitschicken MUSS, wird irgendwann
// vergessen (dieselbe Klasse Fehler hat hier schon Buchdaten still verschluckt).
//
// Eine Abfrage für alle Barcodes, kein N+1. Unbekannte Barcodes bleiben ohne Jahr —
// beim Vorab-Druck für eine Bestellung existieren die Exemplare noch gar nicht, dort
// ist das der richtige Zustand und kein Fehler.
func (s *Server) ergaenzeAnschaffungsjahr(ctx context.Context, items []BarcodeLabelDetail) {
	barcodes := make([]string, 0, len(items))
	for i := range items {
		if items[i].BarcodeID != "" {
			barcodes = append(barcodes, items[i].BarcodeID)
		}
	}
	if len(barcodes) == 0 {
		return
	}

	rows, err := s.DB.Pool.Query(ctx,
		`SELECT barcode_id, to_char(erworben_am, 'YYYY') FROM buecher_exemplare WHERE barcode_id = ANY($1)`,
		barcodes)
	if err != nil {
		log.Printf("Etiketten: Anschaffungsjahr nicht ermittelbar, drucke ohne: %v", err)
		return
	}
	defer rows.Close()

	jahre := make(map[string]string, len(barcodes))
	for rows.Next() {
		var barcode, jahr string
		if err := rows.Scan(&barcode, &jahr); err == nil {
			jahre[barcode] = jahr
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("Etiketten: Anschaffungsjahr unvollständig gelesen: %v", err)
		return
	}

	for i := range items {
		items[i].AnschaffungsJahr = jahre[items[i].BarcodeID]
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
		s.ergaenzeAnschaffungsjahr(ctx, req.Items)

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
