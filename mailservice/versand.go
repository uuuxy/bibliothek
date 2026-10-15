package mailservice

// Der eine Weg jeder Mail zum Server. Zwei Wege hießen zwei Verhaltensweisen im Fehlerfall und
// damit einen Knopf für den Testversand, der etwas anderes tut als der Mahnlauf, den er prüft.

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

// Fristen des Versands. Ohne sie hinge smtp.Dial ohne Ende: Ein Server, der die Verbindung annimmt
// und dann schweigt (Firewall, überlasteter Relay), hielte die Goroutine unbegrenzt fest, und die
// Frist des Kontexts aus fristMiddleware bricht kein blockierendes Lesen auf einem Socket ab. Am
// schwersten träfe das den Wächter für kritische Alarme: Hinge Sende dort, stünde seine Schleife
// für immer, und der Mechanismus, der stille Ausfälle melden soll, fiele selbst still aus.
// Variablen und keine Konstanten, damit Tests sie verkürzen können.
var (
	smtpVerbindungsTimeout = 10 * time.Second
	// Frist für die ganze Sitzung (Begrüßung bis Zustellung). Auf der Verbindung gesetzt, wirkt
	// sie auf jedes Lesen und Schreiben, auch hinter STARTTLS, dessen TLS-Schicht sie erbt.
	smtpSitzungsFrist = 60 * time.Second
)

// VersendeUeberSMTP überträgt eine fertige Nachricht an den konfigurierten Server.
// Fehler tragen den Marker ErrSMTPVersand und die Übersetzung aus
// BeschreibeSMTPFehler; der Aufrufer antwortet damit mit 502 und lesbarer Ursache statt mit einem
// neutralen 500.
func VersendeUeberSMTP(konfig SMTPKonfig, absender string, empfaenger []string, nachricht []byte) error {
	// Die Adressen des Umschlags dürfen keinen Zeilenumbruch enthalten, sonst schmuggelt man
	// Befehle in die Sitzung. net/smtp.SendMail prüft das mit; auf diesem Weg steht die Prüfung
	// ausdrücklich.
	if _, err := KopfzeilenWert("MAIL FROM", absender); err != nil {
		return err
	}
	for _, e := range empfaenger {
		if _, err := KopfzeilenWert("RCPT TO", e); err != nil {
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
	// smtp.NewClient liest schon die Begrüßung des Servers, auch das unter der Frist.
	c, err := smtp.NewClient(conn, konfig.Host)
	if err != nil {
		_ = conn.Close() //nolint:errcheck
		return BeschreibeSMTPFehler(adresse, err)
	}
	defer closeutil.Schliesse(c, "smtp client")

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

	// Ab hier ist die Nachricht zugestellt: Der Server hat den abschließenden Punkt mit 250
	// quittiert. Ein Fehler beim Verabschieden sagt darüber nichts mehr, und manche Server kappen
	// die Verbindung nach der Annahme einfach. Als Fehlschlag gemeldet, zählte der Mahnlauf eine
	// zugestellte Mahnung als „nicht versendet", und der nächste Lauf schickte sie ein zweites
	// Mal an dieselbe Klassenleitung.
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
// Das Zertifikat wird gegen den eingestellten Host geprüft; ohne Prüfung könnte ein Angreifer in
// der Mitte ein beliebiges Zertifikat vorlegen und Zugangsdaten und Inhalt (Schülernamen,
// Mahnungen, Anschriften der Eltern) mitlesen. Für alte Server mit selbst signiertem Zertifikat
// gibt es SMTP_ALLOW_INSECURE_TLS.
//
// Ohne STARTTLS bricht der Versand ab. Sonst hinge die Vertraulichkeit jeder Mahnung an der
// Gegenstelle: Ein Server, der die Erweiterung nicht ankündigt (falsch eingestellt, oder ein
// Angreifer, der sie aus der EHLO-Antwort streicht), bekäme die Mahntexte im Klartext.
// smtp.PlainAuth verweigert die Zugangsdaten über eine unverschlüsselte Verbindung von sich aus,
// den Inhalt aber nicht, und bei einem Relay ohne Zugangsdaten (Versand nach IP, im Schulnetz
// üblich) fiele diese Bremse ganz weg. Der Mailserver der Schule bietet STARTTLS auf 25 und 587
// an; für den Betrieb ändert die Erzwingung nichts.
func sichereVerbindung(c *smtp.Client, host string) error {
	if ok, _ := c.Extension("STARTTLS"); !ok {
		if os.Getenv("SMTP_ALLOW_PLAINTEXT") == "true" {
			log.Printf("mail: WARNUNG — %s bietet kein STARTTLS an, SMTP_ALLOW_PLAINTEXT=true erlaubt den Klartextversand. Mailinhalte (Schülernamen, Elternadressen) gehen unverschlüsselt über das Netz.", host)
			return nil
		}
		return ErrSMTPKlartext
	}
	tlsKonfig := &tls.Config{
		ServerName:         host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: os.Getenv("SMTP_ALLOW_INSECURE_TLS") == "true", // #nosec G402 - bewusst per Env, Default sicher
	}
	return c.StartTLS(tlsKonfig)
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

// uebertrage schickt den Umschlag (MAIL FROM / RCPT TO) und den Text der Nachricht. Sie kehrt
// ohne Fehler zurück, sobald der Server die Nachricht angenommen hat.
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
	// Close schreibt den abschließenden Punkt und liest die Antwort darauf; hier entscheidet sich,
	// ob die Nachricht angenommen wurde.
	return w.Close()
}
