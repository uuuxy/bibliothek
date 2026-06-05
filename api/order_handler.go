package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/utils"
)

// SendOrderMailRequest specifies the recipient email payload.
type SendOrderMailRequest struct {
	Email string `json:"email"`
}

// SendOrderMailHandler handles the entire automated one-click ordering pipeline.
func (s *Server) SendOrderMailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SendOrderMailRequest
		// Optional custom email parsing (don't error if not sent, fallback will be used)
		_ = json.NewDecoder(r.Body).Decode(&req)

		toEmail := req.Email
		if toEmail == "" {
			toEmail = os.Getenv("SMTP_TO")
		}
		if toEmail == "" {
			toEmail = os.Getenv("SUPPLIER_EMAIL")
		}
		if toEmail == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing recipient email in payload or environment (SMTP_TO/SUPPLIER_EMAIL)"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback(ctx)

		// 1. Fetch titles below reorder point
		reorderQuery := `
			SELECT t.id, t.titel, coalesce(t.autor, ''), coalesce(t.isbn, ''), coalesce(t.verlag, ''), t.meldebestand,
				(SELECT COUNT(*) FROM buecher_exemplare e 
				 WHERE e.titel_id = t.id AND e.ist_ausleihbar = true 
				   AND NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
				) AS verfuegbar
			FROM buecher_titel t
			WHERE (
				SELECT COUNT(*) FROM buecher_exemplare e 
				WHERE e.titel_id = t.id AND e.ist_ausleihbar = true 
				  AND NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
			) < t.meldebestand
		`
		rows, err := tx.Query(ctx, reorderQuery)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		type ReorderItem struct {
			ID           string
			Titel        string
			Autor        string
			ISBN         string
			Verlag       string
			Meldebestand int
			Verfuegbar   int
			OrderQty     int
		}

		var itemsToOrder []ReorderItem
		for rows.Next() {
			var item ReorderItem
			if err := rows.Scan(&item.ID, &item.Titel, &item.Autor, &item.ISBN, &item.Verlag, &item.Meldebestand, &item.Verfuegbar); err == nil {
				item.OrderQty = item.Meldebestand - item.Verfuegbar
				if item.OrderQty > 0 {
					itemsToOrder = append(itemsToOrder, item)
				}
			}
		}
		rows.Close()

		// If no items need ordering, respond immediately
		if len(itemsToOrder) == 0 {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":  "success",
				"message": "Alle Bestände ausreichend. Keine Bestellung notwendig.",
				"ordered": 0,
			})
			return
		}

		// 2. Fetch the highest B-XXXXX barcode in the system
		startNum, err := utils.GetNextBarcodeSequence(ctx, tx, "buecher_exemplare", "B", false)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("failed to get next barcode: %w", err))
			return
		}

		// 3. Register copies in DB & collect barcode label details
		labels := make([]BarcodeLabelDetail, 0)
		orderSummaryItems := make([]OrderedItem, 0)
		currentBarcodeIndex := startNum
		isNaacher := strings.Contains(strings.ToLower(toEmail), "naacher")

		qInsert := `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, zustand_notiz, ist_ausleihbar, etikett_gedruckt)
			VALUES ($1, $2, 'bestellt', false, $3)
		`

		for _, item := range itemsToOrder {
			orderSummaryItems = append(orderSummaryItems, OrderedItem{
				Titel:  item.Titel,
				Autor:  item.Autor,
				ISBN:   item.ISBN,
				Verlag: item.Verlag,
				Menge:  item.OrderQty,
			})

			for i := 0; i < item.OrderQty; i++ {
				barcodeID := fmt.Sprintf("B-%05d", currentBarcodeIndex)
				_, err = tx.Exec(ctx, qInsert, item.ID, barcodeID, isNaacher)
				if err != nil {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
					return
				}
				labels = append(labels, BarcodeLabelDetail{
					BarcodeID: barcodeID,
					Titel:     item.Titel,
					Autor:     item.Autor,
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
			"Sehr geehrte Damen und Herren,\n\nanbei erhalten Sie unsere Buchbestellung vom %s sowie den zugehörigen Barcode-Bogen zur Vorab-Beklebung der Exemplare.\n\nBestellte Titel: %d\nGesamtanzahl Exemplare: %d\n\nMit freundlichen Grüßen,\nSchulbibliothek",
			time.Now().Format("02.01.2006"),
			len(orderSummaryItems),
			len(labels),
		)

		attachments := []MailAttachment{
			{
				Name:        fmt.Sprintf("bestelluebersicht_%s.pdf", time.Now().Format("2006-01-02")),
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
			To:          toEmail,
			Subject:     fmt.Sprintf("Buchbestellung Schulbibliothek - %s", time.Now().Format("02.01.2006")),
			Body:        emailBody,
			Attachments: attachments,
		}

		if err := SendEmail(mailReq); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadGateway, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":      "success",
			"message":     fmt.Sprintf("Bestellung erfolgreich an %s gesendet.", toEmail),
			"ordered_qty": len(labels),
			"titles_qty":  len(orderSummaryItems),
		})
	}
}

// ReleaseOrdersHandler releases all pending ordered copies, activating them in inventory.
func (s *Server) ReleaseOrdersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		query := `
			UPDATE buecher_exemplare e
			SET ist_ausleihbar = true,
			    zustand_notiz = '',
			    aktualisiert_am = CURRENT_TIMESTAMP
			FROM buecher_titel t
			WHERE e.titel_id = t.id
			  AND e.ist_ausleihbar = false 
			  AND (e.zustand_notiz = 'bestellt' 
			       OR e.zustand_notiz = 'Bestellt (Lieferanten-Vorab-Barcode)'
			       OR e.zustand_notiz = 'Im Zulauf'
			       OR e.zustand_notiz LIKE 'Im Zulauf%')
			RETURNING e.barcode_id, t.titel, coalesce(t.autor, '') AS autor, e.etikett_gedruckt
		`
		rows, err := s.DB.Pool.Query(ctx, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		type ReleasedItem struct {
			BarcodeID       string `json:"barcode_id"`
			Titel           string `json:"titel"`
			Autor           string `json:"autor"`
			EtikettGedruckt bool   `json:"etikett_gedruckt"`
		}

		items := make([]ReleasedItem, 0)
		for rows.Next() {
			var item ReleasedItem
			if err := rows.Scan(&item.BarcodeID, &item.Titel, &item.Autor, &item.EtikettGedruckt); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			items = append(items, item)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":         "success",
			"message":        fmt.Sprintf("Lieferung vollständig freigegeben. %d Exemplare im Bestand aktiv.", len(items)),
			"released_count": len(items),
			"released_items": items,
		})
	}
}
