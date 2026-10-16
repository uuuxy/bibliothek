package bestellung

import (
	"context"
	"fmt"
	"log"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/internal/buchetikett"
	"bibliothek/internal/service"
	"bibliothek/pkg/bestelllink"
	"bibliothek/repository"
)

// bestellmailDaten ist, was die Mail an den Lieferanten braucht: beim ersten Versand aus der
// eben angelegten Bestellung, beim erneuten aus der gespeicherten.
type bestellmailDaten struct {
	Empfaenger   string
	Kundennummer string
	Mittel       string
	Positionen   []bestellterTitel
	// Exemplare ist die bestellte Menge über alle Positionen.
	Exemplare int
	// Etiketten sind die Exemplare der Positionen mit Vorab-Barcode.
	Etiketten         []buchetikett.Etikett
	IstHauptlieferant bool
	// Token ist der Klartext für den Bestätigungs-Link, leer ohne Bestätigungsschritt.
	Token          string
	LinkGueltigBis *time.Time
}

// bestellmailDatenAus füllt die Angaben der Mail aus der eben angelegten Bestellung. Positionen
// und Etiketten haben in internal/service dieselben Felder wie hier; der Topf jedes Etiketts
// kommt mit der Bestellung und nicht aus einer Anfrage.
func bestellmailDatenAus(res *service.BestellErgebnis) bestellmailDaten {
	positionen := make([]bestellterTitel, 0, len(res.BestellteTitel))
	for _, p := range res.BestellteTitel {
		positionen = append(positionen, bestellterTitel(p))
	}
	etiketten := make([]buchetikett.Etikett, 0, len(res.Etiketten))
	for _, e := range res.Etiketten {
		etiketten = append(etiketten, buchetikett.Etikett(e))
	}
	return bestellmailDaten{
		Empfaenger:        res.LieferantEmail,
		Kundennummer:      res.Kundennummer,
		Mittel:            res.Mittel,
		Positionen:        positionen,
		Exemplare:         res.Menge,
		Etiketten:         etiketten,
		IstHauptlieferant: res.IstHauptlieferant,
		Token:             res.BestaetigungsToken,
		LinkGueltigBis:    res.LinkGueltigBis,
	}
}

// sendeBestellmail baut die Mail aus Vorlage und Einstellungen und verschickt sie. mitLink
// sagt, ob der Bestätigungs-Link darin steht. Der erste und der erneute Versand gehen beide
// hier durch, damit der Lieferant beide Male dieselbe Mail bekommt.
func (t *Tueren) sendeBestellmail(ctx context.Context, d bestellmailDaten) (mitLink bool, err error) {
	// Ein Lesefehler ist ein Fehler: Mit den Vorgaben ginge die Mail ohne die Angaben der Schule
	// und ohne Link hinaus. Der Aufrufer vermerkt den Versand dann als gescheitert.
	einstellungen, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
	if err != nil {
		return false, fmt.Errorf("einstellungen lesen: %w", err)
	}
	schule := tuer.SchuleAus(einstellungen)

	betreffVorlage, textVorlage := service.LadeBestellMailVorlage(ctx, t.DB.Pool)
	// Ohne hinterlegte öffentliche Adresse bleibt der Link leer: Die Bestellung geht dann
	// ohne Bestätigungsschritt raus. Ein Link auf den internen Servernamen wäre beim
	// Lieferanten wertlos und sähe trotzdem echt aus.
	link := bestelllink.Adresse(einstellungen.AdresseFuerLinks(), d.Token)
	betreff, text := service.LoeseBestellMailAuf(betreffVorlage, textVorlage, service.BestellMailWerte{
		Kundennummer:    d.Kundennummer,
		AnzahlTitel:     len(d.Positionen),
		AnzahlExemplare: d.Exemplare,
		Link:            link,
		GueltigBis:      d.LinkGueltigBis,
		Mittel:          d.Mittel,
	})

	err = t.verschickeBestellmail(ctx, bestellMail{
		Empfaenger:           d.Empfaenger,
		Betreff:              betreff,
		Text:                 text,
		Positionen:           d.Positionen,
		Etiketten:            d.Etiketten,
		MitVorabBarcodes:     len(d.Etiketten) > 0,
		IstHauptlieferant:    d.IstHauptlieferant,
		MitBestaetigungsLink: link != "",
		Schule:               schule,
		EtikettKopf:          buchetikett.KopfAus(einstellungen),
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
func bestellVersandMeldung(lieferantName string, ohneLink bool) (meldung string, warnung bool) {
	if ohneLink {
		return fmt.Sprintf(
			"Bestellung an %s gesendet, aber ohne Bestätigungs-Link. In den Einstellungen ist keine öffentliche Adresse "+
				"hinterlegt. Den Link nachträglich in der Bestellhistorie erzeugen.", lieferantName), true
	}
	return fmt.Sprintf("Bestellung per E-Mail an %s gesendet", lieferantName), false
}

// versandStatus ist der Status der Antwort zur Rückmeldung: Mit einer Warnung zeigt die
// Oberfläche keinen Erfolg.
func versandStatus(warnung bool) string {
	if warnung {
		return "warning"
	}
	return "success"
}

// fristBestellmailVermerk begrenzt das Schreiben des Vermerks nach einem Versandversuch.
const fristBestellmailVermerk = 5 * time.Second

// merkeBestellmailGescheitert vermerkt den gescheiterten Versand an der Bestellung. Der
// Vermerk hängt nicht an der Anfrage: Hat der Browser das Warten aufgegeben, während der
// Mailserver nicht antwortete, steht er trotzdem an der Bestellung. Klemmt das Schreiben,
// steht es im Server-Log; die Bestellung selbst ist gespeichert.
func (t *Tueren) merkeBestellmailGescheitert(ctx context.Context, bestellungID string) {
	ctx, abbruch := context.WithTimeout(context.WithoutCancel(ctx), fristBestellmailVermerk)
	defer abbruch()
	if err := repository.MerkeBestellmailGescheitert(ctx, t.DB.Pool, bestellungID); err != nil {
		log.Printf("Bestellung %s: Vermerk über den gescheiterten Mailversand nicht geschrieben: %v", bestellungID, err)
	}
}
