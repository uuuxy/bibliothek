package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Der TEXT der Bestellmail: Vorlage laden, Platzhalter ersetzen, Link unterbringen — und
// die Rückmeldung, die der Betreiber danach auf dem Bildschirm sieht.
//
// Eigene Datei, weil hier eine ganze Fehlerklasse wohnt: Der Ablauf "Händler bestätigt
// über den Link" hängt an einem einzigen Absatz in einem frei editierbaren Text. Fällt er
// weg, funktioniert alles andere weiter — nur klickt niemand mehr.

// bestellVersandMeldung formuliert die Rückmeldung nach dem Versand.
//
// ohneLink ist der stille Ausfall dieses Ablaufs: Der Hauptlieferant soll über einen Link
// bestätigen, aber es ist keine öffentliche Adresse hinterlegt — die Mail geht dann ohne
// Link raus und sieht dabei vollständig aus. Vorher meldete die Oberfläche in genau diesem
// Fall "erfolgreich gesendet"; dass der Händler nichts zum Bestätigen hat, fiel erst
// Wochen später auf, wenn die Bestätigung ausblieb.
//
// Trotzdem nur eine Warnung und kein Fehler: Die Bestellung IST raus, die Barcodes sind
// reserviert, und der Link lässt sich in der Bestellhistorie nachträglich erzeugen.
func bestellVersandMeldung(lieferantName string, ohneLink bool) (status, meldung string) {
	if ohneLink {
		return "warning", fmt.Sprintf(
			"Bestellung an %s gesendet — aber OHNE Bestätigungs-Link: In den Einstellungen ist keine öffentliche Adresse hinterlegt. "+
				"Link nachträglich in der Bestellhistorie erzeugen.", lieferantName)
	}
	return "success", fmt.Sprintf("Bestellung erfolgreich per E-Mail an %s gesendet.", lieferantName)
}

// bestellMailFallback* ist der Standardtext, falls die Vorlage BESTELLUNG_HAENDLER
// fehlt oder ein Feld leer ist — so bleibt der Bestellversand immer versandfähig
// (identisch zum früher hartkodierten Text in pdf_service.go).
const (
	bestellMailFallbackBetreff = "Buchbestellung Schulbibliothek - {{.Datum}} (Kundennummer {{.Kundennummer}})"
	bestellMailFallbackBody    = "Sehr geehrte Damen und Herren,\n\nanbei erhalten Sie unsere Buchbestellung vom {{.Datum}} (Kundennummer: {{.Kundennummer}}) sowie den zugehörigen Barcode-Bogen zur Vorab-Beklebung der Exemplare.\n\nBestellte Titel: {{.AnzahlTitel}}\nGesamtanzahl Exemplare: {{.AnzahlExemplare}}\n\nMit freundlichen Grüßen,\nSchulbibliothek"
)

// loadBestellTemplate lädt die Händler-Bestellvorlage aus der Datenbank; fehlt sie
// oder ist ein Feld leer, greift der hartkodierte Fallback.
func (s *Server) loadBestellTemplate(ctx context.Context) (betreff, textBody string) {
	err := s.DB.Pool.QueryRow(ctx, "SELECT betreff, text_body FROM mail_vorlagen WHERE typ = 'BESTELLUNG_HAENDLER'").Scan(&betreff, &textBody)
	if err != nil || betreff == "" || textBody == "" {
		return bestellMailFallbackBetreff, bestellMailFallbackBody
	}
	return betreff, textBody
}

// resolveBestellMail ersetzt die Platzhalter der Bestellvorlage in Betreff und Text.
//
// link ist der Bestätigungs-Link für Lieferanten, die selbst etikettieren; er ist leer,
// wenn dieser Lieferant keinen bekommt oder keine öffentliche Adresse hinterlegt ist.
func resolveBestellMail(betreff, textBody, kundennummer string, anzahlTitel, anzahlExemplare int, link string) (subject, body string) {
	replacer := strings.NewReplacer(
		"{{.Datum}}", time.Now().Format(dateFormatDE),
		"{{.Kundennummer}}", kundennummer,
		"{{.AnzahlTitel}}", strconv.Itoa(anzahlTitel),
		"{{.AnzahlExemplare}}", strconv.Itoa(anzahlExemplare),
		"{{.BestaetigungsLink}}", link,
	)
	return replacer.Replace(betreff), ergaenzeLinkAbsatz(replacer.Replace(textBody), textBody, link)
}

// linkAbsatz ist der Textblock, der den Link trägt, wenn die Vorlage ihn nicht selbst
// platziert. Die Vorlage ist frei editierbar (Vorlagen-Editor) — ein Lieferant, der den
// Link nicht bekommt, weil jemand den Platzhalter beim Umformulieren verloren hat, wäre
// ein stiller Ausfall des ganzen Ablaufs.
const linkAbsatz = "\n\nEtiketten wählen, drucken und Bestellung bestätigen:\n%s\n\nDer Link ist %d Tage gültig und gehört nur zu dieser Bestellung."

// ergaenzeLinkAbsatz hängt den Link an, falls die Vorlage keinen Platzhalter dafür hat.
// Geprüft wird die ROHE Vorlage, nicht der aufgelöste Text: Nach dem Ersetzen ist nicht
// mehr zu sehen, ob der Platzhalter je da war.
func ergaenzeLinkAbsatz(aufgeloest, rohesTemplate, link string) string {
	if link == "" || strings.Contains(rohesTemplate, "{{.BestaetigungsLink}}") {
		return aufgeloest
	}
	return aufgeloest + fmt.Sprintf(linkAbsatz, link, TokenGueltigkeitTage)
}
