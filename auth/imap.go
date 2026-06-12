package auth

import (
	"fmt"
	"os"

	"github.com/emersion/go-imap/client"
)

// AuthenticateIMAP connects to the IMAP server and verifies credentials.
// It uses STARTTLS on port 143 as requested by the school IT, or defaults from ENV.
func AuthenticateIMAP(email, password string) error {
	host := os.Getenv("IMAP_HOST")
	if host == "" {
		host = "imap.philipp-reis-schule.de:143"
	}

	// MOCK-MODUS für lokale Entwicklung (wie in schul-orga)
	if host == "mock" {
		return nil
	}

	c, err := client.Dial(host)
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer func() { _ = c.Logout() }()

	if err := c.StartTLS(nil); err != nil {
		return fmt.Errorf("failed to start TLS on IMAP connection: %w", err)
	}

	if err := c.Login(email, password); err != nil {
		return fmt.Errorf("IMAP authentication failed for %s: %w", email, err)
	}

	return nil
}
