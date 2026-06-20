package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bibliothek/apierrors"

	"github.com/jackc/pgx/v5"
)

type SendOrderMailRequest struct {
	Email string `json:"email"`
}

func (s *Server) SendOrderMailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SendOrderMailRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

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

		ctx := r.Context()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

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

		if len(itemsToOrder) == 0 {
			RespondJSON(w, http.StatusOK, map[string]any{
				"status":  "success",
				"message": "Alle Bestände ausreichend. Keine Bestellung notwendig.",
				"ordered": 0,
			})
			return
		}

		var lastBarcode string
		qLast := `
			SELECT barcode_id 
			FROM buecher_exemplare 
			WHERE barcode_id LIKE 'B-%' 
			ORDER BY barcode_id DESC 
			LIMIT 1
		`
		err = tx.QueryRow(ctx, qLast).Scan(&lastBarcode)
		startNum := 10001
		if err == nil {
			re := regexp.MustCompile(`B-(\d+)`)
			matches := re.FindStringSubmatch(lastBarcode)
			if len(matches) > 1 {
				if parsed, err := strconv.Atoi(matches[1]); err == nil {
					startNum = parsed + 1
				}
			}
		}

		labels := make([]BarcodeLabelDetail, 0)
		orderSummaryItems := make([]OrderedItem, 0)
		currentBarcodeIndex := startNum
		isNaacher := strings.Contains(strings.ToLower(toEmail), "naacher")

		var copyRows [][]any

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
				copyRows = append(copyRows, []any{item.ID, barcodeID, "bestellt", false, isNaacher})
				labels = append(labels, BarcodeLabelDetail{
					BarcodeID: barcodeID,
					Titel:     item.Titel,
					Autor:     item.Autor,
				})
				currentBarcodeIndex++
			}
		}

		// PR 90: Use pgx.CopyFromRows for massive performance gains in order inserts
		_, err = tx.CopyFrom(
			ctx,
			pgx.Identifier{"buecher_exemplare"},
			[]string{"titel_id", "barcode_id", "zustand_notiz", "ist_ausleihbar", "etikett_gedruckt"},
			pgx.CopyFromRows(copyRows),
		)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

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

		RespondJSON(w, http.StatusOK, map[string]any{
			"status":      "success",
			"message":     fmt.Sprintf("Bestellung erfolgreich an %s gesendet.", toEmail),
			"ordered_qty": len(labels),
			"titles_qty":  len(orderSummaryItems),
		})
	}
}

func (s *Server) ReleaseOrdersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

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

		RespondJSON(w, http.StatusOK, map[string]any{
			"status":         "success",
			"message":        fmt.Sprintf("Lieferung vollständig freigegeben. %d Exemplare im Bestand aktiv.", len(items)),
			"released_count": len(items),
			"released_items": items,
		})
	}
}
