// Package mailservice baut eine Mail mit ihren Anlagen und verschickt sie über den Mailserver
// der Schule, mit erzwungenem STARTTLS. Die Konfiguration kommt aus der Datenbank, ohne
// gespeicherte Zeile aus der Umgebung.
package mailservice

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"bibliothek/db"
)

// ErrUmbruchInKopfzeile meldet einen Zeilenumbruch in einem Kopfzeilen-Wert.
var ErrUmbruchInKopfzeile = errors.New("zeilenumbruch im E-Mail-Kopf")

// KopfzeilenWert stellt sicher, dass ein Wert gefahrlos in eine Kopfzeile geschrieben werden
// kann. Ein CR oder LF darin beendet die Kopfzeile vorzeitig, und alles danach liest der
// Mailserver als weitere Kopfzeile; so schmuggelt man ein "Bcc:" in eine Mail, die für einen
// einzigen Empfänger gedacht war.
//
// Für die Empfängeradresse prüft mail.ParseAddress das schon mit (mailservice_test.go: sieben
// Arten des Angriffs werden alle abgewiesen). Die Prüfung steht trotzdem hier an der Stelle, an
// der die Kopfzeile entsteht: Sie gilt auch für Betreff und Absender, und sie überlebt einen
// Umbau, bei dem jemand ParseAddress herausnimmt.
//
// Abgewiesen statt bereinigt: Eine Adresse mit Zeilenumbruch ist ein Angriff oder ein Fehler in
// den Daten, und beides soll auffallen, statt still zurechtgebogen zu werden.
func KopfzeilenWert(feld, wert string) (string, error) {
	if strings.ContainsAny(wert, "\r\n") {
		return "", fmt.Errorf("%w: Feld %q", ErrUmbruchInKopfzeile, feld)
	}
	return wert, nil
}

// PruefeKopfzeilen prüft die drei Kopfzeilen jeder ausgehenden Mail, der einfachen Testmail hier
// wie der Mail mit Anhängen in mail_sender.go. Ein Zeilenumbruch im Betreff einer Vorlage
// könnte sonst beliebige weitere Kopfzeilen anhängen (Reply-To, ein zweites From, ein
// vorzeitiges Ende des Kopfteils).
//
// Zwei Regeln: Ein Betreff kann aus einer Vorlage in der Datenbank stammen, wo beim Bearbeiten
// leicht ein Umbruch stehen bleibt; der wird geglättet, er ist ein Tippfehler. Eine Adresse mit
// Umbruch wird abgewiesen, sie ist keine Adresse, die man reparieren möchte.
func PruefeKopfzeilen(absender, an, betreff string) (string, string, string, error) {
	betreff = strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(betreff)

	felder := []struct {
		name string
		wert *string
	}{{"From", &absender}, {"To", &an}, {"Subject", &betreff}}

	for _, f := range felder {
		geprueft, err := KopfzeilenWert(f.name, *f.wert)
		if err != nil {
			return "", "", "", err
		}
		*f.wert = geprueft
	}
	return absender, an, betreff, nil
}

// baueTextNachricht setzt eine einfache Textmail zusammen und prüft dabei jede Kopfzeile.
func baueTextNachricht(absender, an, betreff, text string) ([]byte, error) {
	absender, an, betreff, err := PruefeKopfzeilen(absender, an, betreff)
	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n", absender, an, betreff, text)), nil
}

// ErrSMTPVersand markiert jeden Fehler aus dem Gespräch mit dem SMTP-Server, im Unterschied zu
// einer Störung im Programm selbst.
//
// apierrors.AntworteFehler macht aus jedem 500 eine neutrale Meldung („Ein interner
// Datenbankfehler ist aufgetreten"). Ein Testversand, der den Fehler als 500 zurückgäbe,
// verwandelte die Diagnose von BeschreibeSMTPFehler in eine Meldung über die Datenbank. Am
// Marker erkennt der Aufrufer den Fehler und antwortet mit 502 und der Meldung.
var ErrSMTPVersand = errors.New("SMTP-Versand fehlgeschlagen")

// BeschreibeSMTPFehler übersetzt die Fehler von net/smtp in eine Meldung, die sagt, was zu tun
// ist. Vor allem beim Zertifikat lohnt sich das: Go prüft bei STARTTLS den Hostnamen streng, und
// viele Schulserver liefern ein Zertifikat, das nur auf ihren echten Namen ausgestellt ist (etwa
// srv1.<domain>) und nicht auf den gewohnten Namen smtp.<domain>.
//
// Exportiert für den Test der Tür, die aus dem Marker den Status der Antwort macht
// (api/system/mail_einstellungen_test.go).
func BeschreibeSMTPFehler(adresse string, err error) error {
	var hostFehler x509.HostnameError
	if errors.As(err, &hostFehler) {
		namen := hostFehler.Certificate.DNSNames
		if len(namen) == 0 {
			namen = []string{hostFehler.Certificate.Subject.CommonName}
		}
		return fmt.Errorf(
			"%w (Server %s): Das Zertifikat des Servers gilt für %s, nicht für %q. "+
				"Bitte den SMTP-Host auf einen dieser Namen ändern",
			ErrSMTPVersand, adresse, strings.Join(namen, ", "), hostFehler.Host)
	}
	return fmt.Errorf("%w (Server %s): %w", ErrSMTPVersand, adresse, err)
}

// SendeTestmail verschickt eine einfache Testmail, um die SMTP-Konfiguration zu prüfen. Die
// Konfiguration kommt aus LadeSMTPKonfig, derselben Quelle, aus der jeder echte Versand liest;
// sonst sagte der Knopf etwas über eine Konfiguration, die niemand benutzt.
func SendeTestmail(ctx context.Context, pool db.Pool, an string) error {
	konfig, err := LadeSMTPKonfig(ctx, pool)
	if err != nil {
		return err
	}
	if !konfig.IstKonfiguriert() {
		return fmt.Errorf("%w: kein SMTP-Server hinterlegt", ErrMailNichtKonfiguriert)
	}

	gelesenerAbsender, err := mail.ParseAddress(konfig.Absender)
	if err != nil {
		return fmt.Errorf("ungültige Absender-E-Mail-Adresse: %w", err)
	}
	absender := gelesenerAbsender.Address

	gelesenerEmpfaenger, err := mail.ParseAddress(an)
	if err != nil {
		return fmt.Errorf("ungültige Empfänger-E-Mail-Adresse: %w", err)
	}
	an = gelesenerEmpfaenger.Address

	betreff := "Test-E-Mail der Schulbibliothek"
	text := "Dies ist eine automatisch generierte Test-E-Mail zur Überprüfung der SMTP-Konfiguration."

	rohtext, err := baueTextNachricht(absender, an, betreff, text)
	if err != nil {
		return err
	}

	// Derselbe Weg wie jede echte Mail, sonst sagte der Knopf etwas über Code, den der Mahnlauf
	// gar nicht ausführt.
	return VersendeUeberSMTP(konfig, absender, []string{an}, rohtext)
}
