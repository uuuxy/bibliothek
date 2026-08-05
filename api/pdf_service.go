package api

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"bibliothek/pdf"
	"bibliothek/repository"
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
//
// bietetBestellbestaetigung: Lieferanten wie Naacher etikettieren selbst und lassen den
// Auftraggeber danach über einen eigenen Link die Etikettengröße wählen (klein/groß).
// Damit er wählen kann, bekommt er BEIDE fertigen Etikettenformate mitgeschickt statt
// nur des kleinen — Bibliosys entscheidet die Größe nicht vorab.
func (s *PDFService) DispatchOrderEmail(
	supplierEmail, subject, body string,
	summaryItems []OrderedItem,
	labels []BarcodeLabelDetail,
	generateBarcodes bool,
	bietetBestellbestaetigung bool,
	schule pdf.SchuleInfo,
) error {
	// Eine Bedingung, an der ALLES hängt: der Bogen, die CSV und der Satz im Anschreiben,
	// der auf den Bogen verweist. Vorher stand der Satz unabhängig davon im Brief — der
	// Lieferant wurde auf eine Anlage hingewiesen, die nicht existierte.
	mitBarcodebogen := generateBarcodes && len(labels) > 0

	summaryPDF, err := GenerateOrderSummaryPDF(summaryItems, schule, mitBarcodebogen)
	if err != nil {
		return err
	}

	kopf := EtikettKopf{Schulname: schule.Name, Eigentumsvermerk: repository.StandardEigentumsvermerk}

	var barcodePDF []byte
	var barcodeCSV []byte
	var lernmittelPDF []byte
	if mitBarcodebogen {
		// Derselbe Etiketten-Generator wie im Selbstdruck (Druck-Center) — voller Inhalt
		// (Schulname, Signatur, Eigentumsvermerk) statt des früheren schmalen Bogens ohne
		// diese Angaben.
		labelDoc, err2 := GenerateLabelsPDF("zweckform_l4760", 1, false, labels, kopf)
		if err2 != nil {
			return err2
		}
		var labelBuf bytes.Buffer
		if err := labelDoc.Output(&labelBuf); err != nil {
			return err
		}
		barcodePDF = labelBuf.Bytes()

		barcodeCSV, err = GenerateBarcodeCSV(labels)
		if err != nil {
			return err
		}
		if bietetBestellbestaetigung {
			lernmittelPDF, err = GenerateLernmittelEtikettenPDF(labels, kopf)
			if err != nil {
				return err
			}
		}
	}

	attachments := []MailAttachment{
		{
			Name:        fmt.Sprintf("bestellanschreiben_%s.pdf", time.Now().Format(dateFormatISO)),
			ContentType: "application/pdf",
			Data:        summaryPDF,
		},
	}

	if mitBarcodebogen {
		attachments = append(attachments, MailAttachment{
			Name:        fmt.Sprintf("etiketten_klein_%s.pdf", time.Now().Format(dateFormatISO)),
			ContentType: "application/pdf",
			Data:        barcodePDF,
		})
		attachments = append(attachments, MailAttachment{
			Name:        fmt.Sprintf("barcode_mapping_%s.csv", time.Now().Format(dateFormatISO)),
			ContentType: "text/csv",
			Data:        barcodeCSV,
		})
		if bietetBestellbestaetigung {
			attachments = append(attachments, MailAttachment{
				Name:        fmt.Sprintf("etiketten_gross_%s.pdf", time.Now().Format(dateFormatISO)),
				ContentType: "application/pdf",
				Data:        lernmittelPDF,
			})
		}
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
