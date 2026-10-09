package api

import (
	"context"
	"fmt"
	"log"
	"time"

	"bibliothek/internal/service"
	"bibliothek/pdf"
	"bibliothek/pkg/bestelllink"
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

// bestellmailDatenAus füllt die Angaben der Mail aus der eben angelegten Bestellung. Positionen
// und Etiketten haben in internal/service dieselben Felder wie hier; der Topf jedes Etiketts
// kommt mit der Bestellung und nicht aus einer Anfrage.
func bestellmailDatenAus(res *service.OrderResult) bestellmailDaten {
	positionen := make([]OrderedItem, 0, len(res.SummaryItems))
	for _, p := range res.SummaryItems {
		positionen = append(positionen, OrderedItem(p))
	}
	etiketten := make([]BarcodeLabelDetail, 0, len(res.Labels))
	for _, e := range res.Labels {
		etiketten = append(etiketten, BarcodeLabelDetail(e))
	}
	return bestellmailDaten{
		Empfaenger:        res.SupplierEmail,
		Kundennummer:      res.CustomerNumber,
		Mittel:            res.Mittel,
		Positionen:        positionen,
		Exemplare:         res.TotalAllocated,
		Etiketten:         etiketten,
		IstHauptlieferant: res.IstHauptlieferant,
		Token:             res.BestaetigungsToken,
		LinkGueltigBis:    res.LinkGueltigBis,
	}
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

	betreff, textBody := service.BestellVorlage(ctx, s.DB.Pool)
	// Ohne hinterlegte öffentliche Adresse bleibt der Link leer: Die Bestellung geht dann
	// ohne Bestätigungsschritt raus. Ein Link auf den internen Servernamen wäre beim
	// Lieferanten wertlos und sähe trotzdem echt aus.
	link := ""
	if settings.OeffentlicheAdresse != nil {
		link = bestelllink.Adresse(*settings.OeffentlicheAdresse, d.Token)
	}
	subject, body := service.LoeseBestellMailAuf(betreff, textBody, service.BestellMailWerte{
		Kundennummer:    d.Kundennummer,
		AnzahlTitel:     len(d.Positionen),
		AnzahlExemplare: d.Exemplare,
		Link:            link,
		GueltigBis:      d.LinkGueltigBis,
		Mittel:          d.Mittel,
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

// bestellVersandMeldung formuliert die Rückmeldung nach dem Versand. ohneLink heißt: Der
// Hauptlieferant soll über einen Link bestätigen, aber es ist keine öffentliche Adresse
// hinterlegt, und die Mail ging ohne Link hinaus. Sie sieht dabei vollständig aus; ohne die
// Warnung fiele es erst auf, wenn die Bestätigung ausbleibt. Eine Warnung und kein Fehler: Die
// Bestellung ist gespeichert, die Barcodes sind reserviert, und der Link lässt sich in der
// Bestellhistorie nachträglich erzeugen.
func bestellVersandMeldung(lieferantName string, ohneLink bool) (status, meldung string) {
	if ohneLink {
		return "warning", fmt.Sprintf(
			"Bestellung an %s gesendet — aber OHNE Bestätigungs-Link: In den Einstellungen ist keine öffentliche Adresse hinterlegt. "+
				"Link nachträglich in der Bestellhistorie erzeugen.", lieferantName)
	}
	return "success", fmt.Sprintf("Bestellung erfolgreich per E-Mail an %s gesendet.", lieferantName)
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
