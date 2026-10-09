package api

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"time"

	"bibliothek/pdf"
	"bibliothek/pkg/csvutil"
)

// OrderedItem ist ein bestellter Titel mit seiner Menge, wie Anschreiben und Mail ihn nennen.
type OrderedItem struct {
	Titel  string
	Autor  string
	ISBN   string
	Verlag string
	Menge  int
}

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
	// EtikettKopf: Schulname und Eigentumsvermerke für die Etikettenbögen, fertig gebaut
	// von etikettKopfAus. Bis zum 01.09.2026 nagelte dieser Mailweg den Vermerk auf die
	// Werksvorgabe fest, während Selbstdruck und Lieferanten-Link die Einstellung lasen —
	// zwei Wege zum selben Buch, zwei verschiedene Aufkleber. Seit dem 21.09.2026 baut
	// er den Kopf gar nicht mehr selbst.
	EtikettKopf EtikettKopf
	Schule      pdf.SchuleInfo
	// Mittel: der Topf der Bestellung (repository.MittelLand / MittelSchultraeger) —
	// bestimmt Betreff und Vermerk des Anschreibens. Pflicht: Ohne gültigen Topf gibt es
	// kein Anschreiben und damit keine Mail (mittelTexteFuer).
	Mittel string
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
	weg := pdf.OhneEtiketten
	switch {
	case mitBarcodebogen && m.MitBestaetigungsLink:
		weg = pdf.BogenHinterLink
	case mitBarcodebogen:
		weg = pdf.BogenLiegtBei
	}

	anschreiben, err := bestellanschreiben(m.Positionen, weg, m.Mittel)
	if err != nil {
		return nil, err
	}
	summaryPDF, err := pdf.GenerateBestellanschreibenPDF(anschreiben, m.Schule)
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

	if weg == pdf.BogenHinterLink {
		return anhaenge, nil
	}

	boegen, err := etikettenboegen(m.Etiketten, m.EtikettKopf, m.IstHauptlieferant, m.Mittel)
	if err != nil {
		return nil, err
	}
	return append(anhaenge, boegen...), nil
}

// bestellanschreiben füllt die Eingabe des Anschreibens. Betreff und Vermerk kommen aus den
// Texten zum Topf (mittel_vermerk.go); ein unbekannter Topf ist ein Fehler und kein Brief ohne
// Vermerk.
func bestellanschreiben(positionen []OrderedItem, weg pdf.EtikettenWeg, mittel string) (pdf.Bestellanschreiben, error) {
	texte, err := mittelTexteFuer(mittel)
	if err != nil {
		return pdf.Bestellanschreiben{}, err
	}
	zeilen := make([]pdf.BestellPosition, 0, len(positionen))
	for _, p := range positionen {
		zeilen = append(zeilen, pdf.BestellPosition{Titel: p.Titel, Autor: p.Autor, ISBN: p.ISBN, Menge: p.Menge})
	}
	return pdf.Bestellanschreiben{Betreff: texte.Betreff, Vermerk: texte.Vermerk, Etiketten: weg, Positionen: zeilen}, nil
}

// GenerateBarcodeCSV schreibt die Zuordnung von Barcode und ISBN für die Warenwirtschaft des
// Lieferanten.
func GenerateBarcodeCSV(labels []BarcodeLabelDetail) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Comma = ';' // European CSV format

	// Header
	if err := writer.Write([]string{"ISBN", "Titel", "Autor", "Barcode"}); err != nil {
		return nil, err
	}

	for _, l := range labels {
		// Schutz vor Formel-Injection (CWE-1236): Titel/Autor stammen aus Katalog-Importen.
		if err := writer.Write(csvutil.SanitizeRow([]string{l.ISBN, l.Titel, l.Autor, l.BarcodeID})); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buf.Bytes(), writer.Error()
}

// etikettenboegen erzeugt die Etiketten-PDFs für die Mail: immer den kleinen Bogen, für
// den selbst beklebenden Hauptlieferanten zusätzlich das große Lernmittel-Etikett — er
// wählt die Größe, Bibliosys entscheidet sie nicht vorab. Gilt die Bestellung der
// Schülerbücherei, entfällt das große Etikett (grossesLernmittelEtikettFuer).
//
// Der Kopf kommt fertig herein (etikettKopfAus) — derselbe wie im Selbstdruck und hinter
// dem Lieferanten-Link, damit alle drei Wege zum selben Buch denselben Aufkleber ergeben.
func etikettenboegen(labels []BarcodeLabelDetail, kopf EtikettKopf, istHauptlieferant bool, mittel string) ([]MailAttachment, error) {
	// Derselbe Etiketten-Generator wie im Selbstdruck (Druck-Center) — voller Inhalt
	// (Schulname, Signatur, Eigentumsvermerk) statt des früheren schmalen Bogens ohne
	// diese Angaben.
	etiketten := buchEtiketten(labels, kopf)
	labelDoc, err := pdf.GenerateLabelsPDF("zweckform_l4760", 1, false, etiketten)
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

	if !istHauptlieferant || !grossesLernmittelEtikettFuer(mittel) {
		return boegen, nil
	}

	lernmittelPDF, err := pdf.GenerateLernmittelEtikettenPDF(etiketten)
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
