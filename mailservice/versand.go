package mailservice

// versand.go — Der Transportweg für JEDE Mail dieser Anwendung.
//
// Vorher gab es zwei: net/smtp.SendMail für den Testversand und eine eigene
// Client-Schleife (mit verifiziertem STARTTLS) für Mahnungen, Abgänger und
// Bestellungen. Zwei Wege heißt zwei Verhaltensweisen im Fehlerfall — und damit
// einen Test-Knopf, der etwas anderes tut als der Mahnlauf, den er prüfen soll.

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/smtp"
	"os"
	"time"

	"bibliothek/pkg/closeutil"
)

// Fristen des SMTP-Transports. Ohne sie war smtp.Dial der klassische Ewig-Hänger
// (Ausfallmatrix 20.08.2026): Ein Server, der die TCP-Verbindung annimmt und dann
// schweigt (Firewall-Blackhole, überlasteter Relay), blockierte die Goroutine
// unbegrenzt — die Context-Deadline der TimeoutMiddleware bricht kein blockierendes
// Read auf einem Socket ab. Am schwersten traf das den Kritisch-Alarm-Wächter selbst:
// Hängt SendEmail dort, steht seine Ticker-Schleife für immer, und der Mechanismus,
// der stille Ausfälle melden soll, ist selbst still ausgefallen.
// Variablen (keine Konstanten), damit Tests sie verkürzen können.
var (
	smtpVerbindungsTimeout = 10 * time.Second
	// Gesamtfrist für die komplette SMTP-Sitzung (Begrüßung bis Zustellung). Als
	// Deadline auf der Verbindung gesetzt, wirkt sie auf jedes Read/Write — auch
	// hinter STARTTLS, dessen TLS-Schicht die Deadline der Rohverbindung erbt.
	smtpSitzungsFrist = 60 * time.Second
)

// VersendeUeberSMTP überträgt eine fertige Nachricht an den konfigurierten Server.
// Fehler tragen den Marker ErrSMTPVersand und die Übersetzung aus
// BeschreibeSMTPFehler — der Aufrufer kann sie damit als 502 mit lesbarer Ursache
// ausliefern statt als neutralisierte 500.
func VersendeUeberSMTP(konfig SMTPKonfig, absender string, empfaenger []string, nachricht []byte) error {
	// Envelope-Adressen dürfen keinen Zeilenumbruch enthalten — sonst schmuggelt man
	// SMTP-Befehle in die Sitzung. net/smtp.SendMail prüfte das mit; auf diesem Weg
	// muss die Hürde ausdrücklich stehen.
	if _, err := HeaderWert("MAIL FROM", absender); err != nil {
		return err
	}
	for _, e := range empfaenger {
		if _, err := HeaderWert("RCPT TO", e); err != nil {
			return err
		}
	}

	adresse := konfig.Adresse()
	conn, err := net.DialTimeout("tcp", adresse, smtpVerbindungsTimeout)
	if err != nil {
		return BeschreibeSMTPFehler(adresse, err)
	}
	if err := conn.SetDeadline(time.Now().Add(smtpSitzungsFrist)); err != nil {
		_ = conn.Close() //nolint:errcheck
		return BeschreibeSMTPFehler(adresse, err)
	}
	// smtp.NewClient liest bereits die Server-Begrüßung — auch das unter der Deadline.
	c, err := smtp.NewClient(conn, konfig.Host)
	if err != nil {
		_ = conn.Close() //nolint:errcheck
		return BeschreibeSMTPFehler(adresse, err)
	}
	defer closeutil.LogClose(c, "smtp client")

	if err := c.Hello("localhost"); err != nil {
		return BeschreibeSMTPFehler(adresse, err)
	}
	if err := sichereVerbindung(c, konfig.Host); err != nil {
		return BeschreibeSMTPFehler(adresse, err)
	}
	if err := authentifiziere(c, konfig.Auth()); err != nil {
		return BeschreibeSMTPFehler(adresse, err)
	}

	if err := uebertrage(c, absender, empfaenger, nachricht); err != nil {
		return BeschreibeSMTPFehler(adresse, err)
	}

	// Ab hier ist die Nachricht zugestellt: Der Server hat den abschließenden Punkt
	// mit 250 quittiert. Ein Fehler beim Verabschieden sagt darüber nichts mehr aus —
	// und manche Server kappen die Verbindung nach der Annahme einfach.
	//
	// Das als Fehlschlag zu melden, wäre teuer: Der Mahnlauf zählte eine zugestellte
	// Mahnung als „nicht versendet", und der nächste Lauf schickte sie ein zweites Mal
	// an dieselbe Klassenleitung.
	if err := c.Quit(); err != nil {
		log.Printf("mail: Verbindung zu %s wurde nach der Annahme nicht sauber beendet (Nachricht ist zugestellt): %v", adresse, err)
	}
	return nil
}

// ErrSMTPKlartext meldet, dass der Server kein STARTTLS anbietet und der Versand
// deshalb unterblieben ist. Eigener Fehlerwert, damit der Betrieb die Ursache in der
// Oberfläche erkennt statt eines allgemeinen Verbindungsfehlers.
var ErrSMTPKlartext = errors.New("SMTP-Server bietet kein STARTTLS an — Versand abgebrochen, weil die Nachricht sonst im Klartext über das Netz ginge (Ausnahme: SMTP_ALLOW_PLAINTEXT=true)")

// sichereVerbindung hebt die Sitzung auf TLS und bricht ab, wenn das nicht geht.
//
// Das Zertifikat wird gegen den konfigurierten Host VERIFIZIERT — ohne Verifikation
// könnte ein MITM beim Upgrade ein beliebiges Zertifikat vorlegen und sowohl die
// SMTP-AUTH-Zugangsdaten als auch den Mailinhalt (Schülernamen, Mahndaten,
// Elternadressen) mitlesen.
//
// Bietet der Server gar kein STARTTLS an, galt hier bisher "dann eben ohne" — die
// Funktion hieß starttlsWennMoeglich und gab in diesem Fall nil zurück. Damit hing die
// Vertraulichkeit jeder Mahnung am Wohlwollen der Gegenstelle: Ein Server, der die
// Erweiterung nicht ankündigt (falsch konfiguriert, oder ein MITM, der sie aus der
// EHLO-Antwort streicht — "STARTTLS stripping"), bekam Mahntexte mit Schülernamen und
// Elternadressen im Klartext. Die AUTH-Zugangsdaten waren dabei nie in Gefahr:
// smtp.PlainAuth verweigert die Übertragung über eine unverschlüsselte Verbindung von
// sich aus. Der Inhalt aber schon, und bei einem Relay ohne Zugangsdaten (Versand nach
// IP, im Schulnetz üblich) fiel diese Bremse ganz weg.
//
// Geprüft am 06.08.2026: srv1.philipp-reis-schule.de bietet STARTTLS auf 25 und 587 an
// (EHLO-Probe, ohne Anmeldung). Für den Schulbetrieb ändert die Erzwingung also nichts.
func sichereVerbindung(c *smtp.Client, host string) error {
	if ok, _ := c.Extension("STARTTLS"); !ok {
		if os.Getenv("SMTP_ALLOW_PLAINTEXT") == "true" {
			log.Printf("mail: WARNUNG — %s bietet kein STARTTLS an, SMTP_ALLOW_PLAINTEXT=true erlaubt den Klartextversand. Mailinhalte (Schülernamen, Elternadressen) gehen unverschlüsselt über das Netz.", host)
			return nil
		}
		return ErrSMTPKlartext
	}
	config := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}
	return c.StartTLS(config)
}

// authentifiziere meldet sich an, sofern Zugangsdaten vorliegen und der Server
// Authentifizierung anbietet.
func authentifiziere(c *smtp.Client, a smtp.Auth) error {
	if a == nil {
		return nil
	}
	if ok, _ := c.Extension("AUTH"); ok {
		return c.Auth(a)
	}
	return nil
}

// uebertrage schickt Envelope (MAIL FROM / RCPT TO) und Nachrichtentext. Kehrt ohne
// Fehler zurück, sobald der Server die Nachricht angenommen hat.
func uebertrage(c *smtp.Client, absender string, empfaenger []string, nachricht []byte) error {
	if err := c.Mail(absender); err != nil {
		return err
	}
	for _, e := range empfaenger {
		if err := c.Rcpt(e); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(nachricht); err != nil {
		return err
	}
	// Close schreibt den abschließenden Punkt und liest die Antwort darauf — hier
	// entscheidet sich, ob die Nachricht angenommen wurde.
	return w.Close()
}
