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

	"github.com/emersion/go-imap/client"
)

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

func loginIMAP(ctx context.Context, c *client.Client, conn net.Conn, email, password string) error {
	loginDone := make(chan error, 1)
	go func() {
		loginDone <- c.Login(email, password)
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
func AuthenticateIMAP(email, password string) error {
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	c, conn, err := connectIMAP(ctx, addr, tlsConfig)
	if err != nil {
		// Transport-Ausfall, kein Zugangsdaten-Problem: als solcher markiert, damit der
		// Login-Handler 503 antwortet statt 401 + Fehlversuch (siehe Sentinel oben).
		slog.Error("IMAP Connection failed", "addr", addr, "error", err)
		return ErrMailserverNichtErreichbar
	}
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
		// Ein falsches Passwort beantwortet der Server SOFORT mit „NO". Läuft stattdessen
		// die Frist ab, hängt der Server — das ist ein Ausfall, kein Zugangsdaten-Fehler.
		if ctx.Err() != nil {
			return ErrMailserverNichtErreichbar
		}
		return fmt.Errorf("anmeldung fehlgeschlagen")
	}

	return nil
}
