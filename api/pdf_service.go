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

// BestellMail bündelt alles, was die Bestellmail an den Lieferanten braucht.
//
// Als Struct und nicht als Parameterreihe: Die Mail hängt an drei unabhängigen Wahrheiten
// (Vorab-Barcodes ja/nein, Hauptlieferant ja/nein, Link ja/nein), und drei aufeinander
// folgende bool-Argumente im Aufruf sind genau die Sorte Stelle, an der eine Vertauschung
// niemandem auffällt — die Mail geht ja trotzdem raus.
type BestellMail struct {
	Empfaenger string
	// Betreff und Text sind bereits aus der Vorlage BESTELLUNG_HAENDLER aufgelöst,
	// damit dieser Service DB-frei bleibt.
	Betreff    string
	Text       string
	Positionen []OrderedItem
	Etiketten  []BarcodeLabelDetail
	// MitVorabBarcodes: Für mindestens eine Position wurden Barcodes reserviert.
	MitVorabBarcodes bool
	// IstHauptlieferant: Der Händler beklebt die Bücher selbst und wählt dafür die
	// Etikettengröße — er bekommt deshalb BEIDE Formate statt nur des kleinen.
	IstHauptlieferant bool
	// MitBestaetigungsLink: In der Mail steht der Link auf die Bestellseite. Dann liegen
	// die Etikettenbögen DORT und nicht an der Mail — siehe bestellAnhaenge.
	MitBestaetigungsLink bool
	// Eigentumsvermerk: der konfigurierte Etiketten-Aufdruck (Einstellung
	// etikett_eigentumsvermerk); leer = Werksvorgabe. Bis zum 01.09.2026 nagelte
	// dieser Mailweg den Vermerk auf die Werksvorgabe fest, während Selbstdruck
	// und Lieferanten-Link die Einstellung lasen — zwei Wege zum selben Buch,
	// zwei verschiedene Aufkleber (genau die Divergenz, die der Kommentar in
	// bestellbestaetigung_etiketten.go ausschließen will).
	Eigentumsvermerk string
	Schule           pdf.SchuleInfo
}

// DispatchOrderEmail erzeugt die PDFs und verschickt die Bestellmail an den Lieferanten.
func (s *PDFService) DispatchOrderEmail(m BestellMail) error {
	anhaenge, err := bestellAnhaenge(m)
	if err != nil {
		return err
	}

	mailReq := MailRequest{
		To:          m.Empfaenger,
		Subject:     m.Betreff,
		Body:        m.Text,
		Attachments: anhaenge,
	}

	if err := SendEmail(mailReq); err != nil {
		log.Printf("Failed to send order email to %s: %v", m.Empfaenger, err)
		return err
	}
	return nil
}

// bestellAnhaenge stellt die Anlagen der Bestellmail zusammen.
//
// Die Etikettenbögen sind der springende Punkt. Geht ein Bestätigungs-Link mit, hängen
// sie NICHT an der Mail: Der Händler holt sie über den Link, und genau dieser Weg trägt
// die Bestätigung, die in der Bestellhistorie erscheint. Lägen die fertigen Bögen daneben
// im Postfach, druckte er sie von dort — und die Schule wartete auf eine Bestätigung, die
// nie kommt, obwohl die Bücher längst beklebt unterwegs sind.
//
// Ohne Link bleibt es beim alten Weg (Rückfallebene): Der Bogen MUSS dann beiliegen,
// sonst kann der Händler gar nicht bekleben.
func bestellAnhaenge(m BestellMail) ([]MailAttachment, error) {
	mitBarcodebogen := m.MitVorabBarcodes && len(m.Etiketten) > 0

	// Eine Entscheidung, an der ALLES hängt: die Bögen, die CSV und der Satz im
	// Anschreiben, der auf sie verweist. Vorher stand der Satz unabhängig davon im
	// Brief — der Lieferant wurde auf eine Anlage hingewiesen, die nicht existierte.
	weg := ohneEtiketten
	switch {
	case mitBarcodebogen && m.MitBestaetigungsLink:
		weg = bogenHinterLink
	case mitBarcodebogen:
		weg = bogenLiegtBei
	}

	summaryPDF, err := GenerateOrderSummaryPDF(m.Positionen, m.Schule, weg)
	if err != nil {
		return nil, err
	}
	anhaenge := []MailAttachment{
		{Name: datiertName("bestellanschreiben", "pdf"), ContentType: contentTypePDF, Data: summaryPDF},
	}
	if !mitBarcodebogen {
		return anhaenge, nil
	}

	// Die Barcode-Liste geht in beiden Fällen mit: Sie ist die Zuordnung Barcode↔Titel für
	// die Warenwirtschaft des Händlers, kein Druckerzeugnis — der Link ersetzt sie nicht.
	barcodeCSV, err := GenerateBarcodeCSV(m.Etiketten)
	if err != nil {
		return nil, err
	}
	anhaenge = append(anhaenge,
		MailAttachment{Name: datiertName("barcode_mapping", "csv"), ContentType: "text/csv", Data: barcodeCSV})

	if weg == bogenHinterLink {
		return anhaenge, nil
	}

	boegen, err := etikettenboegen(m.Etiketten, m.Schule, m.IstHauptlieferant, m.Eigentumsvermerk)
	if err != nil {
		return nil, err
	}
	return append(anhaenge, boegen...), nil
}

// etikettenboegen erzeugt die Etiketten-PDFs für die Mail: immer den kleinen Bogen, für
// den selbst beklebenden Hauptlieferanten zusätzlich das große Lernmittel-Etikett — er
// wählt die Größe, Bibliosys entscheidet sie nicht vorab.
func etikettenboegen(labels []BarcodeLabelDetail, schule pdf.SchuleInfo, istHauptlieferant bool, eigentumsvermerk string) ([]MailAttachment, error) {
	// Konfigurierter Vermerk vor Werksvorgabe — dieselbe Regel wie s.etikettKopf
	// (Selbstdruck) und der Lieferanten-Link, damit alle drei Wege zum selben
	// Buch denselben Aufkleber ergeben.
	if eigentumsvermerk == "" {
		eigentumsvermerk = repository.StandardEigentumsvermerk
	}
	kopf := EtikettKopf{Schulname: schule.Name, Eigentumsvermerk: eigentumsvermerk}

	// Derselbe Etiketten-Generator wie im Selbstdruck (Druck-Center) — voller Inhalt
	// (Schulname, Signatur, Eigentumsvermerk) statt des früheren schmalen Bogens ohne
	// diese Angaben.
	labelDoc, err := GenerateLabelsPDF("zweckform_l4760", 1, false, labels, kopf)
	if err != nil {
		return nil, err
	}
	var labelBuf bytes.Buffer
	if err := labelDoc.Output(&labelBuf); err != nil {
		return nil, err
	}
	boegen := []MailAttachment{
		{Name: datiertName("etiketten_klein", "pdf"), ContentType: contentTypePDF, Data: labelBuf.Bytes()},
	}

	if !istHauptlieferant {
		return boegen, nil
	}

	lernmittelPDF, err := GenerateLernmittelEtikettenPDF(labels, kopf)
	if err != nil {
		return nil, err
	}
	return append(boegen,
		MailAttachment{Name: datiertName("etiketten_gross", "pdf"), ContentType: contentTypePDF, Data: lernmittelPDF}), nil
}

// datiertName baut den Dateinamen einer Anlage — das Datum steht im Postfach des Händlers
// zwischen allen anderen Bestellungen und ist dort die einzige Unterscheidung.
func datiertName(basis, endung string) string {
	return fmt.Sprintf("%s_%s.%s", basis, time.Now().Format(dateFormatISO), endung)
}
