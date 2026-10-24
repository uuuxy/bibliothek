package bestellung

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"

	"bibliothek/api/tuer"
	"bibliothek/internal/buchetikett"
	"bibliothek/mailservice"
	"bibliothek/pdf"
	"bibliothek/pkg/csvutil"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/pkg/schulzeit"
)

// bestellterTitel ist ein bestellter Titel mit seiner Menge, wie Anschreiben und Mail ihn nennen.
type bestellterTitel struct {
	Titel  string
	Autor  string
	ISBN   string
	Verlag string
	Menge  int
}

// bestellMail bündelt alles, was die Bestellmail an den Lieferanten braucht.
//
// Als Struct und nicht als Parameterreihe: Die Mail hängt an drei unabhängigen Wahrheiten
// (Vorab-Barcodes ja/nein, Hauptlieferant ja/nein, Link ja/nein), und drei aufeinander
// folgende bool-Argumente im Aufruf sind genau die Sorte Stelle, an der eine Vertauschung
// niemandem auffällt — die Mail geht ja trotzdem raus.
type bestellMail struct {
	Empfaenger string
	// Betreff und Text sind bereits aus der Vorlage BESTELLUNG_HAENDLER aufgelöst; hier wird
	// nichts aus der Datenbank gelesen.
	Betreff    string
	Text       string
	Positionen []bestellterTitel
	Etiketten  []buchetikett.Etikett
	// MitVorabBarcodes: Für mindestens eine Position wurden Barcodes reserviert.
	MitVorabBarcodes bool
	// IstHauptlieferant: Der Händler beklebt die Bücher selbst und wählt dafür die
	// Etikettengröße — er bekommt deshalb beide Formate statt nur des kleinen.
	IstHauptlieferant bool
	// MitBestaetigungsLink: In der Mail steht der Link auf die Bestellseite. Dann liegen
	// die Etikettenbögen dort und nicht an der Mail — siehe bestellAnhaenge.
	MitBestaetigungsLink bool
	// EtikettKopf: Schulname und Eigentumsvermerke für die Etikettenbögen, fertig gebaut von
	// buchetikett.KopfAus wie im Selbstdruck und hinter dem Lieferanten-Link. Baute dieser
	// Mailweg den Kopf selbst, ergäben zwei Wege zum selben Buch zwei verschiedene Aufkleber.
	EtikettKopf buchetikett.Kopf
	Schule      pdf.SchuleInfo
	// Mittel: der Topf der Bestellung (mitteltopf.Land / mitteltopf.Schultraeger) —
	// bestimmt Betreff und Vermerk des Anschreibens. Pflicht: Ohne gültigen Topf gibt es
	// kein Anschreiben und damit keine Mail (mitteltopf.TexteFuer).
	Mittel string
}

// verschickeBestellmail erzeugt die Anlagen und verschickt die Bestellmail an den Lieferanten.
func (t *Tueren) verschickeBestellmail(ctx context.Context, m bestellMail) error {
	anhaenge, err := bestellAnhaenge(m)
	if err != nil {
		return err
	}

	nachricht := mailservice.Nachricht{
		An:       m.Empfaenger,
		Betreff:  m.Betreff,
		Text:     m.Text,
		Anhaenge: anhaenge,
	}

	if err := t.Post().Sende(ctx, nachricht); err != nil {
		slog.Error("Failed to send order email", "empfaenger", m.Empfaenger, "fehler", err)
		return err
	}
	return nil
}

// bestellAnhaenge stellt die Anlagen der Bestellmail zusammen.
//
// Die Etikettenbögen sind der springende Punkt. Geht ein Bestätigungs-Link mit, hängen
// sie nicht an der Mail: Der Händler holt sie über den Link, und genau dieser Weg trägt
// die Bestätigung, die in der Bestellhistorie erscheint. Lägen die fertigen Bögen daneben
// im Postfach, druckte er sie von dort — und die Schule wartete auf eine Bestätigung, die
// nie kommt, obwohl die Bücher längst beklebt unterwegs sind.
//
// Ohne Link bleibt es beim alten Weg (Rückfallebene): Der Bogen muss dann beiliegen,
// sonst kann der Händler gar nicht bekleben.
func bestellAnhaenge(m bestellMail) ([]mailservice.Anhang, error) {
	mitBarcodebogen := m.MitVorabBarcodes && len(m.Etiketten) > 0

	// Eine Entscheidung, an der alles hängt: die Bögen, die CSV und der Satz im
	// Anschreiben, der auf sie verweist. Stünde der Satz unabhängig davon im Brief, wiese er
	// den Lieferanten auf eine Anlage hin, die es nicht gibt.
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
	anschreibenPDF, err := pdf.ErzeugeBestellanschreiben(anschreiben, m.Schule)
	if err != nil {
		return nil, err
	}
	anhaenge := []mailservice.Anhang{
		{Name: datiertName("bestellanschreiben", "pdf"), ContentType: tuer.ContentTypePDF, Inhalt: anschreibenPDF},
	}
	if !mitBarcodebogen {
		return anhaenge, nil
	}

	// Die Barcode-Liste geht in beiden Fällen mit: Sie ist die Zuordnung Barcode↔Titel für
	// die Warenwirtschaft des Händlers, kein Druckerzeugnis — der Link ersetzt sie nicht.
	barcodeCSV, err := erzeugeBarcodeCSV(m.Etiketten)
	if err != nil {
		return nil, err
	}
	anhaenge = append(anhaenge,
		mailservice.Anhang{Name: datiertName("barcode_mapping", "csv"), ContentType: "text/csv", Inhalt: barcodeCSV})

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
// Texten zum Topf (pkg/mitteltopf); ein unbekannter Topf ist ein Fehler und kein Brief ohne
// Vermerk.
func bestellanschreiben(positionen []bestellterTitel, weg pdf.EtikettenWeg, mittel string) (pdf.Bestellanschreiben, error) {
	texte, err := mitteltopf.TexteFuer(mittel)
	if err != nil {
		return pdf.Bestellanschreiben{}, err
	}
	zeilen := make([]pdf.BestellPosition, 0, len(positionen))
	for _, p := range positionen {
		zeilen = append(zeilen, pdf.BestellPosition{Titel: p.Titel, Autor: p.Autor, ISBN: p.ISBN, Menge: p.Menge})
	}
	return pdf.Bestellanschreiben{Betreff: texte.Betreff, Vermerk: texte.Vermerk, Etiketten: weg, Positionen: zeilen}, nil
}

// erzeugeBarcodeCSV schreibt die Zuordnung von Barcode und ISBN für die Warenwirtschaft des
// Lieferanten.
func erzeugeBarcodeCSV(etiketten []buchetikett.Etikett) ([]byte, error) {
	var buf bytes.Buffer
	schreiber := csv.NewWriter(&buf)
	schreiber.Comma = ';' // Semikolon, wie Tabellenprogramme in Europa es erwarten

	// Kopfzeile
	if err := schreiber.Write([]string{"ISBN", "Titel", "Autor", "Barcode"}); err != nil {
		return nil, err
	}

	for _, e := range etiketten {
		// Schutz vor Formel-Injection (CWE-1236): Titel/Autor stammen aus Katalog-Importen.
		if err := schreiber.Write(csvutil.EntschaerfeZeile([]string{e.ISBN, e.Titel, e.Autor, e.BarcodeID})); err != nil {
			return nil, err
		}
	}
	schreiber.Flush()
	return buf.Bytes(), schreiber.Error()
}

// etikettenboegen erzeugt die Etiketten-PDFs für die Mail: immer den kleinen Bogen, für
// den selbst beklebenden Hauptlieferanten zusätzlich das große Lernmittel-Etikett — er
// wählt die Größe, Bibliosys entscheidet sie nicht vorab. Gilt die Bestellung der
// Schülerbücherei, entfällt das große Etikett (mitteltopf.HatGrossesLernmittelEtikett).
//
// Der Kopf kommt fertig herein (buchetikett.KopfAus) — derselbe wie im Selbstdruck und hinter
// dem Lieferanten-Link, damit alle drei Wege zum selben Buch denselben Aufkleber ergeben.
func etikettenboegen(bestellEtiketten []buchetikett.Etikett, kopf buchetikett.Kopf, istHauptlieferant bool, mittel string) ([]mailservice.Anhang, error) {
	// Derselbe Etiketten-Generator wie im Selbstdruck (Druck-Center) — voller Inhalt
	// (Schulname, Signatur, Eigentumsvermerk) statt des früheren schmalen Bogens ohne
	// diese Angaben.
	etiketten := buchetikett.PDFEtiketten(bestellEtiketten, kopf)
	bogen, err := pdf.ErzeugeBuchEtiketten("zweckform_l4760", 1, false, etiketten)
	if err != nil {
		return nil, err
	}
	var bogenPuffer bytes.Buffer
	if err := bogen.Output(&bogenPuffer); err != nil {
		return nil, err
	}
	boegen := []mailservice.Anhang{
		{Name: datiertName("etiketten_klein", "pdf"), ContentType: tuer.ContentTypePDF, Inhalt: bogenPuffer.Bytes()},
	}

	if !istHauptlieferant || !mitteltopf.HatGrossesLernmittelEtikett(mittel) {
		return boegen, nil
	}

	lernmittelPDF, err := pdf.ErzeugeLernmittelEtiketten(etiketten)
	if err != nil {
		return nil, err
	}
	return append(boegen,
		mailservice.Anhang{Name: datiertName("etiketten_gross", "pdf"), ContentType: tuer.ContentTypePDF, Inhalt: lernmittelPDF}), nil
}

// datiertName baut den Dateinamen einer Anlage mit dem Kalendertag der Schule: Das Datum steht
// im Postfach des Händlers zwischen allen anderen Bestellungen und ist dort die einzige
// Unterscheidung.
func datiertName(basis, endung string) string {
	return fmt.Sprintf("%s_%s.%s", basis, schulzeit.Jetzt().Format(tuer.DatumsformatISO), endung)
}
