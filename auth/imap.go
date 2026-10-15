package auth

import (
	"bibliothek/pkg/closeutil"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/commands"
)

// imapFrist ist die eine Frist für Verbinden und Anmelden am Mailserver. Die Tür der Anmeldung
// lässt ihr Luft (anmeldeFrist > imapFrist): Sonst endete ihr Kontext zwischen der gelungenen
// Anmeldung am Mailserver und dem Lesen des Kontos, und eine richtige Anmeldung zählte als
// Fehlversuch.
const imapFrist = 15 * time.Second

// imapBefehlsfrist gilt je Befehl an den Mailserver (LOGIN, LOGOUT). go-imap kennt sonst keine
// Frist (Timeout=0): Ein Server, der nach „OK LOGIN" schweigt, hielte die Tür ohne Ende fest.
const imapBefehlsfrist = 5 * time.Second

// errAnmeldungAbgelehnt heißt: Der Server hat geantwortet und die Zugangsdaten abgelehnt (NO
// oder BAD). Nur das ist ein falsches Passwort. Alles andere (Frist abgelaufen, Verbindung
// abgebrochen, keine Begrüßung) ist ein Ausfall, auch wenn die Frist noch läuft: Ein Server, der
// die Verbindung beim LOGIN schließt, hat kein Passwort abgelehnt.
var errAnmeldungAbgelehnt = errors.New("anmeldung fehlgeschlagen")

// verbindeIMAP öffnet die Verbindung zum Mailserver über TLS und liest seine Begrüßung.
// Verbinden und Begrüßung haben zusammen höchstens zehn Sekunden.
func verbindeIMAP(ctx context.Context, adresse string, tlsKonfig *tls.Config) (*client.Client, net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// tls.Dialer beachtet den Kontext schon beim Verbinden.
	netz := &net.Dialer{}
	verbinder := &tls.Dialer{
		NetDialer: netz,
		Config:    tlsKonfig,
	}

	verbindung, err := verbinder.DialContext(ctx, "tcp", adresse)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, nil, fmt.Errorf("zeitüberschreitung bei verbindung")
		}
		if ctx.Err() == context.Canceled {
			return nil, nil, fmt.Errorf("timeout bei verbindung zum server")
		}
		return nil, nil, err
	}

	// client.New liest die Begrüßung und kann dabei hängen. Endet der Kontext vorher, schließt
	// die Goroutine die Verbindung und löst es.
	fertig := make(chan struct{})
	defer close(fertig)

	go func() {
		select {
		case <-ctx.Done():
			closeutil.Schliesse(verbindung, imapVerbindung)
		case <-fertig:
		}
	}()

	c, err := client.New(verbindung)
	if err != nil {
		closeutil.Schliesse(verbindung, imapVerbindung)
		if ctx.Err() == context.DeadlineExceeded {
			return nil, nil, fmt.Errorf("zeitüberschreitung bei verbindung")
		}
		if ctx.Err() == context.Canceled {
			return nil, nil, fmt.Errorf("timeout bei verbindung zum server")
		}
		return nil, nil, err
	}

	return c, verbindung, nil
}

// meldeBeiIMAPAn sendet LOGIN und liefert errAnmeldungAbgelehnt, wenn der Server mit NO oder
// BAD antwortet, sonst den Fehler der Verbindung (Frist, EOF, Verbindung geschlossen). Sie nimmt
// c.Execute statt c.Login: Login faltet die Antwort des Servers in errors.New(info), und am
// Fehler ließe sich „Server hat abgelehnt" nicht mehr von „Server ist weg" unterscheiden. Genau
// das trennt den Fehlversuch vom Ausfall.
func meldeBeiIMAPAn(ctx context.Context, c *client.Client, verbindung net.Conn, email, passwort string) error {
	ergebnis := make(chan error, 1)
	go func() {
		status, err := c.Execute(&commands.Login{Username: email, Password: passwort}, nil)
		switch {
		case err != nil:
			ergebnis <- err
		case status == nil:
			ergebnis <- errors.New("imap: keine Status-Antwort auf LOGIN")
		case status.Type == imap.StatusRespOk:
			c.SetState(imap.AuthenticatedState, nil)
			ergebnis <- nil
		default:
			ergebnis <- fmt.Errorf("%w: %s", errAnmeldungAbgelehnt, status.Info)
		}
	}()

	select {
	case err := <-ergebnis:
		return err
	case <-ctx.Done():
		// Die Verbindung schließen, damit die Goroutine aus c.Execute zurückkehrt, und ihr
		// Ergebnis abholen, damit sie enden kann. Sonst bliebe bei jeder abgelaufenen Frist eine
		// Goroutine hängen.
		closeutil.Schliesse(verbindung, imapVerbindung)
		<-ergebnis
		return fmt.Errorf("zeitüberschreitung beim login")
	}
}

// istLokaleUmgebung meldet, ob APP_ENV eine Entwicklungs- oder Testumgebung bezeichnet. Nur dort
// darf die Attrappe des Mailservers (IMAP_HOST=mock) greifen.
func istLokaleUmgebung() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))) {
	case "local", "development", "test":
		return true
	default:
		return false
	}
}

// errMailserverNichtErreichbar trennt den Ausfall des Mailservers vom falschen Passwort. Wer bei
// einem Ausfall sein richtiges Passwort erneut probiert, sperrte sich sonst nach fünf Versuchen
// für 15 Minuten aus, und ein Ausfall des Servers sperrte das ganze Kollegium aus. Die Tür der
// Anmeldung antwortet darauf mit 503 und zählt keinen Fehlversuch.
var errMailserverNichtErreichbar = errors.New("mailserver nicht erreichbar")

// meldungMailserverNichtErreichbar ist der Satz, mit dem Anmeldung und Entsperren auf einen
// Ausfall des Mailservers antworten.
const meldungMailserverNichtErreichbar = "Der Mailserver ist nicht erreichbar. Die Anmeldung geht gerade nicht, bitte später erneut versuchen."

// PruefeIMAPKonfiguration prüft IMAP_HOST beim Start des Servers, damit eine unbrauchbare oder
// gefährliche Konfiguration der Anmeldung sofort auffällt und nicht erst bei der ersten
// Anmeldung.
//
// Abgelehnt werden zwei Fälle:
//
//   - leer: Ohne Mailserver ist keine Anmeldung möglich. Geraten wird keiner, sonst spräche ein
//     „go run" ohne Umgebung den echten Mailserver der Schule an, mit echten Anmeldeversuchen.
//   - "mock" außerhalb von APP_ENV=local/development/test: Die Attrappe nimmt jedes Passwort für
//     jede in benutzer eingetragene E-Mail-Adresse an. In einer Umgebung, die sich production
//     nennt, ist das keine Konfiguration, sondern eine offene Tür.
func PruefeIMAPKonfiguration() error {
	host := strings.TrimSpace(os.Getenv("IMAP_HOST"))
	if host == "" {
		return errors.New("IMAP_HOST ist nicht gesetzt — ohne Mailserver ist keine Anmeldung möglich (lokal: IMAP_HOST=mock mit APP_ENV=local)")
	}
	if host == "mock" && !istLokaleUmgebung() {
		return fmt.Errorf("IMAP_HOST=mock nimmt jedes Passwort an — auf einem Server den Mailserver der Schule eintragen und APP_ENV nicht ändern; mock gilt nur auf dem Entwicklungsrechner zusammen mit APP_ENV=local (aktuell: APP_ENV=%q)", os.Getenv("APP_ENV"))
	}
	return nil
}

// imapOhneServer entscheidet die Fälle, in denen kein Mailserver gefragt wird. Ohne Host wird
// abgelehnt statt geraten (siehe PruefeIMAPKonfiguration). Die Attrappe (mock) nimmt jedes
// Passwort an und gilt nur in der lokalen Entwicklung; die Prüfung hier ist die zweite Schranke
// hinter der beim Start, falls die Variable zur Laufzeit gesetzt wird.
func imapOhneServer(host string) (entschieden bool, err error) {
	if host == "" {
		slog.Error("IMAP_HOST ist nicht gesetzt — Anmeldung wird abgelehnt")
		return true, fmt.Errorf("anmeldung fehlgeschlagen")
	}
	if host != "mock" {
		return false, nil
	}
	if !istLokaleUmgebung() {
		slog.Error("IMAP_HOST=mock außerhalb der lokalen Entwicklung — Anmeldung wird abgelehnt",
			"app_env", os.Getenv("APP_ENV"))
		return true, fmt.Errorf("anmeldung fehlgeschlagen")
	}
	slog.Warn("⚠️  IMAP MOCK-MODUS AKTIV: Jedes Passwort wird akzeptiert! NUR für lokale Entwicklung verwenden!")
	return true, nil
}

// imapVerbindungsdaten baut Adresse und Einstellungen für TLS zum Mailserver. Ein Port am
// Hostnamen (alte Konfiguration, etwa :143) wird verworfen; es gilt IMAP_PORT, sonst 993.
func (a *Anmeldung) imapVerbindungsdaten(host string) (string, *tls.Config) {
	if strings.Contains(host, ":") {
		teile := strings.Split(host, ":")
		host = teile[0]
	}

	port := os.Getenv("IMAP_PORT")
	if port == "" {
		port = "993"
	}
	adresse := fmt.Sprintf("%s:%s", host, port)

	tlsKonfig := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	if a.imapTLS != nil {
		a.imapTLS(tlsKonfig)
	}
	return adresse, tlsKonfig
}

// pruefeIMAPAnmeldung prüft E-Mail-Adresse und Passwort am Mailserver der Schule, über TLS ab
// der ersten Zeile (Port 993, sofern IMAP_PORT nichts anderes sagt). ctx kommt von der Tür der
// Anmeldung; die eigene Frist imapFrist liegt darunter.
func (a *Anmeldung) pruefeIMAPAnmeldung(ctx context.Context, email, passwort string) error {
	host := strings.TrimSpace(os.Getenv("IMAP_HOST"))
	if entschieden, err := imapOhneServer(host); entschieden {
		return err
	}

	// Ein Name ohne @ ist der Benutzername an der Schule und bekommt ihre Domain.
	if !strings.Contains(email, "@") {
		email = fmt.Sprintf("%s@philipp-reis-schule.de", email)
	}
	adresse, tlsKonfig := a.imapVerbindungsdaten(host)

	ctx, cancel := context.WithTimeout(ctx, imapFrist)
	defer cancel()

	c, verbindung, err := verbindeIMAP(ctx, adresse, tlsKonfig)
	if err != nil {
		// Ein Ausfall der Verbindung ist kein falsches Passwort: Die Tür antwortet 503 statt 401
		// mit Fehlversuch (ErrMailserverNichtErreichbar).
		slog.Error("IMAP Connection failed", "addr", adresse, "error", err)
		return errMailserverNichtErreichbar
	}
	c.Timeout = imapBefehlsfrist
	defer func() {
		if c != nil {
			if err := c.Logout(); err != nil {
				log.Printf("imap: Logout fehlgeschlagen: %v", err)
			}
		}
	}()

	if err := meldeBeiIMAPAn(ctx, c, verbindung, email, passwort); err != nil {
		slog.Warn("IMAP Login failed", "error", err)
		closeutil.Schliesse(verbindung, imapVerbindung)
		// Es entscheidet der Inhalt, nicht die Uhr: Nur eine Antwort des Servers (NO oder BAD)
		// heißt falsche Zugangsdaten. Frist, EOF, Abbruch von TLS oder fehlende Begrüßung sind
		// ein Ausfall.
		if errors.Is(err, errAnmeldungAbgelehnt) {
			return errAnmeldungAbgelehnt
		}
		return errMailserverNichtErreichbar
	}

	return nil
}
