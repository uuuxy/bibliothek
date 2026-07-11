package auth

import (
	"context"
	"crypto/tls"
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
			_ = conn.Close()
		case <-done:
		}
	}()

	c, err := client.New(conn)
	if err != nil {
		_ = conn.Close()
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
		_ = conn.Close()
		// Drain the result so the goroutine can exit
		<-loginDone
		return fmt.Errorf("zeitüberschreitung beim login")
	}
}

// AuthenticateIMAP connects to the IMAP server and verifies credentials.
// It uses implicit TLS on port 993 as successfully implemented in schul-orga.
func AuthenticateIMAP(email, password string) error {
	host := os.Getenv("IMAP_HOST")
	if host == "" {
		host = "imap.philipp-reis-schule.de"
	}

	// MOCK-MODUS für lokale Entwicklung
	if host == "mock" {
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
		slog.Error("IMAP Connection failed", "addr", addr, "error", err)
		return fmt.Errorf("verbindung fehlgeschlagen: %v", err)
	}
	defer func() {
		if c != nil {
			if err := c.Logout(); err != nil {
				log.Printf("imap: Logout fehlgeschlagen: %v", err)
			}
		}
		if conn != nil {
			_ = conn.Close()
		}
	}()

	if err := loginIMAP(ctx, c, conn, email, password); err != nil {
		slog.Warn("IMAP Login failed", "error", err)
		_ = conn.Close()
		return fmt.Errorf("anmeldung fehlgeschlagen")
	}

	return nil
}
