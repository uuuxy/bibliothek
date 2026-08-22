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

// imapFrist ist die EINE Frist für Verbinden + Anmelden am Mailserver. Der Login-Handler
// muss ihr Luft lassen (loginHandlerFrist > imapFrist) — bis zum 22.08.2026 stand der
// Handler auf 10 s, AuthenticateIMAP auf 15 s mit eigenem Background-Kontext: Ein
// korrektes Login, das 11 s dauerte, scheiterte danach am DB-Lookup (Handler-ctx tot)
// und zählte als Fehlversuch (Prüfung 22.08., A7).
const imapFrist = 15 * time.Second

// imapBefehlsfrist gilt je IMAP-Kommando (Login, Logout) — go-imap kennt sonst keine
// Frist (Timeout=0), ein Server, der nach „OK LOGIN" schweigt, hing den Handler ewig.
const imapBefehlsfrist = 5 * time.Second

// imapTLSAnpassung erlaubt Tests, die TLS-Konfiguration zu verändern (selbstsignierter
// Mini-Server). In Produktion nil.
var imapTLSAnpassung func(*tls.Config)

// errAnmeldungAbgelehnt: Der Server hat geantwortet und die Zugangsdaten abgelehnt (NO/BAD).
// Nur DAS ist ein falsches Passwort; alles andere (Timeout, Verbindungsabbruch, kein
// Greeting) ist ein Ausfall. Vorher entschied die ZEIT: Fehler ohne abgelaufene Frist =
// Passwort — ein Server, der die Verbindung beim LOGIN schloss, zählte als Fehlversuch.
var errAnmeldungAbgelehnt = errors.New("anmeldung fehlgeschlagen")

func connectIMAP(ctx context.Context, addr string, tlsConfig *tls.Config) (*client.Client, net.Conn, error) {
	// Enforce 10s timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Use tls.Dialer to support Context
	dialer := &net.Dialer{}
	tlsDialer := &tls.Dialer{
		NetDialer: dialer,
		Config:    tlsConfig,
	}

	conn, err := tlsDialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, nil, fmt.Errorf("zeitüberschreitung bei verbindung")
		}
		if ctx.Err() == context.Canceled {
			return nil, nil, fmt.Errorf("timeout bei verbindung zum server")
		}
		return nil, nil, err
	}

	// Watch for context cancellation during client.New which can block on I/O
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			closeutil.LogClose(conn, imapConnSource)
		case <-done:
		}
	}()

	c, err := client.New(conn)
	if err != nil {
		closeutil.LogClose(conn, imapConnSource)
		if ctx.Err() == context.DeadlineExceeded {
			return nil, nil, fmt.Errorf("zeitüberschreitung bei verbindung")
		}
		if ctx.Err() == context.Canceled {
			return nil, nil, fmt.Errorf("timeout bei verbindung zum server")
		}
		return nil, nil, err
	}

	return c, conn, nil
}

// loginIMAP sendet LOGIN und liefert errAnmeldungAbgelehnt, wenn der Server mit NO/BAD
// antwortet, sonst den Transportfehler (Timeout, EOF, Verbindung zu). Bewusst über
// c.Execute statt c.Login: Login() faltet die Status-Antwort in errors.New(info) — aus dem
// Fehler ließe sich „Server hat NEIN gesagt" nicht mehr von „Server ist weg" unterscheiden,
// und genau diese Unterscheidung trennt Fehlversuch von Ausfall (Prüfung 22.08.2026, A7).
func loginIMAP(ctx context.Context, c *client.Client, conn net.Conn, email, password string) error {
	loginDone := make(chan error, 1)
	go func() {
		status, err := c.Execute(&commands.Login{Username: email, Password: password}, nil)
		switch {
		case err != nil:
			loginDone <- err
		case status == nil:
			loginDone <- errors.New("imap: keine Status-Antwort auf LOGIN")
		case status.Type == imap.StatusRespOk:
			c.SetState(imap.AuthenticatedState, nil)
			loginDone <- nil
		default:
			loginDone <- fmt.Errorf("%w: %s", errAnmeldungAbgelehnt, status.Info)
		}
	}()

	select {
	case err := <-loginDone:
		return err
	case <-ctx.Done():
		// Force-close the connection to unblock the goroutine stuck in c.Login()
		// This prevents a goroutine leak on every timeout.
		closeutil.LogClose(conn, imapConnSource)
		// Drain the result so the goroutine can exit
		<-loginDone
		return fmt.Errorf("zeitüberschreitung beim login")
	}
}

// istLokaleUmgebung meldet, ob APP_ENV eine Entwicklungs- oder Testumgebung
// bezeichnet. Nur dort darf der IMAP-Mock greifen.
func istLokaleUmgebung() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))) {
	case "local", "development", "test":
		return true
	default:
		return false
	}
}

// ErrMailserverNichtErreichbar trennt den Transport-Ausfall vom falschen Passwort
// (Ausfallmatrix 20.08.2026): Vorher wurde JEDER IMAP-Fehler — Server down, Timeout,
// Firewall — dem Nutzer als „invalid email or password" gemeldet UND als Fehlversuch
// gezählt. Bei einem Mailserver-Ausfall probierte das Kollegium sein (richtiges)
// Passwort erneut und sperrte sich nach fünf Versuchen für 15 Minuten selbst aus —
// ein Serverausfall wurde zur Massen-Selbstsperre. Der Login-Handler antwortet auf
// diesen Fehler mit 503 und zählt KEINEN Fehlversuch.
//
//nolint:staticcheck // ST1005: bewusst großgeschrieben, Endnutzer-Meldung (503)
var ErrMailserverNichtErreichbar = errors.New("Mailserver ist nicht erreichbar — Anmeldung derzeit nicht möglich, bitte später erneut versuchen")

// PruefeIMAPKonfiguration validiert IMAP_HOST beim Serverstart, damit eine
// unbrauchbare oder gefährliche Anmeldekonfiguration sofort auffällt und nicht
// erst beim ersten Login-Versuch eines Nutzers.
//
// Zwei Fälle werden abgelehnt:
//
//   - leer: Vorher fiel der Code still auf imap.philipp-reis-schule.de zurück.
//     Ein "go run" ohne Umgebung sprach damit unbeabsichtigt den Produktiv-IMAP
//     der Schule an — echte Verbindungen, echte Login-Versuche.
//   - "mock" außerhalb von APP_ENV=local/development/test: Der Mock akzeptiert
//     JEDES Passwort für jede in benutzer eingetragene E-Mail. In einer Umgebung,
//     die sich production nennt, ist das keine Konfiguration, sondern eine offene Tür.
func PruefeIMAPKonfiguration() error {
	host := strings.TrimSpace(os.Getenv("IMAP_HOST"))
	if host == "" {
		return errors.New("IMAP_HOST ist nicht gesetzt — ohne Mailserver ist keine Anmeldung möglich (lokal: IMAP_HOST=mock mit APP_ENV=local)")
	}
	if host == "mock" && !istLokaleUmgebung() {
		return fmt.Errorf("IMAP_HOST=mock akzeptiert jedes Passwort und ist nur mit APP_ENV=local/development/test zulässig (aktuell: APP_ENV=%q)", os.Getenv("APP_ENV"))
	}
	return nil
}

// AuthenticateIMAP connects to the IMAP server and verifies credentials.
// It uses implicit TLS on port 993 as successfully implemented in schul-orga.
// ctx kommt vom Aufrufer (Login-Handler); die eigene Frist imapFrist liegt darunter.
func AuthenticateIMAP(ctx context.Context, email, password string) error {
	host := strings.TrimSpace(os.Getenv("IMAP_HOST"))

	// Kein stiller Rückfall mehr auf den Schulserver: Ohne konfigurierten Host
	// wird nicht geraten, sondern abgelehnt. Siehe PruefeIMAPKonfiguration.
	if host == "" {
		slog.Error("IMAP_HOST ist nicht gesetzt — Anmeldung wird abgelehnt")
		return fmt.Errorf("anmeldung fehlgeschlagen")
	}

	// MOCK-MODUS für lokale Entwicklung — zweite Schranke hinter dem Startup-Check
	// in PruefeIMAPKonfiguration, falls die Variable zur Laufzeit gesetzt wird.
	if host == "mock" {
		if !istLokaleUmgebung() {
			slog.Error("IMAP_HOST=mock außerhalb der lokalen Entwicklung — Anmeldung wird abgelehnt",
				"app_env", os.Getenv("APP_ENV"))
			return fmt.Errorf("anmeldung fehlgeschlagen")
		}
		slog.Warn("⚠️  IMAP MOCK-MODUS AKTIV: Jedes Passwort wird akzeptiert! NUR für lokale Entwicklung verwenden!")
		return nil
	}

	// Remove port from host if it was provided via env/old config (e.g. :143)
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		host = parts[0]
	}

	port := os.Getenv("IMAP_PORT")
	if port == "" {
		port = "993"
	}

	// Format email correctly if only username was provided (wie in schul-orga)
	if !strings.Contains(email, "@") {
		email = fmt.Sprintf("%s@philipp-reis-schule.de", email)
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	tlsConfig := &tls.Config{
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

	if imapTLSAnpassung != nil {
		imapTLSAnpassung(tlsConfig)
	}

	ctx, cancel := context.WithTimeout(ctx, imapFrist)
	defer cancel()

	c, conn, err := connectIMAP(ctx, addr, tlsConfig)
	if err != nil {
		// Transport-Ausfall, kein Zugangsdaten-Problem: als solcher markiert, damit der
		// Login-Handler 503 antwortet statt 401 + Fehlversuch (siehe Sentinel oben).
		slog.Error("IMAP Connection failed", "addr", addr, "error", err)
		return ErrMailserverNichtErreichbar
	}
	c.Timeout = imapBefehlsfrist
	defer func() {
		if c != nil {
			if err := c.Logout(); err != nil {
				log.Printf("imap: Logout fehlgeschlagen: %v", err)
			}
		}
	}()

	if err := loginIMAP(ctx, c, conn, email, password); err != nil {
		slog.Warn("IMAP Login failed", "error", err)
		closeutil.LogClose(conn, imapConnSource)
		// Inhalt statt Uhr: Nur eine Status-Antwort des Servers (NO/BAD) ist ein
		// Zugangsdaten-Fehler. Timeout, EOF, TLS-Abbruch, fehlendes Greeting = Ausfall.
		if errors.Is(err, errAnmeldungAbgelehnt) {
			return errAnmeldungAbgelehnt
		}
		return ErrMailserverNichtErreichbar
	}

	return nil
}
