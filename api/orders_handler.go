package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/service"
	"bibliothek/inventur"
	"bibliothek/pdf"
	"bibliothek/repository"
)

// OrderItemRequest represents a single item to order from the cart
type OrderItemRequest struct {
	TitelID          string  `json:"titel_id"`
	Menge            int     `json:"menge"`
	Preis            float64 `json:"preis"`
	GenerateBarcodes bool    `json:"generate_barcodes"`
}

// SubmitOrderRequest ist ein kompletter Warenkorb an EINEN Lieferanten. Die Bestellung
// entsteht daraus in einer Transaktion — schlägt eine Position fehl, geht keine raus.
type SubmitOrderRequest struct {
	SupplierID string             `json:"supplier_id"`
	Items      []OrderItemRequest `json:"items"`
	// IdempotencyKey: vom Client pro Absende-Vorgang vergeben. Ein Doppelklick schickt
	// denselben Schlüssel; die zweite Anfrage wird zum No-op (keine zweite Bestellung,
	// keine zweite Lieferanten-Mail). Optional — ohne Schlüssel läuft alles wie bisher.
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// SubmitOrderHandler processes a full cart order via the OrderService and dispatches PDFs via PDFService.
func (s *Server) SubmitOrderHandler(orderSvc *OrderService, pdfSvc *PDFService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SubmitOrderRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		if req.SupplierID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("supplier_id is required"))
			return
		}
		if len(req.Items) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("order cart cannot be empty"))
			return
		}

		ctx := r.Context()

		res, err := orderSvc.ProcessOrder(ctx, req)
		if err != nil {
			apierrors.SendHTTPError(w, mapProcessOrderError(err), err)
			return
		}

		// Doppelklick: dieselbe Bestellung lief schon durch — KEINE zweite Mail, keine
		// zweite Bestellung. Die erste Anfrage hat Mail und Etiketten bereits erledigt.
		if res.BereitsVorhanden {
			RespondJSON(w, http.StatusOK, map[string]any{
				"status":      "success",
				"message":     fmt.Sprintf("Bestellung an %s war bereits erfasst (Doppelklick) — es wurde keine zweite Bestellung ausgelöst.", res.SupplierName),
				"ordered_qty": res.TotalAllocated,
			})
			return
		}

		if !smtpKonfiguriert() {
			log.Println("WARNUNG: Kein (echter) SMTP-Server hinterlegt. E-Mail-Versand übersprungen, die Bestellung ist gespeichert.")
			RespondJSON(w, http.StatusOK, map[string]any{
				"status":      "success",
				"message":     fmt.Sprintf("Bestellung erfasst (E-Mail-Versand an %s übersprungen - SMTP nicht konfiguriert).", res.SupplierName),
				"ordered_qty": res.TotalAllocated,
			})
			return
		}

		// Sum up how many items have generate_barcodes to pass to pdfSvc
		anyBarcodesGenerated := hatVorabBarcodes(req.Items)

		settingsRepo := repository.NewSystemSettingsRepository(s.DB.Pool)
		settings, _ := settingsRepo.GetSettings(ctx) //nolint:errcheck
		schule := pdf.SchuleInfo{
			Name:    settings.SchuleName,
			Strasse: settings.SchuleStrasse,
			PLZ:     settings.SchulePLZ,
			Ort:     settings.SchuleOrt,
		}

		betreff, textBody := s.loadBestellTemplate(ctx)
		// Ohne hinterlegte öffentliche Adresse bleibt der Link leer: Die Bestellung geht
		// dann wie bisher raus, nur ohne Bestätigungsschritt. Ein Link auf den internen
		// Servernamen wäre beim Lieferanten wertlos und sähe trotzdem echt aus.
		link := ""
		if settings.OeffentlicheAdresse != nil {
			link = bestaetigungsLink(*settings.OeffentlicheAdresse, res.BestaetigungsToken)
		}
		subject, body := resolveBestellMail(betreff, textBody, res.CustomerNumber, len(res.SummaryItems), len(res.Labels), link)

		if err := pdfSvc.DispatchOrderEmail(BestellMail{
			Empfaenger:           res.SupplierEmail,
			Betreff:              subject,
			Text:                 body,
			Positionen:           res.SummaryItems,
			Etiketten:            res.Labels,
			MitVorabBarcodes:     anyBarcodesGenerated,
			IstHauptlieferant:    res.IstHauptlieferant,
			MitBestaetigungsLink: link != "",
			Schule:               schule,
			Eigentumsvermerk:     settings.EtikettEigentumsvermerk,
		}); err != nil {
			RespondJSON(w, http.StatusOK, map[string]any{
				"status":      "warning",
				"message":     fmt.Sprintf("Bestellung gespeichert, aber E-Mail-Versand an %s fehlgeschlagen.", res.SupplierEmail),
				"ordered_qty": res.TotalAllocated,
			})
			return
		}

		status, meldung := bestellVersandMeldung(res.SupplierName, res.IstHauptlieferant && link == "")
		RespondJSON(w, http.StatusOK, map[string]any{
			"status":      status,
			"message":     meldung,
			"ordered_qty": res.TotalAllocated,
		})
	}
}

// mapProcessOrderError bildet die (textbasierten) Fehler von ProcessOrder auf HTTP-Status ab.
func mapProcessOrderError(err error) int {
	switch {
	case err.Error() == "supplier not found":
		return http.StatusNotFound
	case strings.HasPrefix(err.Error(), "invalid quantity"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// istPlaceholderSMTP ist entfallen: Die Platzhalter-Erkennung steckt jetzt in
// mailservice.SMTPKonfig.IstKonfiguriert und gilt damit für alle Versender — vorher
// kannte nur die Bestell-Abwicklung sie.

// hatVorabBarcodes prüft, ob mindestens eine Bestellposition Vorab-Barcodes generiert.
func hatVorabBarcodes(items []OrderItemRequest) bool {
	for _, item := range items {
		if item.GenerateBarcodes {
			return true
		}
	}
	return false
}

// GetIncomingShipmentsHandler returns a list of ordered copies that are currently in transit,
// grouped by creation date and supplier.
func (s *Server) GetIncomingShipmentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		groups, err := service.GetIncomingShipments(ctx, s.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, groups)
	}
}

// OrderSearchItem ist ein Suchtreffer der Bestellsuche. Source unterscheidet die
// Herkunft: "local" kommt aus dem eigenen Katalog und trägt dann auch CurrentStock,
// "dnb" ist ein Fremdtreffer ohne Bestand. IsDuplicate warnt, dass der Titel bereits im
// Haus steht — nachbestellen ist erlaubt, aber es soll niemand versehentlich tun.
type OrderSearchItem struct {
	ID           string `json:"id,omitempty"`
	Titel        string `json:"titel"`
	Autor        string `json:"autor"`
	ISBN         string `json:"isbn"`
	Verlag       string `json:"verlag,omitempty"`
	CoverURL     string `json:"cover_url,omitempty"`
	Source       string `json:"source"` // "local" or "dnb"
	CurrentStock int    `json:"current_stock,omitempty"`
	IsDuplicate  bool   `json:"is_duplicate,omitempty"`
}

// OrderSearchRequest trägt den Suchbegriff der Bestellsuche — ISBN oder Freitext.
type OrderSearchRequest struct {
	Query string `json:"query"`
}

// SearchOrdersHandler sucht einen Titel zum Bestellen, zuerst im eigenen Katalog und
// zusätzlich bei der DNB. Der Metadaten-Client wird EINMAL beim Registrieren der Route
// gebaut, nicht je Anfrage — er hält seinen eigenen HTTP-Client mit Zeitgrenze.
func (s *Server) SearchOrdersHandler() http.HandlerFunc {
	metaClient := inventur.NeuerMetadatenClient()
	return func(w http.ResponseWriter, r *http.Request) {
		var req OrderSearchRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		query := strings.TrimSpace(req.Query)
		if query == "" {
			RespondJSON(w, http.StatusOK, []service.OrderSearchItem{})
			return
		}

		ctx := r.Context()
		results, err := service.SearchOrders(ctx, s.DB.Pool, metaClient, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, results)
	}
}

// BulkReceiveRequest represents the payload for bulk receiving an order.
type BulkReceiveRequest struct {
	ExemplarIDs []string `json:"exemplar_ids"`
}

// BulkReceiveOrderHandler marks all pre-allocated items for a specific order group as received.
func (s *Server) BulkReceiveOrderHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BulkReceiveRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		if len(req.ExemplarIDs) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("no exemplar_ids provided"))
			return
		}

		ctx := r.Context()
		var adminID string
		if claims, ok := auth.GetClaims(ctx); ok {
			adminID = claims.UserID
		}

		auditRepo := repository.NewAuditRepository(s.DB.Pool)

		receivedItems, err := service.BulkReceiveOrder(ctx, s.DB.Pool, auditRepo, req.ExemplarIDs, adminID, getIP(r))
		if err != nil {
			if err.Error() == "keine zu aktualisierenden Exemplare gefunden (bereits freigegeben?)" {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, map[string]any{
			"status":         "success",
			"received_count": len(receivedItems),
			"received_items": receivedItems,
		})
	}
}
