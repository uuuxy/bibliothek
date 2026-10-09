package api

import (
	"context"
	"log"
	"time"

	"bibliothek/pdf"
	"bibliothek/repository"
)

// bestellmailDaten ist, was die Mail an den Lieferanten braucht: beim ersten Versand aus der
// eben angelegten Bestellung, beim erneuten aus der gespeicherten.
type bestellmailDaten struct {
	Empfaenger   string
	Kundennummer string
	Mittel       string
	Positionen   []OrderedItem
	// Exemplare ist die bestellte Menge über alle Positionen.
	Exemplare int
	// Etiketten sind die Exemplare der Positionen mit Vorab-Barcode.
	Etiketten         []BarcodeLabelDetail
	IstHauptlieferant bool
	// Token ist der Klartext für den Bestätigungs-Link, leer ohne Bestätigungsschritt.
	Token          string
	LinkGueltigBis *time.Time
}

// sendeBestellmail baut die Mail aus Vorlage und Einstellungen und verschickt sie. mitLink
// sagt, ob der Bestätigungs-Link darin steht. Der erste und der erneute Versand gehen beide
// hier durch, damit der Lieferant beide Male dieselbe Mail bekommt.
func (s *Server) sendeBestellmail(ctx context.Context, d bestellmailDaten) (mitLink bool, err error) {
	settings, _ := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx) //nolint:errcheck
	schule := pdf.SchuleInfo{
		Name:    settings.SchuleName,
		Strasse: settings.SchuleStrasse,
		PLZ:     settings.SchulePLZ,
		Ort:     settings.SchuleOrt,
	}

	betreff, textBody := s.loadBestellTemplate(ctx)
	// Ohne hinterlegte öffentliche Adresse bleibt der Link leer: Die Bestellung geht dann
	// ohne Bestätigungsschritt raus. Ein Link auf den internen Servernamen wäre beim
	// Lieferanten wertlos und sähe trotzdem echt aus.
	link := ""
	if settings.OeffentlicheAdresse != nil {
		link = bestaetigungsLink(*settings.OeffentlicheAdresse, d.Token)
	}
	subject, body := resolveBestellMail(betreff, textBody, bestellMailWerte{
		kundennummer:    d.Kundennummer,
		anzahlTitel:     len(d.Positionen),
		anzahlExemplare: d.Exemplare,
		link:            link,
		gueltigBis:      d.LinkGueltigBis,
		mittel:          d.Mittel,
	})

	err = verschickeBestellmail(BestellMail{
		Empfaenger:           d.Empfaenger,
		Betreff:              subject,
		Text:                 body,
		Positionen:           d.Positionen,
		Etiketten:            d.Etiketten,
		MitVorabBarcodes:     len(d.Etiketten) > 0,
		IstHauptlieferant:    d.IstHauptlieferant,
		MitBestaetigungsLink: link != "",
		Schule:               schule,
		EtikettKopf:          etikettKopfAus(settings),
		Mittel:               d.Mittel,
	})
	return link != "", err
}

// fristBestellmailVermerk begrenzt das Schreiben des Vermerks nach einem Versandversuch.
const fristBestellmailVermerk = 5 * time.Second

// merkeBestellmailGescheitert vermerkt den gescheiterten Versand an der Bestellung. Der
// Vermerk hängt nicht an der Anfrage: Hat der Browser das Warten aufgegeben, während der
// Mailserver nicht antwortete, steht er trotzdem an der Bestellung. Klemmt das Schreiben,
// steht es im Server-Log; die Bestellung selbst ist gespeichert.
func (s *Server) merkeBestellmailGescheitert(ctx context.Context, bestellungID string) {
	ctx, abbruch := context.WithTimeout(context.WithoutCancel(ctx), fristBestellmailVermerk)
	defer abbruch()
	if err := repository.MerkeBestellmailGescheitert(ctx, s.DB.Pool, bestellungID); err != nil {
		log.Printf("Bestellung %s: Vermerk über den gescheiterten Mailversand nicht geschrieben: %v", bestellungID, err)
	}
}
