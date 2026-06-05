package api

import (
	"context"
	"encoding/json"
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
	"bibliothek/utils"

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
	SupplierID string             `json:"supplier_id"`
	Items      []OrderItemRequest `json:"items"`
}

// SubmitOrderHandler processes a full cart order, allocates barcodes,
// generates the order PDF, and sends it to the supplier via SMTP.
func (s *Server) SubmitOrderHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SubmitOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
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

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		// 1. Fetch supplier details
		var supplierName, supplierEmail, customerNumber string
		err := s.DB.Pool.QueryRow(ctx, `
			SELECT name, email, kundennummer 
			FROM lieferanten 
			WHERE id = $1
		`, req.SupplierID).Scan(&supplierName, &supplierEmail, &customerNumber)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("supplier not found"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback(ctx)

		// 2. Fetch the highest B-XXXXX barcode in the system to calculate the next sequence
		startNum, err := utils.GetNextBarcodeSequence(ctx, tx, "buecher_exemplare", "B", false)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to get next barcode: %w", err))
			return
		}

		// 3. Register copies in DB & collect details for PDFs
		labels := make([]BarcodeLabelDetail, 0)
		orderSummaryItems := make([]OrderedItem, 0)
		currentBarcodeIndex := startNum
		isNaacher := strings.Contains(strings.ToLower(supplierName), "naacher")

		qInsert := `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, zustand_notiz, ist_ausleihbar, etikett_gedruckt, einkaufspreis)
			VALUES ($1, $2, $3, false, $4, $5)
		`

		for _, item := range req.Items {
			if item.Menge <= 0 || item.Menge > 200 {
				apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("invalid quantity %d for title %s", item.Menge, item.TitelID))
				return
			}

			// Resolve book details
			var titel, autor, isbn, verlag string
			err = tx.QueryRow(ctx, "SELECT titel, coalesce(autor, ''), coalesce(isbn, ''), coalesce(verlag, '') FROM buecher_titel WHERE id = $1", item.TitelID).Scan(&titel, &autor, &isbn, &verlag)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("book title %s not found", item.TitelID))
					return
				}
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}

			orderSummaryItems = append(orderSummaryItems, OrderedItem{
				Titel:  titel,
				Autor:  autor,
				ISBN:   isbn,
				Verlag: verlag,
				Menge:  item.Menge,
			})

			for i := 0; i < item.Menge; i++ {
				barcodeID := fmt.Sprintf("B-%05d", currentBarcodeIndex)
				statusText := fmt.Sprintf("Im Zulauf - %s", supplierName)
				_, err = tx.Exec(ctx, qInsert, item.TitelID, barcodeID, statusText, isNaacher, item.Preis)
				if err != nil {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
					return
				}
				labels = append(labels, BarcodeLabelDetail{
					BarcodeID: barcodeID,
					Titel:     titel,
					Autor:     autor,
				})
				currentBarcodeIndex++
			}
		}

		// Commit DB updates
		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 4. Generate PDFs in memory
		summaryPDF, err := GenerateOrderSummaryPDF(orderSummaryItems)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		barcodePDF, err := GenerateBarcodeSheetPDF(labels)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 5. Send SMTP Email with PDF Attachments
		emailBody := fmt.Sprintf(
			"Sehr geehrte Damen und Herren,\n\nanbei erhalten Sie unsere Buchbestellung vom %s (Kundennummer: %s) sowie den zugehörigen Barcode-Bogen zur Vorab-Beklebung der Exemplare.\n\nBestellte Titel: %d\nGesamtanzahl Exemplare: %d\n\nMit freundlichen Grüßen,\nSchulbibliothek",
			time.Now().Format("02.01.2006"),
			customerNumber,
			len(orderSummaryItems),
			len(labels),
		)

		attachments := []MailAttachment{
			{
				Name:        fmt.Sprintf("bestellanschreiben_%s.pdf", time.Now().Format("2006-01-02")),
				ContentType: "application/pdf",
				Data:        summaryPDF,
			},
		}
		if isNaacher {
			attachments = append(attachments, MailAttachment{
				Name:        fmt.Sprintf("barcode_bogen_%s.pdf", time.Now().Format("2006-01-02")),
				ContentType: "application/pdf",
				Data:        barcodePDF,
			})
		}

		mailReq := MailRequest{
			To:          supplierEmail,
			Subject:     fmt.Sprintf("Buchbestellung Schulbibliothek - %s (Kundennummer %s)", time.Now().Format("02.01.2006"), customerNumber),
			Body:        emailBody,
			Attachments: attachments,
		}

		// Fallback for missing SMTP configuration during local development
		host := os.Getenv("SMTP_HOST")
		if host == "" {
			log.Println("WARNING: SMTP_HOST environment variable not set. Email dispatch skipped. Order has been saved locally.")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":      "success",
				"message":     fmt.Sprintf("Bestellung erfasst (E-Mail-Versand an %s übersprungen - SMTP nicht konfiguriert).", supplierName),
				"ordered_qty": len(labels),
			})
			return
		}

		if err := SendEmail(mailReq); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadGateway, fmt.Errorf("email delivery failed: %w", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":      "success",
			"message":     fmt.Sprintf("Bestellung erfolgreich per E-Mail an %s gesendet.", supplierName),
			"ordered_qty": len(labels),
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

type GroupedItem struct {
	TitelID string `json:"titel_id"`
	Titel   string `json:"titel"`
	Menge   int    `json:"menge"`
}

// GetIncomingShipmentsHandler returns a list of ordered copies that are currently in transit,
// grouped by creation date and supplier.
func (s *Server) GetIncomingShipmentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

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

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(groups)
	}
}

type ReceiveItemRequest struct {
	TitelID string `json:"titel_id"`
	Barcode string `json:"barcode"`
}

func (s *Server) ReceiveItemHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReceiveItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		if req.TitelID == "" || req.Barcode == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("titel_id and barcode are required"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

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
				apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("Kein offenes (bestelltes) Exemplar für diesen Titel gefunden."))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Exemplar erfolgreich freigegeben.",
		})
	}
}
