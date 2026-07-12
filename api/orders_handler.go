package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
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

type SubmitOrderRequest struct {
	SupplierID string             `json:"supplier_id"`
	Items      []OrderItemRequest `json:"items"`
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

		if istPlaceholderSMTP() {
			log.Println("WARNING: SMTP_HOST environment variable not set. Email dispatch skipped. Order has been saved locally.")
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

		if err := pdfSvc.DispatchOrderEmail(res.SupplierName, res.SupplierEmail, res.CustomerNumber, res.SummaryItems, res.Labels, anyBarcodesGenerated, schule); err != nil {
			RespondJSON(w, http.StatusOK, map[string]any{
				"status":      "warning",
				"message":     fmt.Sprintf("Bestellung gespeichert, aber E-Mail-Versand an %s fehlgeschlagen.", res.SupplierEmail),
				"ordered_qty": res.TotalAllocated,
			})
			return
		}

		RespondJSON(w, http.StatusOK, map[string]any{
			"status":      "success",
			"message":     fmt.Sprintf("Bestellung erfolgreich per E-Mail an %s gesendet.", res.SupplierName),
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

// istPlaceholderSMTP erkennt eine nicht (produktiv) konfigurierte SMTP-Umgebung.
func istPlaceholderSMTP() bool {
	host := os.Getenv("SMTP_HOST")
	pass := os.Getenv("SMTP_PASS")
	return host == "" || host == "Ihr SMTP-Host" || strings.Contains(pass, "Passwort") || pass == "secret"
}

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

type OrderSearchRequest struct {
	Query string `json:"query"`
}

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
