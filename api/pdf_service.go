package api

import (
	"fmt"
	"log"
	"time"
)

// PDFService handles the generation of PDF documents and email dispatch.
type PDFService struct{}

// NewPDFService creates a new PDFService instance.
func NewPDFService() *PDFService {
	return &PDFService{}
}

// DispatchOrderEmail generates the necessary PDFs and sends the order email to the supplier.
func (s *PDFService) DispatchOrderEmail(
	supplierName, supplierEmail, customerNumber string,
	summaryItems []OrderedItem,
	labels []BarcodeLabelDetail,
	generateBarcodes bool,
) error {
	summaryPDF, err := GenerateOrderSummaryPDF(summaryItems)
	if err != nil {
		return err
	}

	var barcodePDF []byte
	var barcodeCSV []byte
	if generateBarcodes && len(labels) > 0 {
		barcodePDF, err = GenerateBarcodeSheetPDF(labels)
		if err != nil {
			return err
		}
		barcodeCSV, err = GenerateBarcodeCSV(labels)
		if err != nil {
			return err
		}
	}

	emailBody := fmt.Sprintf(
		"Sehr geehrte Damen und Herren,\n\nanbei erhalten Sie unsere Buchbestellung vom %s (Kundennummer: %s) sowie den zugehörigen Barcode-Bogen zur Vorab-Beklebung der Exemplare.\n\nBestellte Titel: %d\nGesamtanzahl Exemplare: %d\n\nMit freundlichen Grüßen,\nSchulbibliothek",
		time.Now().Format("02.01.2006"),
		customerNumber,
		len(summaryItems),
		len(labels),
	)

	attachments := []MailAttachment{
		{
			Name:        fmt.Sprintf("bestellanschreiben_%s.pdf", time.Now().Format("2006-01-02")),
			ContentType: "application/pdf",
			Data:        summaryPDF,
		},
	}

	if generateBarcodes && len(labels) > 0 {
		attachments = append(attachments, MailAttachment{
			Name:        fmt.Sprintf("barcode_bogen_%s.pdf", time.Now().Format("2006-01-02")),
			ContentType: "application/pdf",
			Data:        barcodePDF,
		})
		attachments = append(attachments, MailAttachment{
			Name:        fmt.Sprintf("barcode_mapping_%s.csv", time.Now().Format("2006-01-02")),
			ContentType: "text/csv",
			Data:        barcodeCSV,
		})
	}

	mailReq := MailRequest{
		To:          supplierEmail,
		Subject:     fmt.Sprintf("Buchbestellung Schulbibliothek - %s (Kundennummer %s)", time.Now().Format("02.01.2006"), customerNumber),
		Body:        emailBody,
		Attachments: attachments,
	}

	if err := SendEmail(mailReq); err != nil {
		log.Printf("Failed to send order email to %s: %v", supplierEmail, err)
		return err
	}
	return nil
}
