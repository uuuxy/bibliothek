package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"bibliothek/apierrors"

	"github.com/jackc/pgx/v5"
)

// OrderItemRequest represents a single item to order from the cart
type OrderItemRequest struct {
	TitelID string  `json:"titel_id"`
	Menge   int     `json:"menge"`
	Preis   float64 `json:"preis"`
}

// SubmitOrderRequest represents the payload for POST /api/orders
type SubmitOrderRequest struct {
	SupplierID       string             `json:"supplier_id"`
	Items            []OrderItemRequest `json:"items"`
	GenerateBarcodes bool               `json:"generate_barcodes"`
}

// SubmitOrderHandler processes a full cart order via the OrderService and dispatches PDFs via PDFService.
func (s *Server) SubmitOrderHandler(orderSvc *OrderService, pdfSvc *PDFService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SubmitOrderRequest
		if !DecodeJSON(w, r, &req) {
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
			status := http.StatusInternalServerError
			if err.Error() == "supplier not found" {
				status = http.StatusNotFound
			} else if strings.HasPrefix(err.Error(), "invalid quantity") {
				status = http.StatusBadRequest
			}
			apierrors.SendHTTPError(w, status, err)
			return
		}

		host := os.Getenv("SMTP_HOST")
		if host == "" {
			log.Println("WARNING: SMTP_HOST environment variable not set. Email dispatch skipped. Order has been saved locally.")
			RespondJSON(w, http.StatusOK, map[string]any{
				"status":      "success",
				"message":     fmt.Sprintf("Bestellung erfasst (E-Mail-Versand an %s übersprungen - SMTP nicht konfiguriert).", res.SupplierName),
				"ordered_qty": len(res.Labels),
			})
			return
		}

		if err := pdfSvc.DispatchOrderEmail(res.SupplierName, res.SupplierEmail, res.CustomerNumber, res.SummaryItems, res.Labels, req.GenerateBarcodes); err != nil {
			RespondJSON(w, http.StatusOK, map[string]any{
				"status":      "warning",
				"message":     fmt.Sprintf("Bestellung gespeichert, aber E-Mail-Versand an %s fehlgeschlagen.", res.SupplierEmail),
				"ordered_qty": len(res.Labels),
			})
			return
		}

		RespondJSON(w, http.StatusOK, map[string]any{
			"status":      "success",
			"message":     fmt.Sprintf("Bestellung erfolgreich per E-Mail an %s gesendet.", res.SupplierName),
			"ordered_qty": len(res.Labels),
		})
	}
}

// ShipmentGroup helps structure the incoming shipments response.
type ShipmentGroup struct {
	ID           string         `json:"id"`
	SupplierName string         `json:"supplierName"`
	Date         string         `json:"date"`
	Timestamp    time.Time      `json:"-"`
	Items        []*GroupedItem `json:"items"`
}

// GroupedItem represents an item within a ShipmentGroup.
type GroupedItem struct {
	TitelID string `json:"titel_id"`
	Titel   string `json:"titel"`
	Menge   int    `json:"menge"`
}

// GetIncomingShipmentsHandler returns a list of ordered copies that are currently in transit,
// grouped by creation date and supplier.
func (s *Server) GetIncomingShipmentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		query := `
			SELECT e.titel_id, e.erstellt_am, e.zustand_notiz, t.titel
			FROM buecher_exemplare e
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE e.ist_ausleihbar = false 
			  AND (e.zustand_notiz LIKE 'Im Zulauf%' OR e.zustand_notiz = 'bestellt' OR e.zustand_notiz LIKE 'Bestellt%')
			ORDER BY e.erstellt_am DESC
		`

		rows, err := s.DB.Pool.Query(ctx, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		// Key: date_string|supplier_name
		groupsMap := make(map[string]*ShipmentGroup)

		for rows.Next() {
			var titelID, zustandNotiz, titel string
			var erstelltAm time.Time
			if err := rows.Scan(&titelID, &erstelltAm, &zustandNotiz, &titel); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}

			// Parse supplier name
			supplierName := "Unbekannter Lieferant"
			if strings.HasPrefix(zustandNotiz, "Im Zulauf - ") {
				supplierName = strings.TrimPrefix(zustandNotiz, "Im Zulauf - ")
			} else if strings.HasPrefix(zustandNotiz, "Bestellt (Lieferanten-Vorab-Barcode)") {
				supplierName = "Vorab-Barcode Bestellung"
			} else if zustandNotiz == "bestellt" {
				supplierName = "Automatische Nachbestellung"
			}

			// Group by day to make a simple date string
			dateStr := erstelltAm.Format("02.01.2006")
			groupKey := dateStr + "|" + supplierName

			group, exists := groupsMap[groupKey]
			if !exists {
				group = &ShipmentGroup{
					ID:           strconv.FormatInt(erstelltAm.UnixNano(), 10),
					SupplierName: supplierName,
					Date:         dateStr,
					Timestamp:    erstelltAm,
					Items:        []*GroupedItem{},
				}
				groupsMap[groupKey] = group
			}

			// Find if item already exists in this group
			var itemFound *GroupedItem
			for _, item := range group.Items {
				if item.Titel == titel {
					itemFound = item
					break
				}
			}

			if itemFound != nil {
				itemFound.Menge++
			} else {
				group.Items = append(group.Items, &GroupedItem{
					TitelID: titelID,
					Titel:   titel,
					Menge:   1,
				})
			}
		}

		// Sort groups by timestamp descending
		groups := make([]*ShipmentGroup, 0)
		for _, g := range groupsMap {
			groups = append(groups, g)
		}

		sort.Slice(groups, func(i, j int) bool {
			return groups[i].Timestamp.After(groups[j].Timestamp)
		})

		RespondJSON(w, http.StatusOK, groups)
	}
}

// ReceiveItemRequest represents the payload for receiving an ordered item.
type ReceiveItemRequest struct {
	TitelID string `json:"titel_id"`
	Barcode string `json:"barcode"`
}

// ReceiveItemHandler handles the reception of a single ordered item via barcode scan.
func (s *Server) ReceiveItemHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReceiveItemRequest
		if !DecodeJSON(w, r, &req) {
			return
		}

		if req.TitelID == "" || req.Barcode == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("titel_id and barcode are required"))
			return
		}

		ctx := r.Context()

		// Overwrite exactly ONE placeholder
		query := `
			UPDATE buecher_exemplare
			SET barcode_id = $1, ist_ausleihbar = true, zustand_notiz = ''
			WHERE id = (
				SELECT id 
				FROM buecher_exemplare 
				WHERE titel_id = $2 
				  AND ist_ausleihbar = false 
				  AND (zustand_notiz LIKE 'Im Zulauf%' OR zustand_notiz = 'bestellt' OR zustand_notiz LIKE 'Bestellt%')
				LIMIT 1
				FOR UPDATE SKIP LOCKED
			)
			RETURNING id
		`
		var updatedID string
		err := s.DB.Pool.QueryRow(ctx, query, req.Barcode, req.TitelID).Scan(&updatedID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("kein offenes (bestelltes) Exemplar für diesen Titel gefunden"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, map[string]string{
			"status":  "success",
			"message": "Exemplar erfolgreich freigegeben.",
		})
	}
}
