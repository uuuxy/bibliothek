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
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

// OrderItemRequest represents a single item to order from the cart
type OrderItemRequest struct {
	TitelID          string  `json:"titel_id" validate:"required,uuid_oder_leer"`
	Menge            int     `json:"menge"`
	Preis            float64 `json:"preis"`
	GenerateBarcodes bool    `json:"generate_barcodes"`
}

// SubmitOrderRequest ist ein kompletter Warenkorb an EINEN Lieferanten. Die Bestellung
// entsteht daraus in einer Transaktion — schlägt eine Position fehl, geht keine raus.
type SubmitOrderRequest struct {
	SupplierID string             `json:"supplier_id" validate:"omitempty,uuid_oder_leer"`
	Items      []OrderItemRequest `json:"items" validate:"dive"`
	// IdempotencyKey: vom Client pro Absende-Vorgang vergeben. Ein Doppelklick schickt
	// denselben Schlüssel; die zweite Anfrage wird zum No-op (keine zweite Bestellung,
	// keine zweite Lieferanten-Mail). Optional — ohne Schlüssel läuft alles wie bisher.
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	// Mittel: der Topf dieser Bestellung — 'land' (Lernmittelfreiheit) oder
	// 'schultraeger' (Schülerbücherei), siehe pkg/mitteltopf. PFLICHT: Der Vermerk
	// muss auf der Bestellung stehen, und ein Standardwert wäre eine stille Zuordnung
	// zum falschen Topf. Ein gemischter Warenkorb schickt zwei Anfragen — eine je Topf.
	Mittel string `json:"mittel"`
}

// bestellAuftrag füllt den Auftrag an das Anlegen der Bestellung aus der Anfrage.
func bestellAuftrag(req SubmitOrderRequest) service.BestellAuftrag {
	positionen := make([]service.BestellAuftragPosition, 0, len(req.Items))
	for _, item := range req.Items {
		positionen = append(positionen, service.BestellAuftragPosition(item))
	}
	return service.BestellAuftrag{
		SupplierID:     req.SupplierID,
		Items:          positionen,
		IdempotencyKey: req.IdempotencyKey,
		Mittel:         req.Mittel,
	}
}

// SubmitOrderHandler legt die Bestellung aus dem Warenkorb an und verschickt die Bestellmail.
func (s *Server) SubmitOrderHandler(orderSvc *service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.handleSubmitOrder(w, r, orderSvc)
	}
}

// handleSubmitOrder speichert die Bestellung und verschickt danach die Mail an den Lieferanten;
// die Bestellung gilt auch, wenn der Versand ausbleibt.
func (s *Server) handleSubmitOrder(w http.ResponseWriter, r *http.Request, orderSvc *service.OrderService) {
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
	if !mitteltopf.Gueltig(req.Mittel) {
		apierrors.SendHTTPError(w, http.StatusBadRequest, mitteltopf.ErrUngueltig)
		return
	}

	ctx := r.Context()

	res, err := orderSvc.ProcessOrder(ctx, bestellAuftrag(req))
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
		s.merkeBestellmailGescheitert(ctx, res.BestellungID)
		RespondJSON(w, http.StatusOK, map[string]any{
			"status":      "success",
			"message":     fmt.Sprintf("Bestellung erfasst (E-Mail-Versand an %s übersprungen - SMTP nicht konfiguriert).", res.SupplierName),
			"ordered_qty": res.TotalAllocated,
		})
		return
	}

	mitLink, err := s.sendeBestellmail(ctx, bestellmailDatenAus(res))
	if err != nil {
		// Die Bestellung ist gespeichert, die Mail nicht raus: Der Vermerk an der Bestellung
		// bleibt, wenn diese Meldung vom Bildschirm verschwunden ist.
		s.merkeBestellmailGescheitert(ctx, res.BestellungID)
		RespondJSON(w, http.StatusOK, map[string]any{
			"status":      "warning",
			"message":     fmt.Sprintf("Bestellung gespeichert, aber E-Mail-Versand an %s fehlgeschlagen.", res.SupplierEmail),
			"ordered_qty": res.TotalAllocated,
		})
		return
	}

	status, meldung := bestellVersandMeldung(res.SupplierName, res.IstHauptlieferant && !mitLink)
	RespondJSON(w, http.StatusOK, map[string]any{
		"status":      status,
		"message":     meldung,
		"ordered_qty": res.TotalAllocated,
	})
}

// mapProcessOrderError ordnet die benannten Ablehnungen von ProcessOrder ihrem Status zu.
// Alles andere ist ein Serverfehler, dessen Text die Tür nicht ausgibt.
func mapProcessOrderError(err error) int {
	switch {
	case errors.Is(err, service.ErrLieferantUnbekannt), errors.Is(err, service.ErrTitelUnbekannt):
		return http.StatusNotFound
	case errors.Is(err, mitteltopf.ErrUngueltig), errors.Is(err, service.ErrMengeUngueltig):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// istPlaceholderSMTP ist entfallen: Die Platzhalter-Erkennung steckt jetzt in
// mailservice.SMTPKonfig.IstKonfiguriert und gilt damit für alle Versender — vorher
// kannte nur die Bestell-Abwicklung sie.

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

// dnbAusfallKopf sagt der Bestellsuche, dass die DNB nicht geantwortet hat und die Liste nur
// den eigenen Katalog trägt. Ein Kopf statt eines Felds, weil die Antwort eine Liste ist, wie
// X-Treffer-Gesamt an der Katalogsuche (opac.go).
const dnbAusfallKopf = "X-DNB-Ausfall"

// SearchOrdersHandler sucht einen Titel zum Bestellen, zuerst im eigenen Katalog und
// zusätzlich bei der DNB. Der Metadaten-Client wird EINMAL beim Registrieren der Route
// gebaut, nicht je Anfrage — er hält seinen eigenen HTTP-Client mit Zeitgrenze.
func (s *Server) SearchOrdersHandler() http.HandlerFunc {
	return s.sucheBestellung(inventur.NeuerMetadatenClient())
}

// sucheBestellung ist die Tür mit ihrem Client für die DNB als Parameter: Ein Test stellt
// die Antwort der DNB nach, wie bei isbnZuTitel.
func (s *Server) sucheBestellung(metaClient *inventur.MetadatenClient) http.HandlerFunc {
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
		results, dnbAusfall, err := service.SearchOrders(ctx, s.DB.Pool, metaClient, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if dnbAusfall {
			w.Header().Set(dnbAusfallKopf, "1")
		}

		RespondJSON(w, http.StatusOK, results)
	}
}

// BulkReceiveRequest represents the payload for bulk receiving an order.
type BulkReceiveRequest struct {
	ExemplarIDs []string `json:"exemplar_ids" validate:"omitempty,dive,required,uuid_oder_leer"`
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

		receivedItems, err := service.BulkReceiveOrder(ctx, s.DB.Pool, auditRepo, service.BulkReceiveParams{
			ExemplarIDs: req.ExemplarIDs,
			AdminID:     adminID,
			IPAddr:      getIP(r),
		})
		if err != nil {
			if errors.Is(err, service.ErrNichtsEinzubuchen) {
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
