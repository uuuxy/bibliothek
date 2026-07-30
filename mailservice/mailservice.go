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

// ErrHeaderUmbruch meldet einen Zeilenumbruch in einem Kopfzeilen-Wert.
var ErrHeaderUmbruch = errors.New("zeilenumbruch im E-Mail-Kopf")

// HeaderWert stellt sicher, dass ein Wert gefahrlos in eine Kopfzeile geschrieben
// werden kann. Ein CR oder LF darin beendet die Kopfzeile vorzeitig; alles danach
// liest der Mailserver als WEITERE Kopfzeile — so schmuggelt man ein "Bcc:" in eine
// Mail, die für einen einzigen Empfänger gedacht war.
//
// Für die Empfängeradresse prüft mail.ParseAddress das bereits mit (nachgewiesen in
// mailservice_test.go: sieben Angriffsvarianten werden alle abgewiesen). Diese Hürde
// steht hier trotzdem, und zwar direkt an der Schreibstelle — aus drei Gründen:
// Sie gilt auch für Betreff und Absender, sie überlebt einen Umbau, bei dem jemand
// ParseAddress herausnimmt, und sie ist an genau der Stelle sichtbar, an der die
// Kopfzeile entsteht, statt vierzig Zeilen weiter oben.
//
// Zurückgewiesen statt bereinigt: Eine Adresse mit Zeilenumbruch ist keine Adresse,
// die man reparieren möchte — sie ist ein Angriff oder ein Datenfehler. Beides soll
// auffallen, nicht stillschweigend zurechtgebogen werden.
func HeaderWert(feld, wert string) (string, error) {
	if strings.ContainsAny(wert, "\r\n") {
		return "", fmt.Errorf("%w: Feld %q", ErrHeaderUmbruch, feld)
	}
	return wert, nil
}

// baueTextNachricht setzt eine einfache Textmail zusammen und prüft dabei jede
// Kopfzeile. Eine Stelle statt zwei: Vorher stand derselbe Sprintf zweimal im
// Paket, und eine Absicherung an nur einer der beiden Stellen wäre keine.
func baueTextNachricht(sender, to, betreff, body string) ([]byte, error) {
	// Ein Betreff kann aus einer Vorlage in der Datenbank stammen, wo beim Bearbeiten
	// leicht ein Umbruch stehen bleibt. Den weisen wir nicht ab, sondern glätten ihn
	// zu einem Leerzeichen — er ist ein Tippfehler, kein Angriff. Danach greift für
	// alle drei Felder dieselbe Hürde.
	betreff = strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(betreff)

	felder := []struct {
		name string
		wert *string
	}{{"From", &sender}, {"To", &to}, {"Subject", &betreff}}

	for _, f := range felder {
		geprueft, err := HeaderWert(f.name, *f.wert)
		if err != nil {
			return nil, err
		}
		*f.wert = geprueft
	}

	return []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n", sender, to, betreff, body)), nil
}

// ErrSMTPVersand markiert jeden Fehler, der beim Sprechen mit dem SMTP-Server
// entstanden ist — im Unterschied zu einer Störung in der Anwendung selbst.
//
// Der Marker ist nicht Kosmetik: apierrors dampft jede HTTP 500 auf eine neutrale
// Meldung ein ("Ein interner Datenbankfehler ist aufgetreten"). Ein Testversand, der
// den SMTP-Fehler als 500 zurückgibt, verwandelt also genau die Diagnose, für die
// BeschreibeSMTPFehler geschrieben wurde, in eine Meldung über die Datenbank. Anhand
// dieses Markers kann der Aufrufer den Fehler stattdessen als Antwort über den
// Zielserver ausliefern und die Meldung durchreichen.
var ErrSMTPVersand = errors.New("SMTP-Versand fehlgeschlagen")

// BeschreibeSMTPFehler übersetzt die technischen Fehler von net/smtp in eine Meldung,
// aus der hervorgeht, was zu tun ist. Vor allem der Zertifikatsfall lohnt sich:
// Go prüft beim STARTTLS den Hostnamen streng, und viele Schulserver liefern ein
// Zertifikat, das nur auf ihren echten Namen ausgestellt ist (z.B. srv1.<domain>)
// und nicht auf den gewohnten smtp.<domain>-Alias.
//
// Exportiert, weil der Aufbau der Nachricht und ihr Transport in verschiedenen
// Päckchen liegen (api/mail_sender.go baut die Mahnung mit PDF-Anhang, versendet wird
// sie über versand.go) — die Übersetzung muss für beide dieselbe sein.
func BeschreibeSMTPFehler(addr string, err error) error {
	var hostErr x509.HostnameError
	if errors.As(err, &hostErr) {
		names := hostErr.Certificate.DNSNames
		if len(names) == 0 {
			names = []string{hostErr.Certificate.Subject.CommonName}
		}
		return fmt.Errorf(
			"%w (Server %s): Das Zertifikat des Servers gilt für %s, nicht für %q. "+
				"Bitte den SMTP-Host auf einen dieser Namen ändern",
			ErrSMTPVersand, addr, strings.Join(names, ", "), hostErr.Host)
	}
	return fmt.Errorf("%w (Server %s): %w", ErrSMTPVersand, addr, err)
}

// SendTestMail versendet eine einfache Testnachricht, um die SMTP-Konfiguration zu
// validieren. Die Konfiguration kommt aus LadeSMTPKonfig — derselben Quelle, aus der
// auch jeder echte Versand liest. Sonst wäre der Test-Knopf eine Aussage über eine
// Konfiguration, die niemand benutzt.
func SendTestMail(ctx context.Context, dbPool db.PgxPoolIface, to string) error {
	konfig, err := LadeSMTPKonfig(ctx, dbPool)
	if err != nil {
		return err
	}
	if !konfig.IstKonfiguriert() {
		return fmt.Errorf("%w: kein SMTP-Server hinterlegt", ErrMailNichtKonfiguriert)
	}

	parsedSender, err := mail.ParseAddress(konfig.Absender)
	if err != nil {
		return fmt.Errorf("ungültige Absender-E-Mail-Adresse: %w", err)
	}
	sender := parsedSender.Address

	parsedTo, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("ungültige Empfänger-E-Mail-Adresse: %w", err)
	}
	to = parsedTo.Address

	betreff := "Test-E-Mail der Schulbibliothek"
	bodyText := "Dies ist eine automatisch generierte Test-E-Mail zur Überprüfung der SMTP-Konfiguration."

	msg, err := baueTextNachricht(sender, to, betreff, bodyText)
	if err != nil {
		return err
	}

	// Derselbe Transportweg wie jede echte Mail — sonst wäre der Test-Knopf eine
	// Aussage über Code, den der Mahnlauf gar nicht ausführt.
	return VersendeUeberSMTP(konfig, sender, []string{to}, msg)
}
