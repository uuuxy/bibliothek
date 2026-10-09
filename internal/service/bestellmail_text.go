package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bibliothek/db"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Der Text der Bestellmail an den Lieferanten: Vorlage laden, Platzhalter ersetzen und die zwei
// Absätze unterbringen, an denen ein Ablauf hängt. Die Vorlage ist im Vorlagen-Editor frei
// änderbar. Fällt beim Umformulieren der Platzhalter für den Link weg, bestätigt kein Händler
// mehr; fällt der für den Topf weg, fehlt der Vermerk, nach dem der Händler seinen Nachlass
// richtet. LoeseBestellMailAuf hängt beide an, wenn die Vorlage sie nicht selbst setzt.

// Der Text, der gilt, wenn die Vorlage BESTELLUNG_HAENDLER fehlt oder ein Feld leer ist: Die
// Bestellung bleibt so versandfähig. {{.Mittel}} steht im Betreff, damit der Topf im Postfach
// des Händlers zwischen allen anderen Bestellungen sichtbar ist, und im Text mit der Bitte, ihn
// auf die Rechnung zu übernehmen.
const (
	bestellMailVorgabeBetreff = "Buchbestellung {{.Mittel}} - {{.Datum}} (Kundennummer {{.Kundennummer}})"
	bestellMailVorgabeText    = "Sehr geehrte Damen und Herren,\n\nanbei erhalten Sie unsere Buchbestellung vom {{.Datum}} (Kundennummer: {{.Kundennummer}}) sowie den zugehörigen Barcode-Bogen zur Vorab-Beklebung der Exemplare.\n\nDiese Bestellung: {{.Mittel}} — den Vermerk finden Sie auch im Anschreiben; bitte führen Sie ihn auf der Rechnung.\n\nBestellte Titel: {{.AnzahlTitel}}\nGesamtanzahl Exemplare: {{.AnzahlExemplare}}\n\nMit freundlichen Grüßen,\nSchulbibliothek"
)

// BestellVorlage lädt Betreff und Text der Bestellmail. Fehlt die gespeicherte Vorlage, ist ein
// Feld leer oder lässt sie sich nicht lesen, gilt der Text oben.
func BestellVorlage(ctx context.Context, pool db.PgxPoolIface) (betreff, text string) {
	betreff, text, err := repository.LadeBestellVorlage(ctx, pool)
	if err != nil || betreff == "" || text == "" {
		return bestellMailVorgabeBetreff, bestellMailVorgabeText
	}
	return betreff, text
}

// platzhalterMittel steht in der Vorlage für den Topf der Bestellung.
const platzhalterMittel = "{{.Mittel}}"

// BestellMailWerte sind die Angaben der Bestellung, die die Platzhalter der Vorlage füllen.
type BestellMailWerte struct {
	Kundennummer    string
	AnzahlTitel     int
	AnzahlExemplare int
	// Link ist der Bestätigungs-Link für Lieferanten, die selbst etikettieren. Er ist leer,
	// wenn dieser Lieferant keinen bekommt oder keine öffentliche Adresse hinterlegt ist.
	Link string
	// GueltigBis ist der Ablauf des Links, nil ohne Link. Die Mail nennt ihn als Datum: Das
	// kann der Händler in den Kalender schreiben, eine Tageszahl müsste er ausrechnen.
	GueltigBis *time.Time
	// Mittel ist der Topf der Bestellung (mitteltopf.Land / mitteltopf.Schultraeger).
	Mittel string
}

// LoeseBestellMailAuf ersetzt die Platzhalter der Bestellvorlage in Betreff und Text. Setzt die
// Vorlage den Topf oder den Link nicht selbst, hängt die Funktion sie an: Der Vermerk ist
// Pflicht auf der Bestellung, und ohne den Link bestätigt der Händler nicht.
func LoeseBestellMailAuf(betreff, text string, w BestellMailWerte) (subject, body string) {
	// Ohne gültigen Topf bleiben Platzhalter und Vermerk aus. Verschickt wird eine solche Mail
	// nicht: Ihr Anschreiben lehnt den Topf ab.
	texte, err := mitteltopf.TexteFuer(w.Mittel)
	if err != nil {
		texte = mitteltopf.Texte{}
	}
	replacer := strings.NewReplacer(
		"{{.Datum}}", schulzeit.Jetzt().Format(dateFormatDE),
		"{{.Kundennummer}}", w.Kundennummer,
		"{{.AnzahlTitel}}", strconv.Itoa(w.AnzahlTitel),
		"{{.AnzahlExemplare}}", strconv.Itoa(w.AnzahlExemplare),
		"{{.BestaetigungsLink}}", w.Link,
		"{{.LinkGueltigBis}}", linkFrist(w.GueltigBis),
		platzhalterMittel, texte.Kurz,
	)
	subject, body = ergaenzeMittelVermerk(replacer.Replace(betreff), replacer.Replace(text), betreff, text, texte)
	return subject, ergaenzeLinkAbsatz(body, text, w.Link, w.GueltigBis)
}

// ergaenzeMittelVermerk hängt den Topf an, wo die Vorlage {{.Mittel}} nicht selbst setzt: an
// den Betreff als Zusatz, an den Text als eigenen Absatz. Geprüft wird die rohe Vorlage, wie
// in ergaenzeLinkAbsatz. Ohne Topf bleibt alles unverändert.
func ergaenzeMittelVermerk(subject, body, rohBetreff, rohText string, texte mitteltopf.Texte) (string, string) {
	if texte.Kurz == "" {
		return subject, body
	}
	if !strings.Contains(rohBetreff, platzhalterMittel) {
		subject += " – " + texte.Kurz
	}
	if !strings.Contains(rohText, platzhalterMittel) {
		body += "\n\n" + texte.Vermerk + " Bitte führen Sie diesen Vermerk auch auf der Rechnung."
	}
	return subject, body
}

// linkAbsatz trägt den Link, wenn die Vorlage ihn nicht selbst setzt.
const linkAbsatz = "\n\nEtiketten wählen, drucken und Bestellung bestätigen:\n%s\n\nDer Link ist bis zum %s gültig und gehört nur zu dieser Bestellung."

// linkFrist nennt den Ablauf für Mail und Platzhalter {{.LinkGueltigBis}} als Kalendertag der
// Schule. Der Wert kommt als TIMESTAMPTZ aus der Datenbank und trägt die Zone des Servers, im
// Container UTC: Ohne die Umrechnung stünde ein Ablauf kurz nach Mitternacht mit dem Vortag in
// der Mail, und der Händler hielte den Link einen Tag zu früh für abgelaufen.
func linkFrist(gueltigBis *time.Time) string {
	if gueltigBis == nil {
		return ""
	}
	return gueltigBis.In(schulzeit.Zone()).Format(dateFormatDE)
}

// ergaenzeLinkAbsatz hängt den Link an, falls die Vorlage keinen Platzhalter dafür hat. Geprüft
// wird die rohe Vorlage: Nach dem Ersetzen ist nicht mehr zu sehen, ob der Platzhalter je da
// war.
func ergaenzeLinkAbsatz(aufgeloest, rohesTemplate, link string, gueltigBis *time.Time) string {
	if link == "" || strings.Contains(rohesTemplate, "{{.BestaetigungsLink}}") {
		return aufgeloest
	}
	frist := linkFrist(gueltigBis)
	if frist == "" {
		frist = "Ablauf laut Bestellhistorie"
	}
	return aufgeloest + fmt.Sprintf(linkAbsatz, link, frist)
}
