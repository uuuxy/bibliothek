package api

import (
	"fmt"
	"log"
	"time"

	"bibliothek/pdf"
)

// PDFService handles the generation of PDF documents and email dispatch.
type PDFService struct{}

// NewPDFService creates a new PDFService instance.
func NewPDFService() *PDFService {
	return &PDFService{}
}

// DispatchOrderEmail generates the necessary PDFs and sends the order email to the supplier.
// Betreff (subject) und Text (body) werden vom Aufrufer bereits aus der Vorlage
// BESTELLUNG_HAENDLER aufgelöst übergeben, damit dieser Service DB-frei bleibt.
func (s *PDFService) DispatchOrderEmail(
	supplierEmail, subject, body string,
	summaryItems []OrderedItem,
	labels []BarcodeLabelDetail,
	generateBarcodes bool,
	schule pdf.SchuleInfo,
) error {
	summaryPDF, err := GenerateOrderSummaryPDF(summaryItems, schule)
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

	attachments := []MailAttachment{
		{
			Name:        fmt.Sprintf("bestellanschreiben_%s.pdf", time.Now().Format(dateFormatISO)),
			ContentType: "application/pdf",
			Data:        summaryPDF,
		},
	}

	if generateBarcodes && len(labels) > 0 {
		attachments = append(attachments, MailAttachment{
			Name:        fmt.Sprintf("barcode_bogen_%s.pdf", time.Now().Format(dateFormatISO)),
			ContentType: "application/pdf",
			Data:        barcodePDF,
		})
		attachments = append(attachments, MailAttachment{
			Name:        fmt.Sprintf("barcode_mapping_%s.csv", time.Now().Format(dateFormatISO)),
			ContentType: "text/csv",
			Data:        barcodeCSV,
		})
	}

	mailReq := MailRequest{
		To:          supplierEmail,
		Subject:     subject,
		Body:        body,
		Attachments: attachments,
	}

	if err := SendEmail(mailReq); err != nil {
		log.Printf("Failed to send order email to %s: %v", supplierEmail, err)
		return err
	}
	return nil
}
