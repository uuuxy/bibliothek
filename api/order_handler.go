package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
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
	Bestellt     int // bereits bestellte, noch nicht gelieferte Exemplare (Platzhalter)
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
	// Bereits bestellte Exemplare (Platzhalter mit ist_ausleihbar=false und
	// zustand_notiz 'bestellt' / 'Bestellt…' / 'Im Zulauf…', siehe order_service.go
	// und baueBestellpositionen) zählen mit: Sonst ignoriert die Query jede laufende
	// Bestellung und bestellt Woche für Woche dieselben Titel nach, bis die Lieferung
	// physisch eintrifft (Budget-Falle). OrderQty = Fehlmenge NACH Zulauf.
	rows, err := tx.Query(ctx, `
		SELECT t.id, t.titel, coalesce(t.autor, ''), coalesce(t.isbn, ''), coalesce(t.verlag, ''), t.meldebestand,
			v.verfuegbar, v.bestellt
		FROM buecher_titel t
		JOIN LATERAL (
			SELECT
				COUNT(*) FILTER (
					WHERE e.ist_ausleihbar = true
					  AND NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
				)::int AS verfuegbar,
				COUNT(*) FILTER (
					WHERE e.ist_ausgesondert = false
					  AND (coalesce(e.zustand_notiz, '') = 'bestellt'
					       OR coalesce(e.zustand_notiz, '') LIKE 'Bestellt%'
					       OR coalesce(e.zustand_notiz, '') LIKE 'Im Zulauf%')
				)::int AS bestellt
			FROM buecher_exemplare e
			WHERE e.titel_id = t.id
		) v ON true
		WHERE v.verfuegbar + v.bestellt < t.meldebestand
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var itemsToOrder []reorderItem
	for rows.Next() {
		var item reorderItem
		if err := rows.Scan(&item.ID, &item.Titel, &item.Autor, &item.ISBN, &item.Verlag, &item.Meldebestand, &item.Verfuegbar, &item.Bestellt); err == nil {
			item.OrderQty = item.Meldebestand - item.Verfuegbar - item.Bestellt
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
	var lastNum int
	// Numerisch sortieren, NICHT lexikografisch: Bei ORDER BY barcode_id DESC (String)
	// gilt 'B-99999' > 'B-100000' (die '9' schlägt die '1'). Ab dem Übergang zu
	// sechsstelligen Nummern lieferte die Query dauerhaft 'B-99999' als Maximum, das
	// System erzeugte erneut 'B-100000' und lief in den UNIQUE-Constraint — das
	// Bestellsystem fror ein. Der Cast auf die Ziffernfolge sortiert echt numerisch.
	err := tx.QueryRow(ctx, `
		SELECT (substring(barcode_id from 'B-([0-9]+)'))::bigint
		FROM buecher_exemplare
		WHERE barcode_id ~ '^B-[0-9]+$'
		ORDER BY (substring(barcode_id from 'B-([0-9]+)'))::bigint DESC
		LIMIT 1
	`).Scan(&lastNum)
	if err != nil {
		return startNum
	}

	if lastNum+1 > startNum {
		startNum = lastNum + 1
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
// Bestellung per Mail. Sie antwortet NICHT selbst, sondern liefert einen *apierrors.APIError
// (nil bei Erfolg) — damit der Aufrufer erst nach erfolgreichem Mailversand committen
// kann. Zuvor wurde umgekehrt committet und danach gemailt: Fiel der SMTP-Server aus,
// waren die Bestell-Platzhalter bereits hart in der DB, der Händler hatte aber nie eine
// Mail bekommen (Ghost-Order, nur per Hand korrigierbar).
func (s *Server) versendeBestellung(ctx context.Context, toEmail string, isNaacher bool, labels []BarcodeLabelDetail, orderSummaryItems []OrderedItem) *apierrors.APIError {
	settingsRepo := repository.NewSystemSettingsRepository(s.DB.Pool)
	orderSettings, _ := settingsRepo.GetSettings(ctx) //nolint:errcheck
	schule := pdf.SchuleInfo{
		Name:    orderSettings.SchuleName,
		Strasse: orderSettings.SchuleStrasse,
		PLZ:     orderSettings.SchulePLZ,
		Ort:     orderSettings.SchuleOrt,
	}

	summaryPDF, err := GenerateOrderSummaryPDF(orderSummaryItems, schule)
	if err != nil {
		return apierrors.Internal("Bestellübersicht-PDF fehlgeschlagen", err)
	}

	barcodePDF, err := GenerateBarcodeSheetPDF(labels)
	if err != nil {
		return apierrors.Internal("Barcode-PDF fehlgeschlagen", err)
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
		return apierrors.New(http.StatusBadGateway, "Versand an den Lieferanten fehlgeschlagen", err)
	}
	return nil
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

		// WICHTIG: erst mailen, dann committen. Schlägt der Mailversand fehl, greift der
		// defer-Rollback und es entstehen keine Bestell-Platzhalter, die niemand beim
		// Händler abgegeben hat. Erst wenn die Mail draußen ist, wird die Bestellung
		// dauerhaft.
		if apiErr := s.versendeBestellung(ctx, toEmail, isNaacher, labels, orderSummaryItems); apiErr != nil {
			apierrors.SendHTTPError(w, apiErr.StatusCode, apiErr)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
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
