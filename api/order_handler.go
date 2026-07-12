package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pdf"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

type SendOrderMailRequest struct {
	Email string `json:"email"`
}

type reorderItem struct {
	ID           string
	Titel        string
	Autor        string
	ISBN         string
	Verlag       string
	Meldebestand int
	Verfuegbar   int
	OrderQty     int
}

// resolveRecipientEmail ermittelt die Empfängeradresse aus dem Payload bzw. den
// Umgebungsvariablen SMTP_TO / SUPPLIER_EMAIL.
func resolveRecipientEmail(reqEmail string) (string, error) {
	toEmail := reqEmail
	if toEmail == "" {
		toEmail = os.Getenv("SMTP_TO")
	}
	if toEmail == "" {
		toEmail = os.Getenv("SUPPLIER_EMAIL")
	}
	if toEmail == "" {
		return "", errors.New("missing recipient email in payload or environment (SMTP_TO/SUPPLIER_EMAIL)")
	}
	return toEmail, nil
}

// sammleNachbestellungen liefert alle Titel, deren verfügbarer Bestand unter dem
// Meldebestand liegt, inklusive berechneter Bestellmenge (> 0).
func sammleNachbestellungen(ctx context.Context, tx pgx.Tx) ([]reorderItem, error) {
	rows, err := tx.Query(ctx, `
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
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var itemsToOrder []reorderItem
	for rows.Next() {
		var item reorderItem
		if err := rows.Scan(&item.ID, &item.Titel, &item.Autor, &item.ISBN, &item.Verlag, &item.Meldebestand, &item.Verfuegbar); err == nil {
			item.OrderQty = item.Meldebestand - item.Verfuegbar
			if item.OrderQty > 0 {
				itemsToOrder = append(itemsToOrder, item)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return itemsToOrder, nil
}

// naechsteBarcodeNummer bestimmt die nächste laufende "B-#####"-Nummer aus dem
// höchsten bereits vergebenen Barcode. Fallback ist 10001.
func naechsteBarcodeNummer(ctx context.Context, tx pgx.Tx) int {
	startNum := 10001
	var lastBarcode string
	err := tx.QueryRow(ctx, `
		SELECT barcode_id
		FROM buecher_exemplare
		WHERE barcode_id LIKE 'B-%'
		ORDER BY barcode_id DESC
		LIMIT 1
	`).Scan(&lastBarcode)
	if err != nil {
		return startNum
	}

	re := regexp.MustCompile(`B-(\d+)`)
	matches := re.FindStringSubmatch(lastBarcode)
	if len(matches) > 1 {
		if parsed, err := strconv.Atoi(matches[1]); err == nil {
			startNum = parsed + 1
		}
	}
	return startNum
}

// baueBestellpositionen erzeugt aus den Nachbestellungen die Barcode-Etiketten,
// die Bestellübersicht und die CopyFrom-Zeilen für den Exemplar-Insert.
func baueBestellpositionen(itemsToOrder []reorderItem, startNum int, isNaacher bool) ([]BarcodeLabelDetail, []OrderedItem, [][]any) {
	labels := make([]BarcodeLabelDetail, 0)
	orderSummaryItems := make([]OrderedItem, 0)
	var copyRows [][]any
	currentBarcodeIndex := startNum

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
	return labels, orderSummaryItems, copyRows
}

// versendeBestellung erzeugt Bestellübersicht- und Barcode-PDF und schickt die
// Bestellung per Mail. Antwortet selbst (Erfolg oder Fehler) auf w.
func (s *Server) versendeBestellung(ctx context.Context, w http.ResponseWriter, toEmail string, isNaacher bool, labels []BarcodeLabelDetail, orderSummaryItems []OrderedItem) {
	settingsRepo := repository.NewSystemSettingsRepository(s.DB.Pool)
	orderSettings, _ := settingsRepo.GetSettings(ctx)
	schule := pdf.SchuleInfo{
		Name:    orderSettings.SchuleName,
		Strasse: orderSettings.SchuleStrasse,
		PLZ:     orderSettings.SchulePLZ,
		Ort:     orderSettings.SchuleOrt,
	}

	summaryPDF, err := GenerateOrderSummaryPDF(orderSummaryItems, schule)
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
		time.Now().Format(dateFormatDE),
		len(orderSummaryItems),
		len(labels),
	)

	attachments := []MailAttachment{
		{
			Name:        fmt.Sprintf("bestelluebersicht_%s.pdf", time.Now().Format(dateFormatISO)),
			ContentType: contentTypePDF,
			Data:        summaryPDF,
		},
	}
	if isNaacher {
		attachments = append(attachments, MailAttachment{
			Name:        fmt.Sprintf("barcode_bogen_%s.pdf", time.Now().Format(dateFormatISO)),
			ContentType: contentTypePDF,
			Data:        barcodePDF,
		})
	}

	mailReq := MailRequest{
		To:          toEmail,
		Subject:     fmt.Sprintf("Buchbestellung Schulbibliothek - %s", time.Now().Format(dateFormatDE)),
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

func (s *Server) SendOrderMailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SendOrderMailRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		toEmail, err := resolveRecipientEmail(req.Email)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		ctx := r.Context()

		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer db.SafeRollback(ctx, tx)

		itemsToOrder, err := sammleNachbestellungen(ctx, tx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if len(itemsToOrder) == 0 {
			RespondJSON(w, http.StatusOK, map[string]any{
				"status":  "success",
				"message": "Alle Bestände ausreichend. Keine Bestellung notwendig.",
				"ordered": 0,
			})
			return
		}

		isNaacher := strings.Contains(strings.ToLower(toEmail), "naacher")
		startNum := naechsteBarcodeNummer(ctx, tx)
		labels, orderSummaryItems, copyRows := baueBestellpositionen(itemsToOrder, startNum, isNaacher)

		// PR 90: Use pgx.CopyFromRows for massive performance gains in order inserts
		if _, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{"buecher_exemplare"},
			[]string{"titel_id", "barcode_id", "zustand_notiz", "ist_ausleihbar", "etikett_gedruckt"},
			pgx.CopyFromRows(copyRows),
		); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		s.versendeBestellung(ctx, w, toEmail, isNaacher, labels, orderSummaryItems)
	}
}
